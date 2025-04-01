package mpn

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/bloom"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/decred/dcrd/lru"
	"github.com/dgraph-io/badger/v4"

	"github.com/sat20-labs/indexer/indexer/mpn/addrmgr"
	"github.com/sat20-labs/indexer/indexer/mpn/connmgr"
	"github.com/sat20-labs/indexer/indexer/mpn/peer"

	localCommon "github.com/sat20-labs/indexer/indexer/mpn/common"
	"github.com/sat20-labs/indexer/indexer/mpn/mempool"
	"github.com/sat20-labs/indexer/indexer/mpn/netsync"

	"github.com/sat20-labs/indexer/common"
)

const (
	// defaultServices describes the default services that are supported by
	// the MemPoolNode.
	// defaultServices = wire.SFNodeNetwork | wire.SFNodeNetworkLimited |
	// 	wire.SFNodeBloom | wire.SFNodeWitness | wire.SFNodeCF
	defaultServices = wire.SFNodeNetworkLimited |
		wire.SFNodeBloom | wire.SFNodeWitness | wire.SFNodeCF

	// defaultRequiredServices describes the default services that are
	// required to be supported by outbound peers.
	defaultRequiredServices = wire.SFNodeNetwork

	// defaultTargetOutbound is the default number of outbound peers to target.
	defaultTargetOutbound = 8

	// connectionRetryInterval is the base amount of time to wait in between
	// retries when connecting to persistent peers.  It is adjusted by the
	// number of retries such that there is a retry backoff.
	connectionRetryInterval = time.Second * 5
)

var (
	// userAgentName is the user agent name and is used to help identify
	// ourselves to other bitcoin peers.
	userAgentName = "sat20"

	// userAgentVersion is the user agent version and is used to help
	// identify ourselves to other bitcoin peers.
	userAgentVersion = fmt.Sprintf("%d.%d.%d", appMajor, appMinor, appPatch)
)

// zeroHash is the zero value hash (all zeros).  It is defined as a convenience.
var zeroHash chainhash.Hash

// onionAddr implements the net.Addr interface and represents a tor address.
type onionAddr struct {
	addr string
}

// String returns the onion address.
//
// This is part of the net.Addr interface.
func (oa *onionAddr) String() string {
	return oa.addr
}

// Network returns "onion".
//
// This is part of the net.Addr interface.
func (oa *onionAddr) Network() string {
	return "onion"
}

// Ensure onionAddr implements the net.Addr interface.
var _ net.Addr = (*onionAddr)(nil)

// simpleAddr implements the net.Addr interface with two struct fields
type simpleAddr struct {
	net, addr string
}

// String returns the address.
//
// This is part of the net.Addr interface.
func (a simpleAddr) String() string {
	return a.addr
}

// Network returns the network.
//
// This is part of the net.Addr interface.
func (a simpleAddr) Network() string {
	return a.net
}

// Ensure simpleAddr implements the net.Addr interface.
var _ net.Addr = simpleAddr{}

// broadcastMsg provides the ability to house a bitcoin message to be broadcast
// to all connected peers except specified excluded peers.
type broadcastMsg struct {
	message      wire.Message
	excludePeers []*serverPeer
}

// broadcastInventoryAdd is a type used to declare that the InvVect it contains
// needs to be added to the rebroadcast map
type broadcastInventoryAdd relayMsg

// broadcastInventoryDel is a type used to declare that the InvVect it contains
// needs to be removed from the rebroadcast map
type broadcastInventoryDel *wire.InvVect

// relayMsg packages an inventory vector along with the newly discovered
// inventory so the relay has access to that information.
type relayMsg struct {
	invVect *wire.InvVect
	data    interface{}
}

// updatePeerHeightsMsg is a message sent from the blockmanager to the MemPoolNode
// after a new block has been accepted. The purpose of the message is to update
// the heights of peers that were known to announce the block before we
// connected it to the main chain or recognized it as an orphan. With these
// updates, peer heights will be kept up to date, allowing for fresh data when
// selecting sync peer candidacy.
type updatePeerHeightsMsg struct {
	newHash    *chainhash.Hash
	newHeight  int32
	originPeer *peer.Peer
}

// peerState maintains state of inbound, persistent, outbound peers as well
// as banned peers and outbound groups.
type peerState struct {
	inboundPeers    map[int32]*serverPeer
	outboundPeers   map[int32]*serverPeer
	persistentPeers map[int32]*serverPeer
	banned          map[string]time.Time
	outboundGroups  map[string]int
}

// Count returns the count of all known peers.
func (ps *peerState) Count() int {
	return len(ps.inboundPeers) + len(ps.outboundPeers) +
		len(ps.persistentPeers)
}

// forAllOutboundPeers is a helper function that runs closure on all outbound
// peers known to peerState.
func (ps *peerState) forAllOutboundPeers(closure func(sp *serverPeer)) {
	for _, e := range ps.outboundPeers {
		closure(e)
	}
	for _, e := range ps.persistentPeers {
		closure(e)
	}
}

// forAllPeers is a helper function that runs closure on all peers known to
// peerState.
func (ps *peerState) forAllPeers(closure func(sp *serverPeer)) {
	for _, e := range ps.inboundPeers {
		closure(e)
	}
	ps.forAllOutboundPeers(closure)
}

// cfHeaderKV is a tuple of a filter header and its associated block hash. The
// struct is used to cache cfcheckpt responses.
type cfHeaderKV struct {
	blockHash    chainhash.Hash
	filterHeader chainhash.Hash
}

// MemPoolNode provides a bitcoin MemPoolNode for handling communications to and from
// bitcoin peers.
type MemPoolNode struct {
	// The following variables must only be used atomically.
	// Putting the uint64s first makes them 64-bit aligned for 32-bit systems.
	bytesReceived uint64 // Total bytes received from all peers since start.
	bytesSent     uint64 // Total bytes sent by all peers since start.
	started       int32
	shutdown      int32
	shutdownSched int32
	startupTime   int64

	chainParams          *chaincfg.Params
	addrManager          *addrmgr.AddrManager
	connManager          *connmgr.ConnManager
	sigCache             *txscript.SigCache
	hashCache            *txscript.HashCache
	syncManager          *netsync.SyncManager
	indexer                localCommon.IndexManager
	txMemPool            *mempool.TxPool
	modifyRebroadcastInv chan interface{}
	newPeers             chan *serverPeer
	donePeers            chan *serverPeer
	banPeers             chan *serverPeer
	query                chan interface{}
	relayInv             chan relayMsg
	broadcast            chan broadcastMsg
	peerHeightsUpdate    chan updatePeerHeightsMsg
	wg                   sync.WaitGroup
	quit                 chan struct{}
	nat                  NAT
	db                   *badger.DB
	//timeSource           localCommon.MedianTimeSource
	services             wire.ServiceFlag

	// The following fields are used for optional indexes.  They will be nil
	// if the associated index is not enabled.  These fields are set during
	// initial creation of the MemPoolNode and never changed afterwards, so they
	// do not need to be protected for concurrent access.
	// txIndex   *indexers.TxIndex
	// addrIndex *indexers.AddrIndex
	// cfIndex   *indexers.CfIndex

	// The fee estimator keeps track of how long transactions are left in
	// the mempool before they are mined into blocks.
	feeEstimator *mempool.FeeEstimator

	// cfCheckptCaches stores a cached slice of filter headers for cfcheckpt
	// messages for each filter type.
	cfCheckptCaches    map[wire.FilterType][]cfHeaderKV
	cfCheckptCachesMtx sync.RWMutex

	// agentBlacklist is a list of blacklisted substrings by which to filter
	// user agents.
	agentBlacklist []string

	// agentWhitelist is a list of whitelisted user agent substrings, no
	// whitelisting will be applied if the list is empty or nil.
	agentWhitelist []string
}

// serverPeer extends the peer to maintain state shared by the MemPoolNode and
// the blockmanager.
type serverPeer struct {
	// The following variables must only be used atomically
	feeFilter int64

	*peer.Peer

	connReq        *connmgr.ConnReq
	MemPoolNode    *MemPoolNode
	persistent     bool
	continueHash   *chainhash.Hash
	relayMtx       sync.Mutex
	disableRelayTx bool
	sentAddrs      bool
	isWhitelisted  bool
	filter         *bloom.Filter
	addressesMtx   sync.RWMutex
	knownAddresses lru.Cache
	banScore       connmgr.DynamicBanScore
	quit           chan struct{}
	// The following chans are used to sync blockmanager and MemPoolNode.
	txProcessed    chan struct{}
	blockProcessed chan struct{}
}

// newServerPeer returns a new serverPeer instance. The peer needs to be set by
// the caller.
func newServerPeer(s *MemPoolNode, isPersistent bool) *serverPeer {
	return &serverPeer{
		MemPoolNode:    s,
		persistent:     isPersistent,
		filter:         bloom.LoadFilter(nil),
		knownAddresses: lru.NewCache(5000),
		quit:           make(chan struct{}),
		txProcessed:    make(chan struct{}, 1),
		blockProcessed: make(chan struct{}, 1),
	}
}

// newestBlock returns the current best block hash and height using the format
// required by the configuration for the peer package.
func (sp *serverPeer) newestBlock() (*chainhash.Hash, int32, error) {
	best := sp.MemPoolNode.indexer.BestSnapshot()
	return &best.Hash, best.Height, nil
}

// addKnownAddresses adds the given addresses to the set of known addresses to
// the peer to prevent sending duplicate addresses.
func (sp *serverPeer) addKnownAddresses(addresses []*wire.NetAddressV2) {
	sp.addressesMtx.Lock()
	for _, na := range addresses {
		sp.knownAddresses.Add(addrmgr.NetAddressKey(na))
	}
	sp.addressesMtx.Unlock()
}

// addressKnown true if the given address is already known to the peer.
func (sp *serverPeer) addressKnown(na *wire.NetAddressV2) bool {
	sp.addressesMtx.RLock()
	exists := sp.knownAddresses.Contains(addrmgr.NetAddressKey(na))
	sp.addressesMtx.RUnlock()
	return exists
}

// setDisableRelayTx toggles relaying of transactions for the given peer.
// It is safe for concurrent access.
func (sp *serverPeer) setDisableRelayTx(disable bool) {
	sp.relayMtx.Lock()
	sp.disableRelayTx = disable
	sp.relayMtx.Unlock()
}

// relayTxDisabled returns whether or not relaying of transactions for the given
// peer is disabled.
// It is safe for concurrent access.
func (sp *serverPeer) relayTxDisabled() bool {
	sp.relayMtx.Lock()
	isDisabled := sp.disableRelayTx
	sp.relayMtx.Unlock()

	return isDisabled
}

// pushAddrMsg sends a legacy addr message to the connected peer using the
// provided addresses.
func (sp *serverPeer) pushAddrMsg(addresses []*wire.NetAddressV2) {
	if sp.WantsAddrV2() {
		// If the peer supports addrv2, we'll be pushing an addrv2
		// message instead. The logic is otherwise identical to the
		// addr case below.
		addrs := make([]*wire.NetAddressV2, 0, len(addresses))
		for _, addr := range addresses {
			// Filter addresses already known to the peer.
			if sp.addressKnown(addr) {
				continue
			}

			addrs = append(addrs, addr)
		}

		known, err := sp.PushAddrV2Msg(addrs)
		if err != nil {
			common.Log.Errorf("Can't push addrv2 message to %s: %v",
				sp.Peer, err)
			sp.Disconnect()
			return
		}

		// Add the final set of addresses sent to the set the peer
		// knows of.
		sp.addKnownAddresses(known)
		return
	}

	addrs := make([]*wire.NetAddress, 0, len(addresses))
	for _, addr := range addresses {
		// Filter addresses already known to the peer.
		if sp.addressKnown(addr) {
			continue
		}

		// Must skip the V3 addresses for legacy ADDR messages.
		if addr.IsTorV3() {
			continue
		}

		// Convert the NetAddressV2 to a legacy address.
		addrs = append(addrs, addr.ToLegacy())
	}

	known, err := sp.PushAddrMsg(addrs)
	if err != nil {
		common.Log.Errorf(
			"Can't push address message to %s: %v", sp.Peer, err,
		)
		sp.Disconnect()
		return
	}

	// Convert all of the known addresses to NetAddressV2 to add them to
	// the set of known addresses.
	knownAddrs := make([]*wire.NetAddressV2, 0, len(known))
	for _, knownAddr := range known {
		currentKna := wire.NetAddressV2FromBytes(
			knownAddr.Timestamp, knownAddr.Services,
			knownAddr.IP, knownAddr.Port,
		)
		knownAddrs = append(knownAddrs, currentKna)
	}
	sp.addKnownAddresses(knownAddrs)
}

// addBanScore increases the persistent and decaying ban score fields by the
// values passed as parameters. If the resulting score exceeds half of the ban
// threshold, a warning is logged including the reason provided. Further, if
// the score is above the ban threshold, the peer will be banned and
// disconnected.
func (sp *serverPeer) addBanScore(persistent, transient uint32, reason string) bool {
	// No warning is logged and no score is calculated if banning is disabled.
	if _cfg.DisableBanning {
		return false
	}
	if sp.isWhitelisted {
		common.Log.Debugf("Misbehaving whitelisted peer %s: %s", sp, reason)
		return false
	}

	warnThreshold := _cfg.BanThreshold >> 1
	if transient == 0 && persistent == 0 {
		// The score is not being increased, but a warning message is still
		// logged if the score is above the warn threshold.
		score := sp.banScore.Int()
		if score > warnThreshold {
			common.Log.Warnf("Misbehaving peer %s: %s -- ban score is %d, "+
				"it was not increased this time", sp, reason, score)
		}
		return false
	}
	score := sp.banScore.Increase(persistent, transient)
	if score > warnThreshold {
		common.Log.Warnf("Misbehaving peer %s: %s -- ban score increased to %d",
			sp, reason, score)
		if score > _cfg.BanThreshold {
			common.Log.Warnf("Misbehaving peer %s -- banning and disconnecting",
				sp)
			sp.MemPoolNode.BanPeer(sp)
			sp.Disconnect()
			return true
		}
	}
	return false
}

// hasServices returns whether or not the provided advertised service flags have
// all of the provided desired service flags set.
func hasServices(advertised, desired wire.ServiceFlag) bool {
	return advertised&desired == desired
}

// OnVersion is invoked when a peer receives a version bitcoin message
// and is used to negotiate the protocol version details as well as kick start
// the communications.
func (sp *serverPeer) OnVersion(_ *peer.Peer, msg *wire.MsgVersion) *wire.MsgReject {
	// Update the address manager with the advertised services for outbound
	// connections in case they have changed.  This is not done for inbound
	// connections to help prevent malicious behavior and is skipped when
	// running on the simulation test network since it is only intended to
	// connect to specified peers and actively avoids advertising and
	// connecting to discovered peers.
	//
	// NOTE: This is done before rejecting peers that are too old to ensure
	// it is updated regardless in the case a new minimum protocol version is
	// enforced and the remote node has not upgraded yet.
	isInbound := sp.Inbound()
	remoteAddr := sp.NA()
	addrManager := sp.MemPoolNode.addrManager
	if !_cfg.SimNet && !isInbound {
		addrManager.SetServices(remoteAddr, msg.Services)
	}

	// Ignore peers that have a protocol version that is too old.  The peer
	// negotiation logic will disconnect it after this callback returns.
	if msg.ProtocolVersion < int32(peer.MinAcceptableProtocolVersion) {
		return nil
	}

	// Reject outbound peers that are not full nodes.
	wantServices := wire.SFNodeNetwork
	if !isInbound && !hasServices(msg.Services, wantServices) {
		missingServices := wantServices & ^msg.Services
		common.Log.Debugf("Rejecting peer %s with services %v due to not "+
			"providing desired services %v", sp.Peer, msg.Services,
			missingServices)
		reason := fmt.Sprintf("required services %#x not offered",
			uint64(missingServices))
		return wire.NewMsgReject(msg.Command(), wire.RejectNonstandard, reason)
	}

	if !_cfg.SimNet && !isInbound {
		// After soft-fork activation, only make outbound
		// connection to peers if they flag that they're segwit
		// enabled.
		segwitActive, err := sp.MemPoolNode.indexer.IsDeploymentActive(chaincfg.DeploymentSegwit)
		if err != nil {
			common.Log.Errorf("Unable to query for segwit soft-fork state: %v",
				err)
			return nil
		}

		if segwitActive && !sp.IsWitnessEnabled() {
			common.Log.Infof("Disconnecting non-segwit peer %v, isn't segwit "+
				"enabled and we need more segwit enabled peers", sp)
			sp.Disconnect()
			return nil
		}
	}

	// Add the remote peer time as a sample for creating an offset against
	// the local clock to keep the network time in sync.
	//sp.MemPoolNode.timeSource.AddTimeSample(sp.Addr(), msg.Timestamp)

	// Choose whether or not to relay transactions before a filter command
	// is received.
	sp.setDisableRelayTx(msg.DisableRelayTx)

	return nil
}

// OnVerAck is invoked when a peer receives a verack bitcoin message and is used
// to kick start communication with them.
func (sp *serverPeer) OnVerAck(_ *peer.Peer, _ *wire.MsgVerAck) {
	sp.MemPoolNode.AddPeer(sp)
}

// OnMemPool is invoked when a peer receives a mempool bitcoin message.
// It creates and sends an inventory message with the contents of the memory
// pool up to the maximum inventory allowed per message.  When the peer has a
// bloom filter loaded, the contents are filtered accordingly.
func (sp *serverPeer) OnMemPool(_ *peer.Peer, msg *wire.MsgMemPool) {
	// Only allow mempool requests if the MemPoolNode has bloom filtering
	// enabled.
	if sp.MemPoolNode.services&wire.SFNodeBloom != wire.SFNodeBloom {
		common.Log.Debugf("peer %v sent mempool request with bloom "+
			"filtering disabled -- disconnecting", sp)
		sp.Disconnect()
		return
	}

	// A decaying ban score increase is applied to prevent flooding.
	// The ban score accumulates and passes the ban threshold if a burst of
	// mempool messages comes from a peer. The score decays each minute to
	// half of its value.
	if sp.addBanScore(0, 33, "mempool") {
		return
	}

	// Generate inventory message with the available transactions in the
	// transaction memory pool.  Limit it to the max allowed inventory
	// per message.  The NewMsgInvSizeHint function automatically limits
	// the passed hint to the maximum allowed, so it's safe to pass it
	// without double checking it here.
	txMemPool := sp.MemPoolNode.txMemPool
	txDescs := txMemPool.TxDescs()
	invMsg := wire.NewMsgInvSizeHint(uint(len(txDescs)))

	for _, txDesc := range txDescs {
		// Either add all transactions when there is no bloom filter,
		// or only the transactions that match the filter when there is
		// one.
		if !sp.filter.IsLoaded() || sp.filter.MatchTxAndUpdate(txDesc.Tx) {
			iv := wire.NewInvVect(wire.InvTypeTx, txDesc.Tx.Hash())
			invMsg.AddInvVect(iv)
			if len(invMsg.InvList)+1 > wire.MaxInvPerMsg {
				break
			}
		}
	}

	// Send the inventory message if there is anything to send.
	if len(invMsg.InvList) > 0 {
		sp.QueueMessage(invMsg, nil)
	}
}

// OnTx is invoked when a peer receives a tx bitcoin message.  It blocks
// until the bitcoin transaction has been fully processed.  Unlock the block
// handler this does not serialize all transactions through a single thread
// transactions don't rely on the previous one in a linear fashion like blocks.
func (sp *serverPeer) OnTx(_ *peer.Peer, msg *wire.MsgTx) {
	if _cfg.BlocksOnly {
		common.Log.Tracef("Ignoring tx %v from %v - blocksonly enabled",
			msg.TxHash(), sp)
		return
	}

	// Add the transaction to the known inventory for the peer.
	// Convert the raw MsgTx to a btcutil.Tx which provides some convenience
	// methods and things such as hash caching.
	tx := btcutil.NewTx(msg)
	iv := wire.NewInvVect(wire.InvTypeTx, tx.Hash())
	sp.AddKnownInventory(iv)

	// Queue the transaction up to be handled by the sync manager and
	// intentionally block further receives until the transaction is fully
	// processed and known good or bad.  This helps prevent a malicious peer
	// from queuing up a bunch of bad transactions before disconnecting (or
	// being disconnected) and wasting memory.
	sp.MemPoolNode.syncManager.QueueTx(tx, sp.Peer, sp.txProcessed)
	<-sp.txProcessed
}

// OnBlock is invoked when a peer receives a block bitcoin message.  It
// blocks until the bitcoin block has been fully processed.
func (sp *serverPeer) OnBlock(_ *peer.Peer, msg *wire.MsgBlock, buf []byte) {
	// Convert the raw MsgBlock to a btcutil.Block which provides some
	// convenience methods and things such as hash caching.
	block := btcutil.NewBlockFromBlockAndBytes(msg, buf)

	// Add the block to the known inventory for the peer.
	iv := wire.NewInvVect(wire.InvTypeBlock, block.Hash())
	sp.AddKnownInventory(iv)

	// Queue the block up to be handled by the block
	// manager and intentionally block further receives
	// until the bitcoin block is fully processed and known
	// good or bad.  This helps prevent a malicious peer
	// from queuing up a bunch of bad blocks before
	// disconnecting (or being disconnected) and wasting
	// memory.  Additionally, this behavior is depended on
	// by at least the block acceptance test tool as the
	// reference implementation processes blocks in the same
	// thread and therefore blocks further messages until
	// the bitcoin block has been fully processed.
	sp.MemPoolNode.syncManager.QueueBlock(block, sp.Peer, sp.blockProcessed)
	<-sp.blockProcessed
}

// OnInv is invoked when a peer receives an inv bitcoin message and is
// used to examine the inventory being advertised by the remote peer and react
// accordingly.  We pass the message down to blockmanager which will call
// QueueMessage with any appropriate responses.
func (sp *serverPeer) OnInv(_ *peer.Peer, msg *wire.MsgInv) {
	if !_cfg.BlocksOnly {
		if len(msg.InvList) > 0 {
			sp.MemPoolNode.syncManager.QueueInv(msg, sp.Peer)
		}
		return
	}

	newInv := wire.NewMsgInvSizeHint(uint(len(msg.InvList)))
	for _, invVect := range msg.InvList {
		if invVect.Type == wire.InvTypeTx {
			common.Log.Tracef("Ignoring tx %v in inv from %v -- "+
				"blocksonly enabled", invVect.Hash, sp)
			if sp.ProtocolVersion() >= wire.BIP0037Version {
				common.Log.Infof("Peer %v is announcing "+
					"transactions -- disconnecting", sp)
				sp.Disconnect()
				return
			}
			continue
		}
		err := newInv.AddInvVect(invVect)
		if err != nil {
			common.Log.Errorf("Failed to add inventory vector: %v", err)
			break
		}
	}

	if len(newInv.InvList) > 0 {
		sp.MemPoolNode.syncManager.QueueInv(newInv, sp.Peer)
	}
}

// OnHeaders is invoked when a peer receives a headers bitcoin
// message.  The message is passed down to the sync manager.
func (sp *serverPeer) OnHeaders(_ *peer.Peer, msg *wire.MsgHeaders) {
	sp.MemPoolNode.syncManager.QueueHeaders(msg, sp.Peer)
}

// handleGetData is invoked when a peer receives a getdata bitcoin message and
// is used to deliver block and transaction information.
func (sp *serverPeer) OnGetData(_ *peer.Peer, msg *wire.MsgGetData) {
	numAdded := 0
	notFound := wire.NewMsgNotFound()

	length := len(msg.InvList)
	// A decaying ban score increase is applied to prevent exhausting resources
	// with unusually large inventory queries.
	// Requesting more than the maximum inventory vector length within a short
	// period of time yields a score above the default ban threshold. Sustained
	// bursts of small requests are not penalized as that would potentially ban
	// peers performing IBD.
	// This incremental score decays each minute to half of its value.
	if sp.addBanScore(0, uint32(length)*99/wire.MaxInvPerMsg, "getdata") {
		return
	}

	// We wait on this wait channel periodically to prevent queuing
	// far more data than we can send in a reasonable time, wasting memory.
	// The waiting occurs after the database fetch for the next one to
	// provide a little pipelining.
	var waitChan chan struct{}
	doneChan := make(chan struct{}, 1)

	for i, iv := range msg.InvList {
		var c chan struct{}
		// If this will be the last message we send.
		if i == length-1 && len(notFound.InvList) == 0 {
			c = doneChan
		} else if (i+1)%3 == 0 {
			// Buffered so as to not make the send goroutine block.
			c = make(chan struct{}, 1)
		}
		var err error
		switch iv.Type {
		case wire.InvTypeWitnessTx:
			err = sp.MemPoolNode.pushTxMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeTx:
			err = sp.MemPoolNode.pushTxMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		case wire.InvTypeWitnessBlock:
			err = sp.MemPoolNode.pushBlockMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeBlock:
			err = sp.MemPoolNode.pushBlockMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		case wire.InvTypeFilteredWitnessBlock:
			err = sp.MemPoolNode.pushMerkleBlockMsg(sp, &iv.Hash, c, waitChan, wire.WitnessEncoding)
		case wire.InvTypeFilteredBlock:
			err = sp.MemPoolNode.pushMerkleBlockMsg(sp, &iv.Hash, c, waitChan, wire.BaseEncoding)
		default:
			common.Log.Warnf("Unknown type in inventory request %d",
				iv.Type)
			continue
		}
		if err != nil {
			notFound.AddInvVect(iv)

			// When there is a failure fetching the final entry
			// and the done channel was sent in due to there
			// being no outstanding not found inventory, consume
			// it here because there is now not found inventory
			// that will use the channel momentarily.
			if i == len(msg.InvList)-1 && c != nil {
				<-c
			}
		}
		numAdded++
		waitChan = c
	}
	if len(notFound.InvList) != 0 {
		sp.QueueMessage(notFound, doneChan)
	}

	// Wait for messages to be sent. We can send quite a lot of data at this
	// point and this will keep the peer busy for a decent amount of time.
	// We don't process anything else by them in this time so that we
	// have an idea of when we should hear back from them - else the idle
	// timeout could fire when we were only half done sending the blocks.
	if numAdded > 0 {
		<-doneChan
	}
}

// OnGetBlocks is invoked when a peer receives a getblocks bitcoin
// message.
func (sp *serverPeer) OnGetBlocks(_ *peer.Peer, msg *wire.MsgGetBlocks) {
	// Find the most recent known block in the best chain based on the block
	// locator and fetch all of the block hashes after it until either
	// wire.MaxBlocksPerMsg have been fetched or the provided stop hash is
	// encountered.
	//
	// Use the block after the genesis block if no other blocks in the
	// provided locator are known.  This does mean the client will start
	// over with the genesis block if unknown block locators are provided.
	//
	// This mirrors the behavior in the reference implementation.
	chain := sp.MemPoolNode.indexer
	hashList := chain.LocateBlocks(msg.BlockLocatorHashes, &msg.HashStop,
		wire.MaxBlocksPerMsg)

	// Generate inventory message.
	invMsg := wire.NewMsgInv()
	for i := range hashList {
		iv := wire.NewInvVect(wire.InvTypeBlock, &hashList[i])
		invMsg.AddInvVect(iv)
	}

	// Send the inventory message if there is anything to send.
	if len(invMsg.InvList) > 0 {
		invListLen := len(invMsg.InvList)
		if invListLen == wire.MaxBlocksPerMsg {
			// Intentionally use a copy of the final hash so there
			// is not a reference into the inventory slice which
			// would prevent the entire slice from being eligible
			// for GC as soon as it's sent.
			continueHash := invMsg.InvList[invListLen-1].Hash
			sp.continueHash = &continueHash
		}
		//sp.QueueMessage(invMsg, nil)
	}
	// 空白消息也要应答，不然会导致对端以为连接中断
	sp.QueueMessage(invMsg, nil)
}

// OnGetHeaders is invoked when a peer receives a getheaders bitcoin
// message.
func (sp *serverPeer) OnGetHeaders(_ *peer.Peer, msg *wire.MsgGetHeaders) {
	// Ignore getheaders requests if not in sync.
	if !sp.MemPoolNode.syncManager.IsCurrent() {
		return
	}

	// Find the most recent known block in the best chain based on the block
	// locator and fetch all of the headers after it until either
	// wire.MaxBlockHeadersPerMsg have been fetched or the provided stop
	// hash is encountered.
	//
	// Use the block after the genesis block if no other blocks in the
	// provided locator are known.  This does mean the client will start
	// over with the genesis block if unknown block locators are provided.
	//
	// This mirrors the behavior in the reference implementation.
	chain := sp.MemPoolNode.indexer
	headers := chain.LocateHeaders(msg.BlockLocatorHashes, &msg.HashStop)

	// Send found headers to the requesting peer.
	blockHeaders := make([]*wire.BlockHeader, len(headers))
	for i := range headers {
		blockHeaders[i] = &headers[i]
	}
	sp.QueueMessage(&wire.MsgHeaders{Headers: blockHeaders}, nil)
}

// OnGetCFilters is invoked when a peer receives a getcfilters bitcoin message.
func (sp *serverPeer) OnGetCFilters(_ *peer.Peer, msg *wire.MsgGetCFilters) {
	// Ignore getcfilters requests if not in sync.
	if !sp.MemPoolNode.syncManager.IsCurrent() {
		return
	}

	// We'll also ensure that the remote party is requesting a set of
	// filters that we actually currently maintain.
	switch msg.FilterType {
	case wire.GCSFilterRegular:
		break

	default:
		common.Log.Debugf("Filter request for unknown filter: %v",
			msg.FilterType)
		return
	}

	hashes, err := sp.MemPoolNode.indexer.HeightToHashRange(
		int32(msg.StartHeight), &msg.StopHash, wire.MaxGetCFiltersReqRange,
	)
	if err != nil {
		common.Log.Debugf("Invalid getcfilters request: %v", err)
		return
	}

	// Create []*chainhash.Hash from []chainhash.Hash to pass to
	// FiltersByBlockHashes.
	hashPtrs := make([]*chainhash.Hash, len(hashes))
	for i := range hashes {
		hashPtrs[i] = &hashes[i]
	}

	filters, err := sp.MemPoolNode.indexer.FiltersByBlockHashes(
		hashPtrs, msg.FilterType,
	)
	if err != nil {
		common.Log.Errorf("Error retrieving cfilters: %v", err)
		return
	}

	for i, filterBytes := range filters {
		if len(filterBytes) == 0 {
			common.Log.Warnf("Could not obtain cfilter for %v",
				hashes[i])
			return
		}

		filterMsg := wire.NewMsgCFilter(
			msg.FilterType, &hashes[i], filterBytes,
		)
		sp.QueueMessage(filterMsg, nil)
	}
}

// OnGetCFHeaders is invoked when a peer receives a getcfheader bitcoin message.
func (sp *serverPeer) OnGetCFHeaders(_ *peer.Peer, msg *wire.MsgGetCFHeaders) {
	// Ignore getcfilterheader requests if not in sync.
	if !sp.MemPoolNode.syncManager.IsCurrent() {
		return
	}

	// We'll also ensure that the remote party is requesting a set of
	// headers for filters that we actually currently maintain.
	switch msg.FilterType {
	case wire.GCSFilterRegular:
		break

	default:
		common.Log.Debug("Filter request for unknown headers for "+
			"filter: %v", msg.FilterType)
		return
	}

	startHeight := int32(msg.StartHeight)
	maxResults := wire.MaxCFHeadersPerMsg

	// If StartHeight is positive, fetch the predecessor block hash so we
	// can populate the PrevFilterHeader field.
	if msg.StartHeight > 0 {
		startHeight--
		maxResults++
	}

	// Fetch the hashes from the block index.
	hashList, err := sp.MemPoolNode.indexer.HeightToHashRange(
		startHeight, &msg.StopHash, maxResults,
	)
	if err != nil {
		common.Log.Debugf("Invalid getcfheaders request: %v", err)
	}

	// This is possible if StartHeight is one greater that the height of
	// StopHash, and we pull a valid range of hashes including the previous
	// filter header.
	if len(hashList) == 0 || (msg.StartHeight > 0 && len(hashList) == 1) {
		common.Log.Debug("No results for getcfheaders request")
		return
	}

	// Create []*chainhash.Hash from []chainhash.Hash to pass to
	// FilterHeadersByBlockHashes.
	hashPtrs := make([]*chainhash.Hash, len(hashList))
	for i := range hashList {
		hashPtrs[i] = &hashList[i]
	}

	// Fetch the raw filter hash bytes from the database for all blocks.
	filterHashes, err := sp.MemPoolNode.indexer.FilterHashesByBlockHashes(
		hashPtrs, msg.FilterType,
	)
	if err != nil {
		common.Log.Errorf("Error retrieving cfilter hashes: %v", err)
		return
	}

	// Generate cfheaders message and send it.
	headersMsg := wire.NewMsgCFHeaders()

	// Populate the PrevFilterHeader field.
	if msg.StartHeight > 0 {
		prevBlockHash := &hashList[0]

		// Fetch the raw committed filter header bytes from the
		// database.
		headerBytes, err := sp.MemPoolNode.indexer.FilterHeaderByBlockHash(
			prevBlockHash, msg.FilterType)
		if err != nil {
			common.Log.Errorf("Error retrieving CF header: %v", err)
			return
		}
		if len(headerBytes) == 0 {
			common.Log.Warnf("Could not obtain CF header for %v", prevBlockHash)
			return
		}

		// Deserialize the hash into PrevFilterHeader.
		err = headersMsg.PrevFilterHeader.SetBytes(headerBytes)
		if err != nil {
			common.Log.Warnf("Committed filter header deserialize "+
				"failed: %v", err)
			return
		}

		hashList = hashList[1:]
		filterHashes = filterHashes[1:]
	}

	// Populate HeaderHashes.
	for i, hashBytes := range filterHashes {
		if len(hashBytes) == 0 {
			common.Log.Warnf("Could not obtain CF hash for %v", hashList[i])
			return
		}

		// Deserialize the hash.
		filterHash, err := chainhash.NewHash(hashBytes)
		if err != nil {
			common.Log.Warnf("Committed filter hash deserialize "+
				"failed: %v", err)
			return
		}

		headersMsg.AddCFHash(filterHash)
	}

	headersMsg.FilterType = msg.FilterType
	headersMsg.StopHash = msg.StopHash

	sp.QueueMessage(headersMsg, nil)
}

// OnGetCFCheckpt is invoked when a peer receives a getcfcheckpt bitcoin message.
func (sp *serverPeer) OnGetCFCheckpt(_ *peer.Peer, msg *wire.MsgGetCFCheckpt) {
	// Ignore getcfcheckpt requests if not in sync.
	if !sp.MemPoolNode.syncManager.IsCurrent() {
		return
	}

	// We'll also ensure that the remote party is requesting a set of
	// checkpoints for filters that we actually currently maintain.
	switch msg.FilterType {
	case wire.GCSFilterRegular:
		break

	default:
		common.Log.Debugf("Filter request for unknown checkpoints for "+
			"filter: %v", msg.FilterType)
		return
	}

	// Now that we know the client is fetching a filter that we know of,
	// we'll fetch the block hashes et each check point interval so we can
	// compare against our cache, and create new check points if necessary.
	blockHashes, err := sp.MemPoolNode.indexer.IntervalBlockHashes(
		&msg.StopHash, wire.CFCheckptInterval,
	)
	if err != nil {
		common.Log.Debugf("Invalid getcfilters request: %v", err)
		return
	}

	checkptMsg := wire.NewMsgCFCheckpt(
		msg.FilterType, &msg.StopHash, len(blockHashes),
	)

	// Fetch the current existing cache so we can decide if we need to
	// extend it or if its adequate as is.
	sp.MemPoolNode.cfCheckptCachesMtx.RLock()
	checkptCache := sp.MemPoolNode.cfCheckptCaches[msg.FilterType]

	// If the set of block hashes is beyond the current size of the cache,
	// then we'll expand the size of the cache and also retain the write
	// lock.
	var updateCache bool
	if len(blockHashes) > len(checkptCache) {
		// Now that we know we'll need to modify the size of the cache,
		// we'll release the read lock and grab the write lock to
		// possibly expand the cache size.
		sp.MemPoolNode.cfCheckptCachesMtx.RUnlock()

		sp.MemPoolNode.cfCheckptCachesMtx.Lock()
		defer sp.MemPoolNode.cfCheckptCachesMtx.Unlock()

		// Now that we have the write lock, we'll check again as it's
		// possible that the cache has already been expanded.
		checkptCache = sp.MemPoolNode.cfCheckptCaches[msg.FilterType]

		// If we still need to expand the cache, then We'll mark that
		// we need to update the cache for below and also expand the
		// size of the cache in place.
		if len(blockHashes) > len(checkptCache) {
			updateCache = true

			additionalLength := len(blockHashes) - len(checkptCache)
			newEntries := make([]cfHeaderKV, additionalLength)

			common.Log.Infof("Growing size of checkpoint cache from %v to %v "+
				"block hashes", len(checkptCache), len(blockHashes))

			checkptCache = append(
				sp.MemPoolNode.cfCheckptCaches[msg.FilterType],
				newEntries...,
			)
		}
	} else {
		// Otherwise, we'll hold onto the read lock for the remainder
		// of this method.
		defer sp.MemPoolNode.cfCheckptCachesMtx.RUnlock()

		common.Log.Tracef("Serving stale cache of size %v",
			len(checkptCache))
	}

	// Now that we know the cache is of an appropriate size, we'll iterate
	// backwards until the find the block hash. We do this as it's possible
	// a re-org has occurred so items in the db are now in the main china
	// while the cache has been partially invalidated.
	var forkIdx int
	for forkIdx = len(blockHashes); forkIdx > 0; forkIdx-- {
		if checkptCache[forkIdx-1].blockHash == blockHashes[forkIdx-1] {
			break
		}
	}

	// Now that we know the how much of the cache is relevant for this
	// query, we'll populate our check point message with the cache as is.
	// Shortly below, we'll populate the new elements of the cache.
	for i := 0; i < forkIdx; i++ {
		checkptMsg.AddCFHeader(&checkptCache[i].filterHeader)
	}

	// We'll now collect the set of hashes that are beyond our cache so we
	// can look up the filter headers to populate the final cache.
	blockHashPtrs := make([]*chainhash.Hash, 0, len(blockHashes)-forkIdx)
	for i := forkIdx; i < len(blockHashes); i++ {
		blockHashPtrs = append(blockHashPtrs, &blockHashes[i])
	}
	filterHeaders, err := sp.MemPoolNode.indexer.FilterHeadersByBlockHashes(
		blockHashPtrs, msg.FilterType,
	)
	if err != nil {
		common.Log.Errorf("Error retrieving cfilter headers: %v", err)
		return
	}

	// Now that we have the full set of filter headers, we'll add them to
	// the checkpoint message, and also update our cache in line.
	for i, filterHeaderBytes := range filterHeaders {
		if len(filterHeaderBytes) == 0 {
			common.Log.Warnf("Could not obtain CF header for %v",
				blockHashPtrs[i])
			return
		}

		filterHeader, err := chainhash.NewHash(filterHeaderBytes)
		if err != nil {
			common.Log.Warnf("Committed filter header deserialize "+
				"failed: %v", err)
			return
		}

		checkptMsg.AddCFHeader(filterHeader)

		// If the new main chain is longer than what's in the cache,
		// then we'll override it beyond the fork point.
		if updateCache {
			checkptCache[forkIdx+i] = cfHeaderKV{
				blockHash:    blockHashes[forkIdx+i],
				filterHeader: *filterHeader,
			}
		}
	}

	// Finally, we'll update the cache if we need to, and send the final
	// message back to the requesting peer.
	if updateCache {
		sp.MemPoolNode.cfCheckptCaches[msg.FilterType] = checkptCache
	}

	sp.QueueMessage(checkptMsg, nil)
}

// enforceNodeBloomFlag disconnects the peer if the MemPoolNode is not configured to
// allow bloom filters.  Additionally, if the peer has negotiated to a protocol
// version  that is high enough to observe the bloom filter service support bit,
// it will be banned since it is intentionally violating the protocol.
func (sp *serverPeer) enforceNodeBloomFlag(cmd string) bool {
	if sp.MemPoolNode.services&wire.SFNodeBloom != wire.SFNodeBloom {
		// Ban the peer if the protocol version is high enough that the
		// peer is knowingly violating the protocol and banning is
		// enabled.
		//
		// NOTE: Even though the addBanScore function already examines
		// whether or not banning is enabled, it is checked here as well
		// to ensure the violation is logged and the peer is
		// disconnected regardless.
		if sp.ProtocolVersion() >= wire.BIP0111Version &&
			!_cfg.DisableBanning {

			// Disconnect the peer regardless of whether it was
			// banned.
			sp.addBanScore(100, 0, cmd)
			sp.Disconnect()
			return false
		}

		// Disconnect the peer regardless of protocol version or banning
		// state.
		common.Log.Debugf("%s sent an unsupported %s request -- "+
			"disconnecting", sp, cmd)
		sp.Disconnect()
		return false
	}

	return true
}

// OnFeeFilter is invoked when a peer receives a feefilter bitcoin message and
// is used by remote peers to request that no transactions which have a fee rate
// lower than provided value are inventoried to them.  The peer will be
// disconnected if an invalid fee filter value is provided.
func (sp *serverPeer) OnFeeFilter(_ *peer.Peer, msg *wire.MsgFeeFilter) {
	// Check that the passed minimum fee is a valid amount.
	if msg.MinFee < 0 || msg.MinFee > btcutil.MaxSatoshi {
		common.Log.Debugf("Peer %v sent an invalid feefilter '%v' -- "+
			"disconnecting", sp, btcutil.Amount(msg.MinFee))
		sp.Disconnect()
		return
	}

	atomic.StoreInt64(&sp.feeFilter, msg.MinFee)
}

// OnFilterAdd is invoked when a peer receives a filteradd bitcoin
// message and is used by remote peers to add data to an already loaded bloom
// filter.  The peer will be disconnected if a filter is not loaded when this
// message is received or the MemPoolNode is not configured to allow bloom filters.
func (sp *serverPeer) OnFilterAdd(_ *peer.Peer, msg *wire.MsgFilterAdd) {
	// Disconnect and/or ban depending on the node bloom services flag and
	// negotiated protocol version.
	if !sp.enforceNodeBloomFlag(msg.Command()) {
		return
	}

	if !sp.filter.IsLoaded() {
		common.Log.Debugf("%s sent a filteradd request with no filter "+
			"loaded -- disconnecting", sp)
		sp.Disconnect()
		return
	}

	sp.filter.Add(msg.Data)
}

// OnFilterClear is invoked when a peer receives a filterclear bitcoin
// message and is used by remote peers to clear an already loaded bloom filter.
// The peer will be disconnected if a filter is not loaded when this message is
// received  or the MemPoolNode is not configured to allow bloom filters.
func (sp *serverPeer) OnFilterClear(_ *peer.Peer, msg *wire.MsgFilterClear) {
	// Disconnect and/or ban depending on the node bloom services flag and
	// negotiated protocol version.
	if !sp.enforceNodeBloomFlag(msg.Command()) {
		return
	}

	if !sp.filter.IsLoaded() {
		common.Log.Debugf("%s sent a filterclear request with no "+
			"filter loaded -- disconnecting", sp)
		sp.Disconnect()
		return
	}

	sp.filter.Unload()
}

// OnFilterLoad is invoked when a peer receives a filterload bitcoin
// message and it used to load a bloom filter that should be used for
// delivering merkle blocks and associated transactions that match the filter.
// The peer will be disconnected if the MemPoolNode is not configured to allow bloom
// filters.
func (sp *serverPeer) OnFilterLoad(_ *peer.Peer, msg *wire.MsgFilterLoad) {
	// Disconnect and/or ban depending on the node bloom services flag and
	// negotiated protocol version.
	if !sp.enforceNodeBloomFlag(msg.Command()) {
		return
	}

	sp.setDisableRelayTx(false)

	sp.filter.Reload(msg)
}

// OnGetAddr is invoked when a peer receives a getaddr bitcoin message
// and is used to provide the peer with known addresses from the address
// manager.
func (sp *serverPeer) OnGetAddr(_ *peer.Peer, msg *wire.MsgGetAddr) {
	// Don't return any addresses when running on the simulation test
	// network.  This helps prevent the network from becoming another
	// public test network since it will not be able to learn about other
	// peers that have not specifically been provided.
	if _cfg.SimNet {
		return
	}

	// Do not accept getaddr requests from outbound peers.  This reduces
	// fingerprinting attacks.
	if !sp.Inbound() {
		common.Log.Debugf("Ignoring getaddr request from outbound peer "+
			"%v", sp)
		return
	}

	// Only allow one getaddr request per connection to discourage
	// address stamping of inv announcements.
	if sp.sentAddrs {
		common.Log.Debugf("Ignoring repeated getaddr request from peer "+
			"%v", sp)
		return
	}
	sp.sentAddrs = true

	// Get the current known addresses from the address manager.
	addrCache := sp.MemPoolNode.addrManager.AddressCache()

	// Push the addresses.
	sp.pushAddrMsg(addrCache)
}

// OnAddr is invoked when a peer receives an addr bitcoin message and is
// used to notify the MemPoolNode about advertised addresses.
func (sp *serverPeer) OnAddr(_ *peer.Peer, msg *wire.MsgAddr) {
	// Ignore addresses when running on the simulation test network.  This
	// helps prevent the network from becoming another public test network
	// since it will not be able to learn about other peers that have not
	// specifically been provided.
	if _cfg.SimNet {
		return
	}

	// Ignore old style addresses which don't include a timestamp.
	if sp.ProtocolVersion() < wire.NetAddressTimeVersion {
		return
	}

	// A message that has no addresses is invalid.
	if len(msg.AddrList) == 0 {
		common.Log.Errorf("Command [%s] from %s does not contain any addresses",
			msg.Command(), sp.Peer)
		sp.Disconnect()
		return
	}

	addrs := make([]*wire.NetAddressV2, 0, len(msg.AddrList))
	for _, na := range msg.AddrList {
		// Don't add more address if we're disconnecting.
		if !sp.Connected() {
			return
		}

		// Set the timestamp to 5 days ago if it's more than 24 hours
		// in the future so this address is one of the first to be
		// removed when space is needed.
		now := time.Now()
		if na.Timestamp.After(now.Add(time.Minute * 10)) {
			na.Timestamp = now.Add(-1 * time.Hour * 24 * 5)
		}

		// Add address to known addresses for this peer. This is
		// converted to NetAddressV2 since that's what the address
		// manager uses.
		currentNa := wire.NetAddressV2FromBytes(
			na.Timestamp, na.Services, na.IP, na.Port,
		)
		addrs = append(addrs, currentNa)
		sp.addKnownAddresses([]*wire.NetAddressV2{currentNa})
	}

	// Add addresses to MemPoolNode address manager.  The address manager handles
	// the details of things such as preventing duplicate addresses, max
	// addresses, and last seen updates.
	// XXX bitcoind gives a 2 hour time penalty here, do we want to do the
	// same?
	sp.MemPoolNode.addrManager.AddAddresses(addrs, sp.NA())
}

// OnAddrV2 is invoked when a peer receives an addrv2 bitcoin message and is
// used to notify the MemPoolNode about advertised addresses.
func (sp *serverPeer) OnAddrV2(_ *peer.Peer, msg *wire.MsgAddrV2) {
	// Ignore if simnet for the same reasons as the regular addr message.
	if _cfg.SimNet {
		return
	}

	// An empty AddrV2 message is invalid.
	if len(msg.AddrList) == 0 {
		common.Log.Errorf("Command [%s] from %s does not contain any "+
			"addresses", msg.Command(), sp.Peer)
		sp.Disconnect()
		return
	}

	for _, na := range msg.AddrList {
		// Don't add more to the set of known addresses if we're
		// disconnecting.
		if !sp.Connected() {
			return
		}

		// Set the timestamp to 5 days ago if the timestamp received is
		// more than 10 minutes in the future so this address is one of
		// the first to be removed.
		now := time.Now()
		if na.Timestamp.After(now.Add(time.Minute * 10)) {
			na.Timestamp = now.Add(-1 * time.Hour * 24 * 5)
		}

		// Add to the set of known addresses.
		sp.addKnownAddresses([]*wire.NetAddressV2{na})
	}

	// Add the addresses to the addrmanager.
	sp.MemPoolNode.addrManager.AddAddresses(msg.AddrList, sp.NA())
}

// OnRead is invoked when a peer receives a message and it is used to update
// the bytes received by the MemPoolNode.
func (sp *serverPeer) OnRead(_ *peer.Peer, bytesRead int, msg wire.Message, err error) {
	sp.MemPoolNode.AddBytesReceived(uint64(bytesRead))
}

// OnWrite is invoked when a peer sends a message and it is used to update
// the bytes sent by the MemPoolNode.
func (sp *serverPeer) OnWrite(_ *peer.Peer, bytesWritten int, msg wire.Message, err error) {
	sp.MemPoolNode.AddBytesSent(uint64(bytesWritten))
}

// OnNotFound is invoked when a peer sends a notfound message.
func (sp *serverPeer) OnNotFound(p *peer.Peer, msg *wire.MsgNotFound) {
	if !sp.Connected() {
		return
	}

	var numBlocks, numTxns uint32
	for _, inv := range msg.InvList {
		switch inv.Type {
		case wire.InvTypeBlock:
			numBlocks++
		case wire.InvTypeWitnessBlock:
			numBlocks++
		case wire.InvTypeTx:
			numTxns++
		case wire.InvTypeWitnessTx:
			numTxns++
		default:
			common.Log.Debugf("Invalid inv type '%d' in notfound message from %s",
				inv.Type, sp)
			sp.Disconnect()
			return
		}
	}
	if numBlocks > 0 {
		blockStr := localCommon.PickNoun(uint64(numBlocks), "block", "blocks")
		reason := fmt.Sprintf("%d %v not found", numBlocks, blockStr)
		if sp.addBanScore(20*numBlocks, 0, reason) {
			return
		}
	}
	if numTxns > 0 {
		txStr := localCommon.PickNoun(uint64(numTxns), "transaction", "transactions")
		reason := fmt.Sprintf("%d %v not found", numTxns, txStr)
		if sp.addBanScore(0, 10*numTxns, reason) {
			return
		}
	}

	sp.MemPoolNode.syncManager.QueueNotFound(msg, p)
}

// randomUint16Number returns a random uint16 in a specified input range.  Note
// that the range is in zeroth ordering; if you pass it 1800, you will get
// values from 0 to 1800.
func randomUint16Number(max uint16) uint16 {
	// In order to avoid modulo bias and ensure every possible outcome in
	// [0, max) has equal probability, the random number must be sampled
	// from a random source that has a range limited to a multiple of the
	// modulus.
	var randomNumber uint16
	var limitRange = (math.MaxUint16 / max) * max
	for {
		binary.Read(rand.Reader, binary.LittleEndian, &randomNumber)
		if randomNumber < limitRange {
			return (randomNumber % max)
		}
	}
}

// AddRebroadcastInventory adds 'iv' to the list of inventories to be
// rebroadcasted at random intervals until they show up in a block.
func (s *MemPoolNode) AddRebroadcastInventory(iv *wire.InvVect, data interface{}) {
	// Ignore if shutting down.
	if atomic.LoadInt32(&s.shutdown) != 0 {
		return
	}

	s.modifyRebroadcastInv <- broadcastInventoryAdd{invVect: iv, data: data}
}

// RemoveRebroadcastInventory removes 'iv' from the list of items to be
// rebroadcasted if present.
func (s *MemPoolNode) RemoveRebroadcastInventory(iv *wire.InvVect) {
	// Ignore if shutting down.
	if atomic.LoadInt32(&s.shutdown) != 0 {
		return
	}

	s.modifyRebroadcastInv <- broadcastInventoryDel(iv)
}

// relayTransactions generates and relays inventory vectors for all of the
// passed transactions to all connected peers.
func (s *MemPoolNode) relayTransactions(txns []*mempool.TxDesc) {
	for _, txD := range txns {
		iv := wire.NewInvVect(wire.InvTypeTx, txD.Tx.Hash())
		s.RelayInventory(iv, txD)
	}
}

// AnnounceNewTransactions generates and relays inventory vectors and notifies
// both websocket and getblocktemplate long poll clients of the passed
// transactions.  This function should be called whenever new transactions
// are added to the mempool.
func (s *MemPoolNode) AnnounceNewTransactions(txns []*mempool.TxDesc) {
	// Generate and relay inventory vectors for all newly accepted
	// transactions.
	s.relayTransactions(txns)

	// Notify both websocket and getblocktemplate long poll clients of all
	// newly accepted transactions.
	// if s.rpcServer != nil {
	// 	s.rpcServer.NotifyNewTransactions(txns)
	// }
}

// Transaction has one confirmation on the main chain. Now we can mark it as no
// longer needing rebroadcasting.
func (s *MemPoolNode) TransactionConfirmed(tx *btcutil.Tx) {
	// Rebroadcasting is only necessary when the RPC MemPoolNode is active.
	// if s.rpcServer == nil {
	// 	return
	// }

	iv := wire.NewInvVect(wire.InvTypeTx, tx.Hash())
	s.RemoveRebroadcastInventory(iv)
}

// pushTxMsg sends a tx message for the provided transaction hash to the
// connected peer.  An error is returned if the transaction hash is not known.
func (s *MemPoolNode) pushTxMsg(sp *serverPeer, hash *chainhash.Hash, doneChan chan<- struct{},
	waitChan <-chan struct{}, encoding wire.MessageEncoding) error {

	// Attempt to fetch the requested transaction from the pool.  A
	// call could be made to check for existence first, but simply trying
	// to fetch a missing transaction results in the same behavior.
	tx, err := s.txMemPool.FetchTransaction(hash)
	if err != nil {
		common.Log.Tracef("Unable to fetch tx %v from transaction "+
			"pool: %v", hash, err)

		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return err
	}

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	sp.QueueMessageWithEncoding(tx.MsgTx(), doneChan, encoding)

	return nil
}

// pushBlockMsg sends a block message for the provided block hash to the
// connected peer.  An error is returned if the block hash is not known.
func (s *MemPoolNode) pushBlockMsg(sp *serverPeer, hash *chainhash.Hash, doneChan chan<- struct{},
	waitChan <-chan struct{}, encoding wire.MessageEncoding) error {

	// Fetch the raw block bytes from the database.
	// var blockBytes []byte
	// err := sp.MemPoolNode.db.View(func(dbTx database.Tx) error {
	// 	var err error
	// 	blockBytes, err = dbTx.FetchBlock(hash)
	// 	return err
	// })
	// if err != nil {
	// 	common.Log.Tracef("Unable to fetch requested block hash %v: %v",
	// 		hash, err)

	// 	if doneChan != nil {
	// 		doneChan <- struct{}{}
	// 	}
	// 	return err
	// }

	// Deserialize the block.
	// var msgBlock wire.MsgBlock
	// err := msgBlock.Deserialize(bytes.NewReader(blockBytes))
	// if err != nil {
	// 	common.Log.Tracef("Unable to deserialize requested block hash "+
	// 		"%v: %v", hash, err)

	// 	if doneChan != nil {
	// 		doneChan <- struct{}{}
	// 	}
	// 	return err
	// }

	block, err := s.indexer.BlockByHash(hash)
	if err != nil {
		return err
	}
	msgBlock := block.MsgBlock()

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	// We only send the channel for this message if we aren't sending
	// an inv straight after.
	var dc chan<- struct{}
	continueHash := sp.continueHash
	sendInv := continueHash != nil && continueHash.IsEqual(hash)
	if !sendInv {
		dc = doneChan
	}
	sp.QueueMessageWithEncoding(msgBlock, dc, encoding)

	// When the peer requests the final block that was advertised in
	// response to a getblocks message which requested more blocks than
	// would fit into a single message, send it a new inventory message
	// to trigger it to issue another getblocks message for the next
	// batch of inventory.
	if sendInv {
		best := sp.MemPoolNode.indexer.BestSnapshot()
		invMsg := wire.NewMsgInvSizeHint(1)
		iv := wire.NewInvVect(wire.InvTypeBlock, &best.Hash)
		invMsg.AddInvVect(iv)
		sp.QueueMessage(invMsg, doneChan)
		sp.continueHash = nil
	}
	return nil
}

// pushMerkleBlockMsg sends a merkleblock message for the provided block hash to
// the connected peer.  Since a merkle block requires the peer to have a filter
// loaded, this call will simply be ignored if there is no filter loaded.  An
// error is returned if the block hash is not known.
func (s *MemPoolNode) pushMerkleBlockMsg(sp *serverPeer, hash *chainhash.Hash,
	doneChan chan<- struct{}, waitChan <-chan struct{}, encoding wire.MessageEncoding) error {

	// Do not send a response if the peer doesn't have a filter loaded.
	if !sp.filter.IsLoaded() {
		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return nil
	}

	// Fetch the raw block bytes from the database.
	blk, err := sp.MemPoolNode.indexer.BlockByHash(hash)
	if err != nil {
		common.Log.Tracef("Unable to fetch requested block hash %v: %v",
			hash, err)

		if doneChan != nil {
			doneChan <- struct{}{}
		}
		return err
	}

	// Generate a merkle block by filtering the requested block according
	// to the filter for the peer.
	merkle, matchedTxIndices := bloom.NewMerkleBlock(blk, sp.filter)

	// Once we have fetched data wait for any previous operation to finish.
	if waitChan != nil {
		<-waitChan
	}

	// Send the merkleblock.  Only send the done channel with this message
	// if no transactions will be sent afterwards.
	var dc chan<- struct{}
	if len(matchedTxIndices) == 0 {
		dc = doneChan
	}
	sp.QueueMessage(merkle, dc)

	// Finally, send any matched transactions.
	blkTransactions := blk.MsgBlock().Transactions
	for i, txIndex := range matchedTxIndices {
		// Only send the done channel on the final transaction.
		var dc chan<- struct{}
		if i == len(matchedTxIndices)-1 {
			dc = doneChan
		}
		if txIndex < uint32(len(blkTransactions)) {
			sp.QueueMessageWithEncoding(blkTransactions[txIndex], dc,
				encoding)
		}
	}

	return nil
}

// handleUpdatePeerHeight updates the heights of all peers who were known to
// announce a block we recently accepted.
func (s *MemPoolNode) handleUpdatePeerHeights(state *peerState, umsg updatePeerHeightsMsg) {
	state.forAllPeers(func(sp *serverPeer) {
		// The origin peer should already have the updated height.
		if sp.Peer == umsg.originPeer {
			return
		}

		// This is a pointer to the underlying memory which doesn't
		// change.
		latestBlkHash := sp.LastAnnouncedBlock()

		// Skip this peer if it hasn't recently announced any new blocks.
		if latestBlkHash == nil {
			return
		}

		// If the peer has recently announced a block, and this block
		// matches our newly accepted block, then update their block
		// height.
		if *latestBlkHash == *umsg.newHash {
			sp.UpdateLastBlockHeight(umsg.newHeight)
			sp.UpdateLastAnnouncedBlock(nil)
		}
	})
}

// handleAddPeerMsg deals with adding new peers.  It is invoked from the
// peerHandler goroutine.
func (s *MemPoolNode) handleAddPeerMsg(state *peerState, sp *serverPeer) bool {
	if sp == nil || !sp.Connected() {
		return false
	}

	// Disconnect peers with unwanted user agents.
	if sp.HasUndesiredUserAgent(s.agentBlacklist, s.agentWhitelist) {
		sp.Disconnect()
		return false
	}

	// Ignore new peers if we're shutting down.
	if atomic.LoadInt32(&s.shutdown) != 0 {
		common.Log.Infof("New peer %s ignored - MemPoolNode is shutting down", sp)
		sp.Disconnect()
		return false
	}

	// Disconnect banned peers.
	host, _, err := net.SplitHostPort(sp.Addr())
	if err != nil {
		common.Log.Debugf("can't split hostport %v", err)
		sp.Disconnect()
		return false
	}
	if banEnd, ok := state.banned[host]; ok {
		if time.Now().Before(banEnd) {
			common.Log.Debugf("Peer %s is banned for another %v - disconnecting",
				host, time.Until(banEnd))
			sp.Disconnect()
			return false
		}

		common.Log.Infof("Peer %s is no longer banned", host)
		delete(state.banned, host)
	}

	// TODO: Check for max peers from a single IP.

	// Limit max number of total peers.
	if state.Count() >= _cfg.MaxPeers {
		common.Log.Infof("Max peers reached [%d] - disconnecting peer %s",
			_cfg.MaxPeers, sp)
		sp.Disconnect()
		// TODO: how to handle permanent peers here?
		// they should be rescheduled.
		return false
	}

	// Add the new peer and start it.
	common.Log.Debugf("New peer %s", sp)
	if sp.Inbound() {
		state.inboundPeers[sp.ID()] = sp
	} else {
		state.outboundGroups[addrmgr.GroupKey(sp.NA())]++
		if sp.persistent {
			state.persistentPeers[sp.ID()] = sp
		} else {
			state.outboundPeers[sp.ID()] = sp
		}
	}

	// Update the address' last seen time if the peer has acknowledged
	// our version and has sent us its version as well.
	if sp.VerAckReceived() && sp.VersionKnown() && sp.NA() != nil {
		s.addrManager.Connected(sp.NA())
	}

	// Signal the sync manager this peer is a new sync candidate.
	s.syncManager.NewPeer(sp.Peer)

	// Update the address manager and request known addresses from the
	// remote peer for outbound connections. This is skipped when running on
	// the simulation test network since it is only intended to connect to
	// specified peers and actively avoids advertising and connecting to
	// discovered peers.
	if !_cfg.SimNet && !sp.Inbound() {
		// Advertise the local address when the MemPoolNode accepts incoming
		// connections and it believes itself to be close to the best
		// known tip.
		if !_cfg.DisableListen && s.syncManager.IsCurrent() {
			// Get address that best matches.
			lna := s.addrManager.GetBestLocalAddress(sp.NA())
			if addrmgr.IsRoutable(lna) {
				// Filter addresses the peer already knows about.
				addresses := []*wire.NetAddressV2{lna}
				sp.pushAddrMsg(addresses)
			}
		}

		// Request known addresses if the MemPoolNode address manager needs
		// more and the peer has a protocol version new enough to
		// include a timestamp with addresses.
		hasTimestamp := sp.ProtocolVersion() >= wire.NetAddressTimeVersion
		if s.addrManager.NeedMoreAddresses() && hasTimestamp {
			sp.QueueMessage(wire.NewMsgGetAddr(), nil)
		}

		// Mark the address as a known good address.
		s.addrManager.Good(sp.NA())
	}

	return true
}

// handleDonePeerMsg deals with peers that have signalled they are done.  It is
// invoked from the peerHandler goroutine.
func (s *MemPoolNode) handleDonePeerMsg(state *peerState, sp *serverPeer) {
	var list map[int32]*serverPeer
	if sp.persistent {
		list = state.persistentPeers
	} else if sp.Inbound() {
		list = state.inboundPeers
	} else {
		list = state.outboundPeers
	}

	// Regardless of whether the peer was found in our list, we'll inform
	// our connection manager about the disconnection. This can happen if we
	// process a peer's `done` message before its `add`.
	if !sp.Inbound() {
		if sp.persistent {
			s.connManager.Disconnect(sp.connReq.ID())
		} else {
			s.connManager.Remove(sp.connReq.ID())
			go s.connManager.NewConnReq()
		}
	}

	if _, ok := list[sp.ID()]; ok {
		if !sp.Inbound() && sp.VersionKnown() {
			state.outboundGroups[addrmgr.GroupKey(sp.NA())]--
		}
		delete(list, sp.ID())
		common.Log.Debugf("Removed peer %s", sp)
		return
	}
}

// handleBanPeerMsg deals with banning peers.  It is invoked from the
// peerHandler goroutine.
func (s *MemPoolNode) handleBanPeerMsg(state *peerState, sp *serverPeer) {
	host, _, err := net.SplitHostPort(sp.Addr())
	if err != nil {
		common.Log.Debugf("can't split ban peer %s %v", sp.Addr(), err)
		return
	}
	direction := localCommon.DirectionString(sp.Inbound())
	common.Log.Infof("Banned peer %s (%s) for %v", host, direction,
		_cfg.BanDuration)
	state.banned[host] = time.Now().Add(_cfg.BanDuration)
}

// handleRelayInvMsg deals with relaying inventory to peers that are not already
// known to have it.  It is invoked from the peerHandler goroutine.
func (s *MemPoolNode) handleRelayInvMsg(state *peerState, msg relayMsg) {
	state.forAllPeers(func(sp *serverPeer) {
		if !sp.Connected() {
			return
		}

		// If the inventory is a block and the peer prefers headers,
		// generate and send a headers message instead of an inventory
		// message.
		if msg.invVect.Type == wire.InvTypeBlock && sp.WantsHeaders() {
			blockHeader, ok := msg.data.(wire.BlockHeader)
			if !ok {
				common.Log.Warnf("Underlying data for headers" +
					" is not a block header")
				return
			}
			msgHeaders := wire.NewMsgHeaders()
			if err := msgHeaders.AddBlockHeader(&blockHeader); err != nil {
				common.Log.Errorf("Failed to add block"+
					" header: %v", err)
				return
			}
			sp.QueueMessage(msgHeaders, nil)
			return
		}

		if msg.invVect.Type == wire.InvTypeTx {
			// Don't relay the transaction to the peer when it has
			// transaction relaying disabled.
			if sp.relayTxDisabled() {
				return
			}

			txD, ok := msg.data.(*mempool.TxDesc)
			if !ok {
				common.Log.Warnf("Underlying data for tx inv "+
					"relay is not a *mempool.TxDesc: %T",
					msg.data)
				return
			}

			// Don't relay the transaction if the transaction fee-per-kb
			// is less than the peer's feefilter.
			feeFilter := atomic.LoadInt64(&sp.feeFilter)
			if feeFilter > 0 && txD.FeePerKB < feeFilter {
				return
			}

			// Don't relay the transaction if there is a bloom
			// filter loaded and the transaction doesn't match it.
			if sp.filter.IsLoaded() {
				if !sp.filter.MatchTxAndUpdate(txD.Tx) {
					return
				}
			}
		}

		// Queue the inventory to be relayed with the next batch.
		// It will be ignored if the peer is already known to
		// have the inventory.
		sp.QueueInventory(msg.invVect)
	})
}

// handleBroadcastMsg deals with broadcasting messages to peers.  It is invoked
// from the peerHandler goroutine.
func (s *MemPoolNode) handleBroadcastMsg(state *peerState, bmsg *broadcastMsg) {
	state.forAllPeers(func(sp *serverPeer) {
		if !sp.Connected() {
			return
		}

		for _, ep := range bmsg.excludePeers {
			if sp == ep {
				return
			}
		}

		sp.QueueMessage(bmsg.message, nil)
	})
}

type getConnCountMsg struct {
	reply chan int32
}

type getPeersMsg struct {
	reply chan []*serverPeer
}

type getOutboundGroup struct {
	key   string
	reply chan int
}

type getAddedNodesMsg struct {
	reply chan []*serverPeer
}

type disconnectNodeMsg struct {
	cmp   func(*serverPeer) bool
	reply chan error
}

type connectNodeMsg struct {
	addr      string
	permanent bool
	reply     chan error
}

type removeNodeMsg struct {
	cmp   func(*serverPeer) bool
	reply chan error
}

// handleQuery is the central handler for all queries and commands from other
// goroutines related to peer state.
func (s *MemPoolNode) handleQuery(state *peerState, querymsg interface{}) {
	switch msg := querymsg.(type) {
	case getConnCountMsg:
		nconnected := int32(0)
		state.forAllPeers(func(sp *serverPeer) {
			if sp.Connected() {
				nconnected++
			}
		})
		msg.reply <- nconnected

	case getPeersMsg:
		peers := make([]*serverPeer, 0, state.Count())
		state.forAllPeers(func(sp *serverPeer) {
			if !sp.Connected() {
				return
			}
			peers = append(peers, sp)
		})
		msg.reply <- peers

	case connectNodeMsg:
		// TODO: duplicate oneshots?
		// Limit max number of total peers.
		if state.Count() >= _cfg.MaxPeers {
			msg.reply <- errors.New("max peers reached")
			return
		}
		for _, peer := range state.persistentPeers {
			if peer.Addr() == msg.addr {
				if msg.permanent {
					msg.reply <- errors.New("peer already connected")
				} else {
					msg.reply <- errors.New("peer exists as a permanent peer")
				}
				return
			}
		}

		netAddr, err := addrStringToNetAddr(msg.addr)
		if err != nil {
			msg.reply <- err
			return
		}

		// TODO: if too many, nuke a non-perm peer.
		go s.connManager.Connect(&connmgr.ConnReq{
			Addr:      netAddr,
			Permanent: msg.permanent,
		})
		msg.reply <- nil
	case removeNodeMsg:
		found := disconnectPeer(state.persistentPeers, msg.cmp, func(sp *serverPeer) {
			// Keep group counts ok since we remove from
			// the list now.
			state.outboundGroups[addrmgr.GroupKey(sp.NA())]--
		})

		if found {
			msg.reply <- nil
		} else {
			msg.reply <- errors.New("peer not found")
		}
	case getOutboundGroup:
		count, ok := state.outboundGroups[msg.key]
		if ok {
			msg.reply <- count
		} else {
			msg.reply <- 0
		}
	// Request a list of the persistent (added) peers.
	case getAddedNodesMsg:
		// Respond with a slice of the relevant peers.
		peers := make([]*serverPeer, 0, len(state.persistentPeers))
		for _, sp := range state.persistentPeers {
			peers = append(peers, sp)
		}
		msg.reply <- peers
	case disconnectNodeMsg:
		// Check inbound peers. We pass a nil callback since we don't
		// require any additional actions on disconnect for inbound peers.
		found := disconnectPeer(state.inboundPeers, msg.cmp, nil)
		if found {
			msg.reply <- nil
			return
		}

		// Check outbound peers.
		found = disconnectPeer(state.outboundPeers, msg.cmp, func(sp *serverPeer) {
			// Keep group counts ok since we remove from
			// the list now.
			state.outboundGroups[addrmgr.GroupKey(sp.NA())]--
		})
		if found {
			// If there are multiple outbound connections to the same
			// ip:port, continue disconnecting them all until no such
			// peers are found.
			for found {
				found = disconnectPeer(state.outboundPeers, msg.cmp, func(sp *serverPeer) {
					state.outboundGroups[addrmgr.GroupKey(sp.NA())]--
				})
			}
			msg.reply <- nil
			return
		}

		msg.reply <- errors.New("peer not found")
	}
}

// disconnectPeer attempts to drop the connection of a targeted peer in the
// passed peer list. Targets are identified via usage of the passed
// `compareFunc`, which should return `true` if the passed peer is the target
// peer. This function returns true on success and false if the peer is unable
// to be located. If the peer is found, and the passed callback: `whenFound'
// isn't nil, we call it with the peer as the argument before it is removed
// from the peerList, and is disconnected from the MemPoolNode.
func disconnectPeer(peerList map[int32]*serverPeer, compareFunc func(*serverPeer) bool, whenFound func(*serverPeer)) bool {
	for addr, peer := range peerList {
		if compareFunc(peer) {
			if whenFound != nil {
				whenFound(peer)
			}

			// This is ok because we are not continuing
			// to iterate so won't corrupt the loop.
			delete(peerList, addr)
			peer.Disconnect()
			return true
		}
	}
	return false
}

// newPeerConfig returns the configuration for the given serverPeer.
func newPeerConfig(sp *serverPeer) *peer.Config {
	return &peer.Config{
		Listeners: peer.MessageListeners{
			OnVersion:      sp.OnVersion,
			OnVerAck:       sp.OnVerAck,
			OnMemPool:      sp.OnMemPool,
			OnTx:           sp.OnTx,
			OnBlock:        sp.OnBlock,
			OnInv:          sp.OnInv,
			OnHeaders:      sp.OnHeaders,
			OnGetData:      sp.OnGetData,
			OnGetBlocks:    sp.OnGetBlocks,
			OnGetHeaders:   sp.OnGetHeaders,
			OnGetCFilters:  sp.OnGetCFilters,
			OnGetCFHeaders: sp.OnGetCFHeaders,
			OnGetCFCheckpt: sp.OnGetCFCheckpt,
			OnFeeFilter:    sp.OnFeeFilter,
			OnFilterAdd:    sp.OnFilterAdd,
			OnFilterClear:  sp.OnFilterClear,
			OnFilterLoad:   sp.OnFilterLoad,
			OnGetAddr:      sp.OnGetAddr,
			OnAddr:         sp.OnAddr,
			OnAddrV2:       sp.OnAddrV2,
			OnRead:         sp.OnRead,
			OnWrite:        sp.OnWrite,
			OnNotFound:     sp.OnNotFound,

			// Note: The reference client currently bans peers that send alerts
			// not signed with its key.  We could verify against their key, but
			// since the reference client is currently unwilling to support
			// other implementations' alert messages, we will not relay theirs.
			OnAlert: nil,
		},
		NewestBlock:         sp.newestBlock,
		HostToNetAddress:    sp.MemPoolNode.addrManager.HostToNetAddress,
		Proxy:               _cfg.Proxy,
		UserAgentName:       userAgentName,
		UserAgentVersion:    userAgentVersion,
		UserAgentComments:   _cfg.UserAgentComments,
		ChainParams:         sp.MemPoolNode.chainParams,
		Services:            sp.MemPoolNode.services,
		DisableRelayTx:      _cfg.BlocksOnly,
		ProtocolVersion:     peer.MaxProtocolVersion,
		TrickleInterval:     _cfg.TrickleInterval,
		DisableStallHandler: _cfg.DisableStallHandler,
	}
}

// inboundPeerConnected is invoked by the connection manager when a new inbound
// connection is established.  It initializes a new inbound MemPoolNode peer
// instance, associates it with the connection, and starts a goroutine to wait
// for disconnection.
func (s *MemPoolNode) inboundPeerConnected(conn net.Conn) {
	sp := newServerPeer(s, false)
	sp.isWhitelisted = isWhitelisted(conn.RemoteAddr())
	sp.Peer = peer.NewInboundPeer(newPeerConfig(sp))
	sp.AssociateConnection(conn)
	go s.peerDoneHandler(sp)
}

// outboundPeerConnected is invoked by the connection manager when a new
// outbound connection is established.  It initializes a new outbound MemPoolNode
// peer instance, associates it with the relevant state such as the connection
// request instance and the connection itself, and finally notifies the address
// manager of the attempt.
func (s *MemPoolNode) outboundPeerConnected(c *connmgr.ConnReq, conn net.Conn) {
	sp := newServerPeer(s, c.Permanent)
	p, err := peer.NewOutboundPeer(newPeerConfig(sp), c.Addr.String())
	if err != nil {
		common.Log.Debugf("Cannot create outbound peer %s: %v", c.Addr, err)
		if c.Permanent {
			s.connManager.Disconnect(c.ID())
		} else {
			s.connManager.Remove(c.ID())
			go s.connManager.NewConnReq()
		}
		return
	}
	sp.Peer = p
	sp.connReq = c
	sp.isWhitelisted = isWhitelisted(conn.RemoteAddr())
	sp.AssociateConnection(conn)
	go s.peerDoneHandler(sp)
}

// peerDoneHandler handles peer disconnects by notifying the MemPoolNode that it's
// done along with other performing other desirable cleanup.
func (s *MemPoolNode) peerDoneHandler(sp *serverPeer) {
	sp.WaitForDisconnect()
	s.donePeers <- sp

	// Only tell sync manager we are gone if we ever told it we existed.
	if sp.VerAckReceived() {
		s.syncManager.DonePeer(sp.Peer)

		// Evict any remaining orphans that were sent by the peer.
		numEvicted := s.txMemPool.RemoveOrphansByTag(mempool.Tag(sp.ID()))
		if numEvicted > 0 {
			common.Log.Debugf("Evicted %d %s from peer %v (id %d)",
				numEvicted, localCommon.PickNoun(numEvicted, "orphan",
					"orphans"), sp, sp.ID())
		}
	}
	close(sp.quit)
}

// peerHandler is used to handle peer operations such as adding and removing
// peers to and from the MemPoolNode, banning peers, and broadcasting messages to
// peers.  It must be run in a goroutine.
func (s *MemPoolNode) peerHandler() {
	// Start the address manager and sync manager, both of which are needed
	// by peers.  This is done here since their lifecycle is closely tied
	// to this handler and rather than adding more channels to synchronize
	// things, it's easier and slightly faster to simply start and stop them
	// in this handler.
	s.addrManager.Start()
	s.syncManager.Start()

	common.Log.Tracef("Starting peer handler")

	state := &peerState{
		inboundPeers:    make(map[int32]*serverPeer),
		persistentPeers: make(map[int32]*serverPeer),
		outboundPeers:   make(map[int32]*serverPeer),
		banned:          make(map[string]time.Time),
		outboundGroups:  make(map[string]int),
	}

	if !_cfg.DisableDNSSeed {
		// Add peers discovered through DNS to the address manager.
		connmgr.SeedFromDNS(activeNetParams.Params, defaultRequiredServices,
			btcdLookup, func(addrs []*wire.NetAddressV2) {
				// Bitcoind uses a lookup of the dns seeder here. This
				// is rather strange since the values looked up by the
				// DNS seed lookups will vary quite a lot.
				// to replicate this behaviour we put all addresses as
				// having come from the first one.
				s.addrManager.AddAddresses(addrs, addrs[0])
			})
	}
	go s.connManager.Start()

out:
	for {
		select {
		// New peers connected to the MemPoolNode.
		case p := <-s.newPeers:
			s.handleAddPeerMsg(state, p)

		// Disconnected peers.
		case p := <-s.donePeers:
			s.handleDonePeerMsg(state, p)

		// Block accepted in mainchain or orphan, update peer height.
		case umsg := <-s.peerHeightsUpdate:
			s.handleUpdatePeerHeights(state, umsg)

		// Peer to ban.
		case p := <-s.banPeers:
			s.handleBanPeerMsg(state, p)

		// New inventory to potentially be relayed to other peers.
		case invMsg := <-s.relayInv:
			s.handleRelayInvMsg(state, invMsg)

		// Message to broadcast to all connected peers except those
		// which are excluded by the message.
		case bmsg := <-s.broadcast:
			s.handleBroadcastMsg(state, &bmsg)

		case qmsg := <-s.query:
			s.handleQuery(state, qmsg)

		case <-s.quit:
			// Disconnect all peers on MemPoolNode shutdown.
			state.forAllPeers(func(sp *serverPeer) {
				common.Log.Tracef("Shutdown peer %s", sp)
				sp.Disconnect()
			})
			break out
		}
	}

	s.connManager.Stop()
	s.syncManager.Stop()
	s.addrManager.Stop()

	// Drain channels before exiting so nothing is left waiting around
	// to send.
cleanup:
	for {
		select {
		case <-s.newPeers:
		case <-s.donePeers:
		case <-s.peerHeightsUpdate:
		case <-s.relayInv:
		case <-s.broadcast:
		case <-s.query:
		default:
			break cleanup
		}
	}
	s.wg.Done()
	common.Log.Tracef("Peer handler done")
}

// AddPeer adds a new peer that has already been connected to the MemPoolNode.
func (s *MemPoolNode) AddPeer(sp *serverPeer) {
	s.newPeers <- sp
}

// BanPeer bans a peer that has already been connected to the MemPoolNode by ip.
func (s *MemPoolNode) BanPeer(sp *serverPeer) {
	s.banPeers <- sp
}

// RelayInventory relays the passed inventory vector to all connected peers
// that are not already known to have it.
func (s *MemPoolNode) RelayInventory(invVect *wire.InvVect, data interface{}) {
	s.relayInv <- relayMsg{invVect: invVect, data: data}
}

// BroadcastMessage sends msg to all peers currently connected to the MemPoolNode
// except those in the passed peers to exclude.
func (s *MemPoolNode) BroadcastMessage(msg wire.Message, exclPeers ...*serverPeer) {
	// XXX: Need to determine if this is an alert that has already been
	// broadcast and refrain from broadcasting again.
	bmsg := broadcastMsg{message: msg, excludePeers: exclPeers}
	s.broadcast <- bmsg
}

// ConnectedCount returns the number of currently connected peers.
func (s *MemPoolNode) ConnectedCount() int32 {
	replyChan := make(chan int32)

	s.query <- getConnCountMsg{reply: replyChan}

	return <-replyChan
}

// OutboundGroupCount returns the number of peers connected to the given
// outbound group key.
func (s *MemPoolNode) OutboundGroupCount(key string) int {
	replyChan := make(chan int)
	s.query <- getOutboundGroup{key: key, reply: replyChan}
	return <-replyChan
}

// AddBytesSent adds the passed number of bytes to the total bytes sent counter
// for the MemPoolNode.  It is safe for concurrent access.
func (s *MemPoolNode) AddBytesSent(bytesSent uint64) {
	atomic.AddUint64(&s.bytesSent, bytesSent)
}

// AddBytesReceived adds the passed number of bytes to the total bytes received
// counter for the MemPoolNode.  It is safe for concurrent access.
func (s *MemPoolNode) AddBytesReceived(bytesReceived uint64) {
	atomic.AddUint64(&s.bytesReceived, bytesReceived)
}

// NetTotals returns the sum of all bytes received and sent across the network
// for all peers.  It is safe for concurrent access.
func (s *MemPoolNode) NetTotals() (uint64, uint64) {
	return atomic.LoadUint64(&s.bytesReceived),
		atomic.LoadUint64(&s.bytesSent)
}

// UpdatePeerHeights updates the heights of all peers who have have announced
// the latest connected main chain block, or a recognized orphan. These height
// updates allow us to dynamically refresh peer heights, ensuring sync peer
// selection has access to the latest block heights for each peer.
func (s *MemPoolNode) UpdatePeerHeights(latestBlkHash *chainhash.Hash, latestHeight int32, updateSource *peer.Peer) {
	s.peerHeightsUpdate <- updatePeerHeightsMsg{
		newHash:    latestBlkHash,
		newHeight:  latestHeight,
		originPeer: updateSource,
	}
}

// rebroadcastHandler keeps track of user submitted inventories that we have
// sent out but have not yet made it into a block. We periodically rebroadcast
// them in case our peers restarted or otherwise lost track of them.
func (s *MemPoolNode) rebroadcastHandler() {
	// Wait 5 min before first tx rebroadcast.
	timer := time.NewTimer(5 * time.Minute)
	pendingInvs := make(map[wire.InvVect]interface{})

out:
	for {
		select {
		case riv := <-s.modifyRebroadcastInv:
			switch msg := riv.(type) {
			// Incoming InvVects are added to our map of RPC txs.
			case broadcastInventoryAdd:
				pendingInvs[*msg.invVect] = msg.data

			// When an InvVect has been added to a block, we can
			// now remove it, if it was present.
			case broadcastInventoryDel:
				delete(pendingInvs, *msg)
			}

		case <-timer.C:
			// Any inventory we have has not made it into a block
			// yet. We periodically resubmit them until they have.
			for iv, data := range pendingInvs {
				ivCopy := iv
				s.RelayInventory(&ivCopy, data)
			}

			// Process at a random time up to 30mins (in seconds)
			// in the future.
			timer.Reset(time.Second *
				time.Duration(randomUint16Number(1800)))

		case <-s.quit:
			break out
		}
	}

	timer.Stop()

	// Drain channels before exiting so nothing is left waiting around
	// to send.
cleanup:
	for {
		select {
		case <-s.modifyRebroadcastInv:
		default:
			break cleanup
		}
	}
	s.wg.Done()
}

// Start begins accepting connections from peers.
func (s *MemPoolNode) Start() {
	// Already started?
	if atomic.AddInt32(&s.started, 1) != 1 {
		return
	}

	common.Log.Trace("Starting MemPoolNode")

	// Server startup time. Used for the uptime command for uptime calculation.
	s.startupTime = time.Now().Unix()

	// Start the peer handler which in turn starts the address and block
	// managers.
	s.wg.Add(1)
	go s.peerHandler()

	if s.nat != nil {
		s.wg.Add(1)
		go s.upnpUpdateThread()
	}

	// if !cfg.DisableRPC {
	s.wg.Add(1)

	// Start the rebroadcastHandler, which ensures user tx received by
	// the RPC MemPoolNode are rebroadcast until being included in a block.
	go s.rebroadcastHandler()

	// 	s.rpcServer.cfg.StartupTime = s.startupTime
	// 	s.rpcServer.Start()
	// }

	// // Start the CPU miner if generation is enabled.
	// if cfg.Generate {
	// 	s.cpuMiner.Start()
	// }
}

// Stop gracefully shuts down the MemPoolNode by stopping and disconnecting all
// peers and the main listener.
func (s *MemPoolNode) Stop() error {
	// Make sure this only happens once.
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		common.Log.Infof("Server is already in the process of shutting down")
		return nil
	}

	common.Log.Warnf("Server shutting down")

	// Stop the CPU miner if needed
	// s.cpuMiner.Stop()

	// Shutdown the RPC MemPoolNode if it's not disabled.
	// if !cfg.DisableRPC {
	// 	s.rpcServer.Stop()
	// }

	// Save fee estimator state in the database.
	// s.db.Update(func(tx database.Tx) error {
	// 	metadata := tx.Metadata()
	// 	metadata.Put(mempool.EstimateFeeDatabaseKey, s.feeEstimator.Save())

	// 	return nil
	// })

	s.db.Close()

	// Signal the remaining goroutines to quit.
	close(s.quit)
	return nil
}

// WaitForShutdown blocks until the main listener and peer handlers are stopped.
func (s *MemPoolNode) WaitForShutdown() {
	s.wg.Wait()
}

// ScheduleShutdown schedules a MemPoolNode shutdown after the specified duration.
// It also dynamically adjusts how often to warn the MemPoolNode is going down based
// on remaining duration.
func (s *MemPoolNode) ScheduleShutdown(duration time.Duration) {
	// Don't schedule shutdown more than once.
	if atomic.AddInt32(&s.shutdownSched, 1) != 1 {
		return
	}
	common.Log.Warnf("Server shutdown in %v", duration)
	go func() {
		remaining := duration
		tickDuration := dynamicTickDuration(remaining)
		done := time.After(remaining)
		ticker := time.NewTicker(tickDuration)
	out:
		for {
			select {
			case <-done:
				ticker.Stop()
				s.Stop()
				break out
			case <-ticker.C:
				remaining = remaining - tickDuration
				if remaining < time.Second {
					continue
				}

				// Change tick duration dynamically based on remaining time.
				newDuration := dynamicTickDuration(remaining)
				if tickDuration != newDuration {
					tickDuration = newDuration
					ticker.Stop()
					ticker = time.NewTicker(tickDuration)
				}
				common.Log.Warnf("Server shutdown in %v", remaining)
			}
		}
	}()
}

// parseListeners determines whether each listen address is IPv4 and IPv6 and
// returns a slice of appropriate net.Addrs to listen on with TCP. It also
// properly detects addresses which apply to "all interfaces" and adds the
// address as both IPv4 and IPv6.
func parseListeners(addrs []string) ([]net.Addr, error) {
	netAddrs := make([]net.Addr, 0, len(addrs)*2)
	for _, addr := range addrs {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			// Shouldn't happen due to already being normalized.
			return nil, err
		}

		// Empty host or host of * on plan9 is both IPv4 and IPv6.
		if host == "" || (host == "*" && runtime.GOOS == "plan9") {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
			continue
		}

		// Strip IPv6 zone id if present since net.ParseIP does not
		// handle it.
		zoneIndex := strings.LastIndex(host, "%")
		if zoneIndex > 0 {
			host = host[:zoneIndex]
		}

		// Parse the IP.
		ip := net.ParseIP(host)
		if ip == nil {
			return nil, fmt.Errorf("'%s' is not a valid IP address", host)
		}

		// To4 returns nil when the IP is not an IPv4 address, so use
		// this determine the address type.
		if ip.To4() == nil {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp6", addr: addr})
		} else {
			netAddrs = append(netAddrs, simpleAddr{net: "tcp4", addr: addr})
		}
	}
	return netAddrs, nil
}

func (s *MemPoolNode) upnpUpdateThread() {
	// Go off immediately to prevent code duplication, thereafter we renew
	// lease every 15 minutes.
	timer := time.NewTimer(0 * time.Second)
	lport, _ := strconv.ParseInt(activeNetParams.DefaultPort, 10, 16)
	first := true
out:
	for {
		select {
		case <-timer.C:
			// TODO: pick external port  more cleverly
			// TODO: know which ports we are listening to on an external net.
			// TODO: if specific listen port doesn't work then ask for wildcard
			// listen port?
			// XXX this assumes timeout is in seconds.
			listenPort, err := s.nat.AddPortMapping("tcp", int(lport), int(lport),
				"btcd listen port", 20*60)
			if err != nil {
				common.Log.Warnf("can't add UPnP port mapping: %v", err)
			}
			if first && err == nil {
				// TODO: look this up periodically to see if upnp domain changed
				// and so did ip.
				externalip, err := s.nat.GetExternalAddress()
				if err != nil {
					common.Log.Warnf("UPnP can't get external address: %v", err)
					continue out
				}
				na := wire.NetAddressV2FromBytes(time.Now(), s.services,
					externalip, uint16(listenPort))
				err = s.addrManager.AddLocalAddress(na, addrmgr.UpnpPrio)
				if err != nil {
					// XXX DeletePortMapping?
				}
				common.Log.Warnf("Successfully bound via UPnP to %s", addrmgr.NetAddressKey(na))
				first = false
			}
			timer.Reset(time.Minute * 15)
		case <-s.quit:
			break out
		}
	}

	timer.Stop()

	if err := s.nat.DeletePortMapping("tcp", int(lport), int(lport)); err != nil {
		common.Log.Warnf("unable to remove UPnP port mapping: %v", err)
	} else {
		common.Log.Debugf("successfully disestablished UPnP port mapping")
	}

	s.wg.Done()
}

// newServer returns a new MemPoolNode configured to listen on addr for the
// bitcoin network type specified by chainParams.  Use start to begin accepting
// connections from peers.
func newServer(listenAddrs, agentBlacklist, agentWhitelist []string,
	db *badger.DB, chainParams *chaincfg.Params,
	indexManager localCommon.IndexManager,
	interrupt <-chan struct{}) (*MemPoolNode, error) {

	services := defaultServices
	if _cfg.NoPeerBloomFilters {
		services &^= wire.SFNodeBloom
	}
	if _cfg.NoCFilters {
		services &^= wire.SFNodeCF
	}
	if _cfg.Prune != 0 {
		services &^= wire.SFNodeNetwork
	}

	amgr := addrmgr.New(_cfg.DataDir, btcdLookup)

	var listeners []net.Listener
	var nat NAT
	if !_cfg.DisableListen {
		var err error
		listeners, nat, err = initListeners(amgr, listenAddrs, services)
		if err != nil {
			return nil, err
		}
		if len(listeners) == 0 {
			return nil, errors.New("no valid listen address")
		}
	}

	if len(agentBlacklist) > 0 {
		common.Log.Infof("User-agent blacklist %s", agentBlacklist)
	}
	if len(agentWhitelist) > 0 {
		common.Log.Infof("User-agent whitelist %s", agentWhitelist)
	}

	s := MemPoolNode{
		chainParams:          chainParams,
		addrManager:          amgr,
		newPeers:             make(chan *serverPeer, _cfg.MaxPeers),
		donePeers:            make(chan *serverPeer, _cfg.MaxPeers),
		banPeers:             make(chan *serverPeer, _cfg.MaxPeers),
		query:                make(chan interface{}),
		relayInv:             make(chan relayMsg, _cfg.MaxPeers),
		broadcast:            make(chan broadcastMsg, _cfg.MaxPeers),
		quit:                 make(chan struct{}),
		modifyRebroadcastInv: make(chan interface{}),
		peerHeightsUpdate:    make(chan updatePeerHeightsMsg),
		nat:                  nat,
		db:                   db,
		//timeSource:           localCommon.NewMedianTime(),
		services:             services,
		sigCache:             txscript.NewSigCache(_cfg.SigCacheMaxSize),
		hashCache:            txscript.NewHashCache(_cfg.SigCacheMaxSize),
		cfCheckptCaches:      make(map[wire.FilterType][]cfHeaderKV),
		agentBlacklist:       agentBlacklist,
		agentWhitelist:       agentWhitelist,
	}

	s.indexer = indexManager

	// Search for a FeeEstimator state in the database. If none can be found
	// or if it cannot be loaded, create a new one.
	// db.Update(func(tx database.Tx) error {
	// 	metadata := tx.Metadata()
	// 	feeEstimationData := metadata.Get(mempool.EstimateFeeDatabaseKey)
	// 	if feeEstimationData != nil {
	// 		// delete it from the database so that we don't try to restore the
	// 		// same thing again somehow.
	// 		metadata.Delete(mempool.EstimateFeeDatabaseKey)

	// 		// If there is an error, log it and make a new fee estimator.
	// 		var err error
	// 		s.feeEstimator, err = mempool.RestoreFeeEstimator(feeEstimationData)

	// 		if err != nil {
	// 			common.Log.Errorf("Failed to restore fee estimator %v", err)
	// 		}
	// 	}

	// 	return nil
	// })

	// If no feeEstimator has been found, or if the one that has been found
	// is behind somehow, create a new one and start over.
	if s.feeEstimator == nil || s.feeEstimator.LastKnownHeight() != s.indexer.BestSnapshot().Height {
		s.feeEstimator = mempool.NewFeeEstimator(
			mempool.DefaultEstimateFeeMaxRollback,
			mempool.DefaultEstimateFeeMinRegisteredBlocks)
	}

	txC := mempool.Config{
		Policy: mempool.Policy{
			DisableRelayPriority: _cfg.NoRelayPriority,
			AcceptNonStd:         _cfg.RelayNonStd,
			FreeTxRelayLimit:     _cfg.FreeTxRelayLimit,
			MaxOrphanTxs:         _cfg.MaxOrphanTxs,
			MaxOrphanTxSize:      defaultMaxOrphanTxSize,
			MaxSigOpCostPerTx:    localCommon.MaxBlockSigOpsCost / 4,
			MinRelayTxFee:        _cfg.minRelayTxFee,
			MaxTxVersion:         2,
			RejectReplacement:    _cfg.RejectReplacement,
		},
		ChainParams:    chainParams,
		FetchUtxoView:  s.indexer.FetchUtxoView,
		BestHeight:     func() int32 { return s.indexer.BestSnapshot().Height },
		MedianTimePast: func() time.Time { return s.indexer.BestSnapshot().MedianTime },
		CalcSequenceLock: func(tx *btcutil.Tx, view *localCommon.UtxoViewpoint) (*localCommon.SequenceLock, error) {
			return s.indexer.CalcSequenceLock(tx, view, true)
		},
		IsDeploymentActive: s.indexer.IsDeploymentActive,
		SigCache:           s.sigCache,
		HashCache:          s.hashCache,
		//AddrIndex:          s.addrIndex,
		FeeEstimator: s.feeEstimator,
	}
	s.txMemPool = mempool.New(&txC)

	var err error
	s.syncManager, err = netsync.New(&netsync.Config{
		PeerNotifier:       &s,
		Chain:              s.indexer,
		TxMemPool:          s.txMemPool,
		ChainParams:        s.chainParams,
		DisableCheckpoints: _cfg.DisableCheckpoints,
		MaxPeers:           _cfg.MaxPeers,
		FeeEstimator:       s.feeEstimator,
	})
	if err != nil {
		return nil, err
	}

	// Create the mining policy and block template generator based on the
	// configuration options.
	//
	// NOTE: The CPU miner relies on the mempool, so the mempool has to be
	// created before calling the function to create the CPU miner.
	// policy := mining.Policy{
	// 	BlockMinWeight:    cfg.BlockMinWeight,
	// 	BlockMaxWeight:    cfg.BlockMaxWeight,
	// 	BlockMinSize:      cfg.BlockMinSize,
	// 	BlockMaxSize:      cfg.BlockMaxSize,
	// 	BlockPrioritySize: cfg.BlockPrioritySize,
	// 	TxMinFreeFee:      cfg.minRelayTxFee,
	// }
	// blockTemplateGenerator := mining.NewBlkTmplGenerator(&policy,
	// 	s.chainParams, s.txMemPool, s.chain, s.timeSource,
	// 	s.sigCache, s.hashCache)
	// s.cpuMiner = cpuminer.New(&cpuminer.Config{
	// 	ChainParams:            chainParams,
	// 	BlockTemplateGenerator: blockTemplateGenerator,
	// 	MiningAddrs:            cfg.miningAddrs,
	// 	ProcessBlock:           s.syncManager.ProcessBlock,
	// 	ConnectedCount:         s.ConnectedCount,
	// 	IsCurrent:              s.syncManager.IsCurrent,
	// })

	// Only setup a function to return new addresses to connect to when
	// not running in connect-only mode.  The simulation network is always
	// in connect-only mode since it is only intended to connect to
	// specified peers and actively avoid advertising and connecting to
	// discovered peers in order to prevent it from becoming a public test
	// network.
	var newAddressFunc func() (net.Addr, error)
	if !_cfg.SimNet && len(_cfg.ConnectPeers) == 0 {
		newAddressFunc = func() (net.Addr, error) {
			for tries := 0; tries < 100; tries++ {
				addr := s.addrManager.GetAddress()
				if addr == nil {
					break
				}

				// Address will not be invalid, local or unroutable
				// because addrmanager rejects those on addition.
				// Just check that we don't already have an address
				// in the same group so that we are not connecting
				// to the same network segment at the expense of
				// others.
				key := addrmgr.GroupKey(addr.NetAddress())
				if s.OutboundGroupCount(key) != 0 {
					continue
				}

				// only allow recent nodes (10mins) after we failed 30
				// times
				if tries < 30 && time.Since(addr.LastAttempt()) < 10*time.Minute {
					continue
				}

				// allow nondefault ports after 50 failed tries.
				if tries < 50 && fmt.Sprintf("%d", addr.NetAddress().Port) !=
					activeNetParams.DefaultPort {
					continue
				}

				// Mark an attempt for the valid address.
				s.addrManager.Attempt(addr.NetAddress())

				addrString := addrmgr.NetAddressKey(addr.NetAddress())
				return addrStringToNetAddr(addrString)
			}

			return nil, errors.New("no valid connect address")
		}
	}

	// Create a connection manager.
	targetOutbound := defaultTargetOutbound
	if _cfg.MaxPeers < targetOutbound {
		targetOutbound = _cfg.MaxPeers
	}
	cmgr, err := connmgr.New(&connmgr.Config{
		Listeners:      listeners,
		OnAccept:       s.inboundPeerConnected,
		RetryDuration:  connectionRetryInterval,
		TargetOutbound: uint32(targetOutbound),
		Dial:           btcdDial,
		OnConnection:   s.outboundPeerConnected,
		GetNewAddress:  newAddressFunc,
	})
	if err != nil {
		return nil, err
	}
	s.connManager = cmgr

	// Start up persistent peers.
	permanentPeers := _cfg.ConnectPeers
	if len(permanentPeers) == 0 {
		permanentPeers = _cfg.AddPeers
	}
	for _, addr := range permanentPeers {
		netAddr, err := addrStringToNetAddr(addr)
		if err != nil {
			return nil, err
		}

		go s.connManager.Connect(&connmgr.ConnReq{
			Addr:      netAddr,
			Permanent: true,
		})
	}

	//if !cfg.DisableRPC {
	// Setup listeners for the configured RPC listen addresses and
	// TLS settings.
	// rpcListeners, err := setupRPCListeners()
	// if err != nil {
	// 	return nil, err
	// }
	// if len(rpcListeners) == 0 {
	// 	return nil, errors.New("RPCS: No valid listen address")
	// }

	// s.rpcServer, err = newRPCServer(&rpcserverConfig{
	// 	Listeners:    rpcListeners,
	// 	StartupTime:  s.startupTime,
	// 	ConnMgr:      &rpcConnManager{&s},
	// 	SyncMgr:      &rpcSyncMgr{&s, s.syncManager},
	// 	TimeSource:   s.timeSource,
	// 	Chain:        s.chain,
	// 	ChainParams:  chainParams,
	// 	DB:           db,
	// 	TxMemPool:    s.txMemPool,
	// 	Generator:    blockTemplateGenerator,
	// 	CPUMiner:     s.cpuMiner,
	// 	TxIndex:      s.txIndex,
	// 	AddrIndex:    s.addrIndex,
	// 	CfIndex:      s.cfIndex,
	// 	FeeEstimator: s.feeEstimator,
	// })
	// if err != nil {
	// 	return nil, err
	// }

	// Signal process shutdown when the RPC MemPoolNode requests it.
	// go func() {
	// 	<-s.rpcServer.RequestedProcessShutdown()
	// 	shutdownRequestChannel <- struct{}{}
	// }()
	//}

	return &s, nil
}

// initListeners initializes the configured net listeners and adds any bound
// addresses to the address manager. Returns the listeners and a NAT interface,
// which is non-nil if UPnP is in use.
func initListeners(amgr *addrmgr.AddrManager, listenAddrs []string, services wire.ServiceFlag) ([]net.Listener, NAT, error) {
	// Listen for TCP connections at the configured addresses
	netAddrs, err := parseListeners(listenAddrs)
	if err != nil {
		return nil, nil, err
	}

	listeners := make([]net.Listener, 0, len(netAddrs))
	for _, addr := range netAddrs {
		listener, err := net.Listen(addr.Network(), addr.String())
		if err != nil {
			common.Log.Warnf("Can't listen on %s: %v", addr, err)
			continue
		}
		listeners = append(listeners, listener)
	}

	var nat NAT
	if len(_cfg.ExternalIPs) != 0 {
		defaultPort, err := strconv.ParseUint(activeNetParams.DefaultPort, 10, 16)
		if err != nil {
			common.Log.Errorf("Can not parse default port %s for active chain: %v",
				activeNetParams.DefaultPort, err)
			return nil, nil, err
		}

		for _, sip := range _cfg.ExternalIPs {
			eport := uint16(defaultPort)
			host, portstr, err := net.SplitHostPort(sip)
			if err != nil {
				// no port, use default.
				host = sip
			} else {
				port, err := strconv.ParseUint(portstr, 10, 16)
				if err != nil {
					common.Log.Warnf("Can not parse port from %s for "+
						"externalip: %v", sip, err)
					continue
				}
				eport = uint16(port)
			}
			na, err := amgr.HostToNetAddress(host, eport, services)
			if err != nil {
				common.Log.Warnf("Not adding %s as externalip: %v", sip, err)
				continue
			}

			err = amgr.AddLocalAddress(na, addrmgr.ManualPrio)
			if err != nil {
				common.Log.Warnf("Skipping specified external IP: %v", err)
			}
		}
	} else {
		if _cfg.Upnp {
			var err error
			nat, err = Discover()
			if err != nil {
				common.Log.Warnf("Can't discover upnp: %v", err)
			}
			// nil nat here is fine, just means no upnp on network.
		}

		// Add bound addresses to address manager to be advertised to peers.
		for _, listener := range listeners {
			addr := listener.Addr().String()
			err := addLocalAddress(amgr, addr, services)
			if err != nil {
				common.Log.Warnf("Skipping bound address %s: %v", addr, err)
			}
		}
	}

	return listeners, nat, nil
}

// addrStringToNetAddr takes an address in the form of 'host:port' and returns
// a net.Addr which maps to the original address with any host names resolved
// to IP addresses.  It also handles tor addresses properly by returning a
// net.Addr that encapsulates the address.
func addrStringToNetAddr(addr string) (net.Addr, error) {
	host, strPort, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, err
	}

	// Skip if host is already an IP address.
	if ip := net.ParseIP(host); ip != nil {
		return &net.TCPAddr{
			IP:   ip,
			Port: port,
		}, nil
	}

	// Tor addresses cannot be resolved to an IP, so just return an onion
	// address instead.
	if strings.HasSuffix(host, ".onion") {
		if _cfg.NoOnion {
			return nil, errors.New("tor has been disabled")
		}

		return &onionAddr{addr: addr}, nil
	}

	// Attempt to look up an IP address associated with the parsed host.
	ips, err := btcdLookup(host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no addresses found for %s", host)
	}

	return &net.TCPAddr{
		IP:   ips[0],
		Port: port,
	}, nil
}

// addLocalAddress adds an address that this node is listening on to the
// address manager so that it may be relayed to peers.
func addLocalAddress(addrMgr *addrmgr.AddrManager, addr string, services wire.ServiceFlag) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return err
	}

	if ip := net.ParseIP(host); ip != nil && ip.IsUnspecified() {
		// If bound to unspecified address, advertise all local interfaces
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			ifaceIP, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}

			// If bound to 0.0.0.0, do not add IPv6 interfaces and if bound to
			// ::, do not add IPv4 interfaces.
			if (ip.To4() == nil) != (ifaceIP.To4() == nil) {
				continue
			}

			netAddr := wire.NetAddressV2FromBytes(
				time.Now(), services, ifaceIP, uint16(port),
			)
			addrMgr.AddLocalAddress(netAddr, addrmgr.BoundPrio)
		}
	} else {
		netAddr, err := addrMgr.HostToNetAddress(host, uint16(port), services)
		if err != nil {
			return err
		}

		addrMgr.AddLocalAddress(netAddr, addrmgr.BoundPrio)
	}

	return nil
}

// dynamicTickDuration is a convenience function used to dynamically choose a
// tick duration based on remaining time.  It is primarily used during
// MemPoolNode shutdown to make shutdown warnings more frequent as the shutdown time
// approaches.
func dynamicTickDuration(remaining time.Duration) time.Duration {
	switch {
	case remaining <= time.Second*5:
		return time.Second
	case remaining <= time.Second*15:
		return time.Second * 5
	case remaining <= time.Minute:
		return time.Second * 15
	case remaining <= time.Minute*5:
		return time.Minute
	case remaining <= time.Minute*15:
		return time.Minute * 5
	case remaining <= time.Hour:
		return time.Minute * 15
	}
	return time.Hour
}

// isWhitelisted returns whether the IP address is included in the whitelisted
// networks and IPs.
func isWhitelisted(addr net.Addr) bool {
	if len(_cfg.whitelists) == 0 {
		return false
	}

	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		common.Log.Warnf("Unable to SplitHostPort on '%s': %v", addr, err)
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		common.Log.Warnf("Unable to parse IP '%s'", addr)
		return false
	}

	for _, ipnet := range _cfg.whitelists {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

// checkpointSorter implements sort.Interface to allow a slice of checkpoints to
// be sorted.
type checkpointSorter []chaincfg.Checkpoint

// Len returns the number of checkpoints in the slice.  It is part of the
// sort.Interface implementation.
func (s checkpointSorter) Len() int {
	return len(s)
}

// Swap swaps the checkpoints at the passed indices.  It is part of the
// sort.Interface implementation.
func (s checkpointSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less returns whether the checkpoint with index i should sort before the
// checkpoint with index j.  It is part of the sort.Interface implementation.
func (s checkpointSorter) Less(i, j int) bool {
	return s[i].Height < s[j].Height
}

// mergeCheckpoints returns two slices of checkpoints merged into one slice
// such that the checkpoints are sorted by height.  In the case the additional
// checkpoints contain a checkpoint with the same height as a checkpoint in the
// default checkpoints, the additional checkpoint will take precedence and
// overwrite the default one.
func mergeCheckpoints(defaultCheckpoints, additional []chaincfg.Checkpoint) []chaincfg.Checkpoint {
	// Create a map of the additional checkpoints to remove duplicates while
	// leaving the most recently-specified checkpoint.
	extra := make(map[int32]chaincfg.Checkpoint)
	for _, checkpoint := range additional {
		extra[checkpoint.Height] = checkpoint
	}

	// Add all default checkpoints that do not have an override in the
	// additional checkpoints.
	numDefault := len(defaultCheckpoints)
	checkpoints := make([]chaincfg.Checkpoint, 0, numDefault+len(extra))
	for _, checkpoint := range defaultCheckpoints {
		if _, exists := extra[checkpoint.Height]; !exists {
			checkpoints = append(checkpoints, checkpoint)
		}
	}

	// Append the additional checkpoints and return the sorted results.
	for _, checkpoint := range extra {
		checkpoints = append(checkpoints, checkpoint)
	}
	sort.Sort(checkpointSorter(checkpoints))
	return checkpoints
}

// HasUndesiredUserAgent determines whether the MemPoolNode should continue to pursue
// a connection with this peer based on its advertised user agent. It performs
// the following steps:
// 1) Reject the peer if it contains a blacklisted agent.
// 2) If no whitelist is provided, accept all user agents.
// 3) Accept the peer if it contains a whitelisted agent.
// 4) Reject all other peers.
func (sp *serverPeer) HasUndesiredUserAgent(blacklistedAgents,
	whitelistedAgents []string) bool {

	agent := sp.UserAgent()

	// First, if peer's user agent contains any blacklisted substring, we
	// will ignore the connection request.
	for _, blacklistedAgent := range blacklistedAgents {
		if strings.Contains(agent, blacklistedAgent) {
			common.Log.Debugf("Ignoring peer %s, user agent "+
				"contains blacklisted user agent: %s", sp,
				agent)
			return true
		}
	}

	// If no whitelist is provided, we will accept all user agents.
	if len(whitelistedAgents) == 0 {
		return false
	}

	// Peer's user agent passed blacklist. Now check to see if it contains
	// one of our whitelisted user agents, if so accept.
	for _, whitelistedAgent := range whitelistedAgents {
		if strings.Contains(agent, whitelistedAgent) {
			return false
		}
	}

	// Otherwise, the peer's user agent was not included in our whitelist.
	// Ignore just in case it could stall the initial block download.
	common.Log.Debugf("Ignoring peer %s, user agent: %s not found in "+
		"whitelist", sp, agent)

	return true
}

// genCertPair generates a key/cert pair to the paths provided.
func genCertPair(certFile, keyFile string) error {
	common.Log.Infof("Generating TLS certificates...")

	org := "btcd autogenerated cert"
	validUntil := time.Now().Add(10 * 365 * 24 * time.Hour)
	cert, key, err := btcutil.NewTLSCertPair(org, validUntil, nil)
	if err != nil {
		return err
	}

	// Write cert and key files.
	if err = os.WriteFile(certFile, cert, 0666); err != nil {
		return err
	}
	if err = os.WriteFile(keyFile, key, 0600); err != nil {
		os.Remove(certFile)
		return err
	}

	common.Log.Infof("Done generating TLS certificates")
	return nil
}
