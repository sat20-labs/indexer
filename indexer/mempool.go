package indexer

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/btcsuite/btcd/peer"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/config"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

const mempoolRetryMaxPasses = 64

// MiniMemPool is a deliberately small mempool view. It tracks confirmed UTXOs
// spent by unconfirmed transactions and only exposes new outputs that can be
// proven to be plain sats. Unconfirmed asset outputs are classified as
// non-plain/unknown and are never recursively rebuilt.
type UserUtxoInMempool struct {
	SpentUtxo               map[string]*common.TxOutput
	UnconfirmedPlainUtxoMap map[string]*common.TxOutput
}

type MiniMemPool struct {
	txMap        map[string]*wire.MsgTx
	spentUtxoMap map[string]*common.TxOutput

	// spentByOutpoint owns the spend admission for an input. inputsByTx and
	// childrenByTx make replacement/conflict cleanup deterministic.
	spentByOutpoint map[string]string
	inputsByTx      map[string][]string
	childrenByTx    map[string]map[string]struct{}

	// confirmedSpent keeps a just-mined input unavailable until the confirmed
	// index has caught up and no longer returns the old UTXO.
	confirmedSpent map[string]struct{}

	// knownPlainUtxoMap keeps plain outputs even after they are spent by a
	// child transaction, so the child can be retried/classified without
	// recursively rebuilding its parent. The per-address map below only keeps
	// currently available plain outputs.
	knownPlainUtxoMap map[string]*common.TxOutput
	utxoStateMap      map[string]mempoolUtxoState
	classifiedTxMap   map[string]bool
	addrUtxoMap       map[string]*UserUtxoInMempool

	// Serialize transaction classification and all graph mutations.
	processingMutex sync.Mutex
	mutex           sync.RWMutex

	// indexerReadBarrier prevents mempool transaction classification from
	// reading indexer state while indexer state is committed/reloaded/closed.
	indexerReadBarrier sync.RWMutex

	// Lifecycle state. Stop closes admission, disconnects the peer and waits
	// for all owned workers instead of relying on a fixed sleep.
	lifecycleMutex sync.Mutex
	running        bool
	syncing        bool
	stopChan       chan struct{}
	workerWG       sync.WaitGroup
	peer           *peer.Peer
	lastSyncTime   int64
}

func NewMiniMemPool() *MiniMemPool {
	p := &MiniMemPool{}
	p.init()
	return p
}

func (p *MiniMemPool) resetStateLocked() {
	p.txMap = make(map[string]*wire.MsgTx)
	p.spentUtxoMap = make(map[string]*common.TxOutput)
	p.spentByOutpoint = make(map[string]string)
	p.inputsByTx = make(map[string][]string)
	p.childrenByTx = make(map[string]map[string]struct{})
	p.confirmedSpent = make(map[string]struct{})
	p.knownPlainUtxoMap = make(map[string]*common.TxOutput)
	p.utxoStateMap = make(map[string]mempoolUtxoState)
	p.classifiedTxMap = make(map[string]bool)
	p.addrUtxoMap = make(map[string]*UserUtxoInMempool)
}

func (p *MiniMemPool) init() {
	p.processingMutex.Lock()
	defer p.processingMutex.Unlock()
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.resetStateLocked()
}

func (p *MiniMemPool) Start(cfg *config.Bitcoin) {
	p.lifecycleMutex.Lock()
	if p.running {
		p.lifecycleMutex.Unlock()
		return
	}
	p.running = true
	p.stopChan = make(chan struct{})
	stop := p.stopChan
	p.lifecycleMutex.Unlock()

	netParam := instance.GetChainParam()
	addr := fmt.Sprintf("%s:%s", cfg.Host, netParam.DefaultPort)
	p.startWorker(stop, func() { p.listenP2PTx(addr, stop) })
	p.startWorker(stop, func() { p.traceThread(stop) })
	p.scheduleSync(true)
}

func (p *MiniMemPool) startWorker(stop chan struct{}, fn func()) bool {
	p.lifecycleMutex.Lock()
	if !p.running || p.stopChan != stop {
		p.lifecycleMutex.Unlock()
		return false
	}
	p.workerWG.Add(1)
	p.lifecycleMutex.Unlock()

	go func() {
		defer p.workerWG.Done()
		fn()
	}()
	return true
}

func (p *MiniMemPool) traceThread(stop <-chan struct{}) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			p.mutex.RLock()
			availablePlain := 0
			for _, user := range p.addrUtxoMap {
				availablePlain += len(user.UnconfirmedPlainUtxoMap)
			}
			common.Log.Infof("mempool: tx=%d spent=%d confirmed-spent=%d known-plain=%d available-plain=%d",
				len(p.txMap), len(p.spentByOutpoint), len(p.confirmedSpent), len(p.knownPlainUtxoMap), availablePlain)
			p.mutex.RUnlock()
		}
	}
}

// Stop closes all new work, disconnects P2P and waits until every owned worker
// and in-flight classifier has exited before indexer databases may be closed.
func (p *MiniMemPool) Stop() {
	p.lifecycleMutex.Lock()
	if !p.running {
		p.lifecycleMutex.Unlock()
		return
	}
	p.running = false
	stop := p.stopChan
	peerConn := p.peer
	p.stopChan = nil
	p.peer = nil
	if stop != nil {
		close(stop)
	}
	p.lifecycleMutex.Unlock()

	if peerConn != nil {
		peerConn.Disconnect()
	}
	p.workerWG.Wait()

	// Peer callbacks are owned by btcd/peer, not workerWG. Drain any callback
	// that entered classification before Disconnect returned.
	p.processingMutex.Lock()
	p.processingMutex.Unlock()

	p.lifecycleMutex.Lock()
	p.syncing = false
	p.lifecycleMutex.Unlock()
}

func (p *MiniMemPool) pauseIndexerReads() {
	p.indexerReadBarrier.Lock()
}

func (p *MiniMemPool) resumeIndexerReads() {
	p.indexerReadBarrier.Unlock()
}

func (p *MiniMemPool) enterIndexerRead() {
	p.indexerReadBarrier.RLock()
}

func (p *MiniMemPool) leaveIndexerRead() {
	p.indexerReadBarrier.RUnlock()
}

func (p *MiniMemPool) scheduleSync(initial bool) {
	p.lifecycleMutex.Lock()
	if !p.running || p.syncing || p.stopChan == nil {
		p.lifecycleMutex.Unlock()
		return
	}
	p.syncing = true
	stop := p.stopChan
	p.workerWG.Add(1)
	p.lifecycleMutex.Unlock()

	go func() {
		defer p.workerWG.Done()
		succeeded := p.syncMempoolFromRPC(stop, initial)
		p.lifecycleMutex.Lock()
		if p.stopChan == stop {
			p.syncing = false
			if succeeded {
				p.lastSyncTime = time.Now().Unix()
			}
		}
		p.lifecycleMutex.Unlock()
	}()
}

func (p *MiniMemPool) shouldStop(stop <-chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

// syncMempoolFromRPC reconciles membership without recursively fetching parent
// transactions. It only removes transactions that existed when the snapshot
// started, so a P2P transaction arriving during the RPC scan is not evicted by
// an older snapshot.
func (p *MiniMemPool) syncMempoolFromRPC(stop <-chan struct{}, initial bool) bool {
	start := time.Now()
	common.Log.Infof("start to reconcile mempool (initial=%v)", initial)

	p.mutex.RLock()
	existingAtStart := make(map[string]struct{}, len(p.txMap))
	for txID := range p.txMap {
		existingAtStart[txID] = struct{}{}
	}
	p.mutex.RUnlock()

	txIDs, err := bitcoin_rpc.ShareBitconRpc.GetMemPool()
	if err != nil {
		common.Log.Infof("GetMemPool error: %v", err)
		return false
	}
	if p.shouldStop(stop) {
		return false
	}

	snapshot := make(map[string]struct{}, len(txIDs))
	added := make(map[string]*wire.MsgTx)
	for _, txID := range txIDs {
		snapshot[txID] = struct{}{}
		if _, existed := existingAtStart[txID]; existed {
			continue
		}
		if p.shouldStop(stop) {
			return false
		}
		txHex, err := bitcoin_rpc.ShareBitconRpc.GetRawTx(txID)
		if err != nil {
			common.Log.Errorf("GetRawTx %s failed, %v", txID, err)
			continue
		}
		tx, err := DecodeMsgTx(txHex)
		if err != nil {
			common.Log.Errorf("DecodeMsgTx %s failed, %v", txID, err)
			continue
		}
		added[txID] = tx
	}

	missing := make([]string, 0)
	for txID := range existingAtStart {
		if _, ok := snapshot[txID]; !ok {
			missing = append(missing, txID)
		}
	}
	sort.Strings(missing)

	p.processingMutex.Lock()
	p.mutex.Lock()
	for _, txID := range missing {
		if _, stillPresent := p.txMap[txID]; stillPresent {
			p.removeTransactionLocked(txID, true, false)
		}
	}
	p.mutex.Unlock()
	p.processingMutex.Unlock()

	for _, txID := range txIDs {
		if tx := added[txID]; tx != nil {
			p.txBroadcasted(tx)
		}
		if p.shouldStop(stop) {
			return false
		}
	}
	p.retryPendingTransactions(mempoolRetryMaxPasses)
	p.pruneConfirmedSpends()

	common.Log.Infof("mempool reconciliation completed: node=%d added=%d removed=%d elapsed=%v",
		len(txIDs), len(added), len(missing), time.Since(start))
	return true
}

func DecodeMsgTx(txHex string) (*wire.MsgTx, error) {
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, fmt.Errorf("error decoding hex string: %v", err)
	}
	msgTx := wire.NewMsgTx(wire.TxVersion)
	if err := msgTx.Deserialize(bytes.NewReader(txBytes)); err != nil {
		return nil, fmt.Errorf("error deserializing transaction: %v", err)
	}
	return msgTx, nil
}

func (p *MiniMemPool) txBroadcasted(tx *wire.MsgTx) {
	p.processingMutex.Lock()
	defer p.processingMutex.Unlock()

	p.enterIndexerRead()
	defer p.leaveIndexerRead()

	txID := tx.TxID()
	p.mutex.Lock()
	if _, exists := p.txMap[txID]; !exists {
		p.admitTransactionLocked(tx)
	} else if p.classifiedTxMap[txID] {
		p.mutex.Unlock()
		common.Log.Debugf("tx %s already classified in mempool", txID)
		return
	}
	p.mutex.Unlock()

	inputs, status := p.resolveMempoolInputs(tx)
	p.commitMempoolSpentInputs(txID, inputs)

	switch status {
	case mempoolResolvePending:
		common.Log.Debugf("mempool tx %s waits for known parent/input", txID)
		return
	case mempoolResolveBlocked:
		p.mutex.Lock()
		p.classifiedTxMap[txID] = true
		p.mutex.Unlock()
		common.Log.Debugf("mempool tx %s output classification stopped by non-plain/unknown mempool input", txID)
		return
	}

	outputs, occupied, ok := p.allocateKnownMempoolTx(tx, inputs)
	if !ok {
		p.mutex.Lock()
		p.classifiedTxMap[txID] = true
		p.mutex.Unlock()
		common.Log.Debugf("mempool tx %s output classification intentionally unresolved", txID)
		return
	}
	p.commitMempoolOutputs(tx, outputs, occupied)
}

func (p *MiniMemPool) admitTransactionLocked(tx *wire.MsgTx) {
	txID := tx.TxID()
	if _, exists := p.txMap[txID]; exists {
		return
	}

	inputs := make([]string, 0, len(tx.TxIn))
	for _, txIn := range tx.TxIn {
		if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
			continue
		}
		outpoint := txIn.PreviousOutPoint.String()
		inputs = append(inputs, outpoint)
		if owner := p.spentByOutpoint[outpoint]; owner != "" && owner != txID {
			// A newly relayed replacement invalidates the previous spender and
			// every descendant that depends on its unconfirmed outputs.
			p.removeTransactionLocked(owner, true, true)
		}
	}

	p.txMap[txID] = tx
	p.classifiedTxMap[txID] = false
	p.inputsByTx[txID] = inputs
	for i := range tx.TxOut {
		p.utxoStateMap[fmt.Sprintf("%s:%d", txID, i)] = mempoolUtxoUnknown
	}
	for _, txIn := range tx.TxIn {
		if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
			continue
		}
		outpoint := txIn.PreviousOutPoint.String()
		parentID := txIn.PreviousOutPoint.Hash.String()
		if p.childrenByTx[parentID] == nil {
			p.childrenByTx[parentID] = make(map[string]struct{})
		}
		p.childrenByTx[parentID][txID] = struct{}{}
		p.spentByOutpoint[outpoint] = txID
		p.removePlainAvailabilityLocked(outpoint)
	}
}

func (p *MiniMemPool) commitMempoolSpentInputs(txID string, inputs []*mempoolResolvedInput) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, resolved := range inputs {
		if resolved == nil || resolved.output == nil {
			continue
		}
		info := resolved.output
		outpoint := info.OutPointStr
		if outpoint == "" || p.spentByOutpoint[outpoint] != txID {
			continue
		}
		p.spentUtxoMap[outpoint] = info.Clone()
		p.addSpentToAddressLocked(outpoint, info)
		p.removePlainAvailabilityLocked(outpoint)
	}
}

func (p *MiniMemPool) commitMempoolOutputs(tx *wire.MsgTx, outputs []*common.TxOutput, occupied []bool) {
	txID := tx.TxID()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i, output := range outputs {
		if output == nil || i >= len(occupied) {
			continue
		}
		outpoint := fmt.Sprintf("%s:%d", txID, i)
		if mempoolOutputUnspendable(tx.TxOut[i]) || occupied[i] {
			p.utxoStateMap[outpoint] = mempoolUtxoNonPlain
			p.removeKnownPlainLocked(outpoint)
			continue
		}

		plain := output.Clone()
		plain.UtxoId = common.INVALID_ID
		plain.OutPointStr = outpoint
		plain.Assets = nil
		plain.Offsets = make(map[common.AssetName]common.AssetOffsets)
		plain.SatBindingMap = make(map[int64]*common.AssetInfo)
		plain.Invalids = make(map[common.AssetName]bool)
		p.utxoStateMap[outpoint] = mempoolUtxoPlain
		p.knownPlainUtxoMap[outpoint] = plain

		if p.spentByOutpoint[outpoint] == "" {
			if _, confirmed := p.confirmedSpent[outpoint]; !confirmed {
				p.addPlainAvailabilityLocked(outpoint, plain)
			}
		}
	}
	p.classifiedTxMap[txID] = true
}

func (p *MiniMemPool) getOrCreateUserLocked(address string) *UserUtxoInMempool {
	user := p.addrUtxoMap[address]
	if user == nil {
		user = &UserUtxoInMempool{
			SpentUtxo:               make(map[string]*common.TxOutput),
			UnconfirmedPlainUtxoMap: make(map[string]*common.TxOutput),
		}
		p.addrUtxoMap[address] = user
	}
	return user
}

func (p *MiniMemPool) addressForOutput(output *common.TxOutput) (string, bool) {
	if output == nil || instance == nil || instance.GetChainParam() == nil {
		return "", false
	}
	address, err := common.PkScriptToAddr(output.OutValue.PkScript, instance.GetChainParam())
	return address, err == nil
}

func (p *MiniMemPool) addSpentToAddressLocked(outpoint string, output *common.TxOutput) {
	address, ok := p.addressForOutput(output)
	if !ok {
		return
	}
	p.getOrCreateUserLocked(address).SpentUtxo[outpoint] = output.Clone()
}

func (p *MiniMemPool) removeSpentDetailLocked(outpoint string) {
	info := p.spentUtxoMap[outpoint]
	delete(p.spentUtxoMap, outpoint)
	address, ok := p.addressForOutput(info)
	if !ok {
		return
	}
	if user := p.addrUtxoMap[address]; user != nil {
		delete(user.SpentUtxo, outpoint)
	}
}

func (p *MiniMemPool) addPlainAvailabilityLocked(outpoint string, output *common.TxOutput) {
	address, ok := p.addressForOutput(output)
	if !ok {
		return
	}
	p.getOrCreateUserLocked(address).UnconfirmedPlainUtxoMap[outpoint] = output.Clone()
}

func (p *MiniMemPool) removePlainAvailabilityLocked(outpoint string) {
	output := p.knownPlainUtxoMap[outpoint]
	address, ok := p.addressForOutput(output)
	if !ok {
		return
	}
	if user := p.addrUtxoMap[address]; user != nil {
		delete(user.UnconfirmedPlainUtxoMap, outpoint)
	}
}

func (p *MiniMemPool) restorePlainAvailabilityLocked(outpoint string) {
	if p.spentByOutpoint[outpoint] != "" {
		return
	}
	if _, confirmed := p.confirmedSpent[outpoint]; confirmed {
		return
	}
	if output := p.knownPlainUtxoMap[outpoint]; output != nil {
		p.addPlainAvailabilityLocked(outpoint, output)
	}
}

func (p *MiniMemPool) removeKnownPlainLocked(outpoint string) {
	p.removePlainAvailabilityLocked(outpoint)
	delete(p.knownPlainUtxoMap, outpoint)
}

// removeTransactionLocked requires p.mutex. recursive is used for RBF/conflict
// eviction. restoreInputs is false only when the transaction has confirmed;
// its inputs remain hidden by confirmedSpent until the index catches up.
func (p *MiniMemPool) removeTransactionLocked(txID string, restoreInputs, recursive bool) {
	if recursive {
		children := make([]string, 0, len(p.childrenByTx[txID]))
		for child := range p.childrenByTx[txID] {
			children = append(children, child)
		}
		sort.Strings(children)
		for _, child := range children {
			p.removeTransactionLocked(child, true, true)
		}
	}

	tx := p.txMap[txID]
	for _, outpoint := range p.inputsByTx[txID] {
		parentID := ""
		if parsed, err := wire.NewOutPointFromString(outpoint); err == nil {
			parentID = parsed.Hash.String()
		}
		if parentID != "" {
			if children := p.childrenByTx[parentID]; children != nil {
				delete(children, txID)
				if len(children) == 0 {
					delete(p.childrenByTx, parentID)
				}
			}
		}
		if p.spentByOutpoint[outpoint] == txID {
			delete(p.spentByOutpoint, outpoint)
			if restoreInputs {
				p.removeSpentDetailLocked(outpoint)
				p.restorePlainAvailabilityLocked(outpoint)
			}
		}
	}

	if tx != nil {
		for i := range tx.TxOut {
			outpoint := fmt.Sprintf("%s:%d", txID, i)
			p.removeKnownPlainLocked(outpoint)
			delete(p.utxoStateMap, outpoint)
		}
	}
	delete(p.txMap, txID)
	delete(p.classifiedTxMap, txID)
	delete(p.inputsByTx, txID)
	delete(p.childrenByTx, txID)
}

func (p *MiniMemPool) retryPendingTransactions(maxPasses int) {
	if maxPasses <= 0 {
		return
	}
	for pass := 0; pass < maxPasses; pass++ {
		p.mutex.RLock()
		pendingIDs := make([]string, 0)
		for txID := range p.txMap {
			if !p.classifiedTxMap[txID] {
				pendingIDs = append(pendingIDs, txID)
			}
		}
		sort.Strings(pendingIDs)
		pending := make([]*wire.MsgTx, 0, len(pendingIDs))
		for _, txID := range pendingIDs {
			pending = append(pending, p.txMap[txID])
		}
		p.mutex.RUnlock()
		if len(pending) == 0 {
			return
		}

		before := len(pending)
		for _, tx := range pending {
			if tx != nil {
				p.txBroadcasted(tx)
			}
		}
		p.mutex.RLock()
		after := 0
		for txID := range p.txMap {
			if !p.classifiedTxMap[txID] {
				after++
			}
		}
		p.mutex.RUnlock()
		if after == 0 || after >= before {
			return
		}
	}
}

func (p *MiniMemPool) confirmTransactionLocked(tx *wire.MsgTx) {
	txID := tx.TxID()
	for _, txIn := range tx.TxIn {
		if txIn.PreviousOutPoint.Index >= wire.MaxPrevOutIndex {
			continue
		}
		outpoint := txIn.PreviousOutPoint.String()
		if owner := p.spentByOutpoint[outpoint]; owner != "" && owner != txID {
			p.removeTransactionLocked(owner, true, true)
		}
		delete(p.spentByOutpoint, outpoint)
		p.confirmedSpent[outpoint] = struct{}{}
		p.removePlainAvailabilityLocked(outpoint)
	}
	// Preserve valid descendants: once this transaction confirms, children may
	// remain in the node mempool and will resolve through the confirmed index.
	p.removeTransactionLocked(txID, false, false)
}

func (p *MiniMemPool) pruneConfirmedSpends() {
	if instance == nil {
		return
	}
	p.mutex.RLock()
	outpoints := make([]string, 0, len(p.confirmedSpent))
	for outpoint := range p.confirmedSpent {
		outpoints = append(outpoints, outpoint)
	}
	p.mutex.RUnlock()
	if len(outpoints) == 0 {
		return
	}
	sort.Strings(outpoints)

	prunable := make([]string, 0)
	p.enterIndexerRead()
	for _, outpoint := range outpoints {
		if instance.GetTxOutputWithUtxoV2(outpoint, true) == nil {
			prunable = append(prunable, outpoint)
		}
	}
	p.leaveIndexerRead()

	p.mutex.Lock()
	for _, outpoint := range prunable {
		if _, stillConfirmed := p.confirmedSpent[outpoint]; !stillConfirmed {
			continue
		}
		delete(p.confirmedSpent, outpoint)
		if p.spentByOutpoint[outpoint] == "" {
			p.removeSpentDetailLocked(outpoint)
		}
	}
	p.mutex.Unlock()
}

func (p *MiniMemPool) listenP2PTx(addr string, stop <-chan struct{}) {
	for {
		if p.shouldStop(stop) {
			return
		}
		cfg := &peer.Config{
			UserAgentName:    "MempoolSync",
			UserAgentVersion: "0.1",
			ChainParams:      instance.GetChainParam(),
			Listeners: peer.MessageListeners{
				OnTx: func(_ *peer.Peer, msg *wire.MsgTx) {
					if p.shouldStop(stop) {
						return
					}
					common.Log.Debugf("OnTx %s", msg.TxID())
					p.txBroadcasted(msg)
					p.retryPendingTransactions(mempoolRetryMaxPasses)
				},
				OnBlock: func(_ *peer.Peer, msg *wire.MsgBlock, _ []byte) {
					if p.shouldStop(stop) {
						return
					}
					common.Log.Infof("OnBlock %s", msg.BlockHash().String())
					p.ProcessBlock(msg)
				},
				OnInv: func(remote *peer.Peer, msg *wire.MsgInv) {
					if p.shouldStop(stop) {
						return
					}
					common.Log.Debugf("OnInv: %v", msg.InvList)
					var getDataMsg wire.MsgGetData
					for _, inv := range msg.InvList {
						if inv.Type == wire.InvTypeTx || inv.Type == wire.InvTypeBlock {
							_ = getDataMsg.AddInvVect(inv)
						}
					}
					if len(getDataMsg.InvList) > 0 {
						remote.QueueMessage(&getDataMsg, nil)
					}
				},
			},
		}
		outbound, err := peer.NewOutboundPeer(cfg, addr)
		if err != nil {
			common.Log.Errorf("NewOutboundPeer error: %v", err)
			select {
			case <-stop:
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			common.Log.Errorf("Dial P2P error: %v", err)
			select {
			case <-stop:
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}
		outbound.AssociateConnection(conn)

		p.lifecycleMutex.Lock()
		if !p.running || p.stopChan != stop {
			p.lifecycleMutex.Unlock()
			outbound.Disconnect()
			return
		}
		p.peer = outbound
		p.lifecycleMutex.Unlock()
		common.Log.Infof("Connected to P2P node: %s", addr)

		for outbound.Connected() {
			select {
			case <-stop:
				outbound.Disconnect()
				return
			case <-time.After(3 * time.Second):
			}
		}
		p.lifecycleMutex.Lock()
		if p.peer == outbound {
			p.peer = nil
		}
		p.lifecycleMutex.Unlock()
		common.Log.Warningf("Disconnected from P2P node: %s, will reconnect...", addr)
		select {
		case <-stop:
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (p *MiniMemPool) ProcessBlock(msg *wire.MsgBlock) {
	start := time.Now()
	p.processingMutex.Lock()
	p.mutex.Lock()
	for _, tx := range msg.Transactions {
		p.confirmTransactionLocked(tx)
	}
	remaining := len(p.txMap)
	p.mutex.Unlock()
	p.processingMutex.Unlock()
	common.Log.Infof("ProcessBlock completed, new size %d. %v", remaining, time.Since(start))

	// Reconcile every block. scheduleSync is single-flight, so a slow previous
	// reconciliation cannot create overlapping resets or RPC scans.
	p.scheduleSync(false)
}

func (p *MiniMemPool) ProcessReorg() {
	p.Stop()
	p.init()
	common.Log.Infof("ProcessReorg, reset mempool")
}

func (p *MiniMemPool) RemoveSpentUtxo(utxos []string) []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]string, 0)
	for _, utxo := range utxos {
		_, pending := p.spentByOutpoint[utxo]
		_, confirmed := p.confirmedSpent[utxo]
		if !pending && !confirmed {
			result = append(result, utxo)
		}
	}
	return result
}

func (p *MiniMemPool) IsSpent(utxo string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if p.spentByOutpoint[utxo] != "" {
		return true
	}
	_, confirmed := p.confirmedSpent[utxo]
	return confirmed
}

func (p *MiniMemPool) GetSpentUtxoByAddress(address string) []string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	addrUtxo := p.addrUtxoMap[address]
	if addrUtxo == nil {
		return nil
	}
	result := make([]string, 0, len(addrUtxo.SpentUtxo))
	for outpoint := range addrUtxo.SpentUtxo {
		if p.spentByOutpoint[outpoint] != "" {
			result = append(result, outpoint)
			continue
		}
		if _, confirmed := p.confirmedSpent[outpoint]; confirmed {
			result = append(result, outpoint)
		}
	}
	sort.Strings(result)
	return result
}

// GetUnconfirmedPlainUtxoByAddress returns only currently unspent mempool
// outputs that have been fully classified and contain no supported asset.
func (p *MiniMemPool) GetUnconfirmedPlainUtxoByAddress(address string) map[string]*common.TxOutput {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	addrUtxo := p.addrUtxoMap[address]
	if addrUtxo == nil {
		return nil
	}
	result := make(map[string]*common.TxOutput, len(addrUtxo.UnconfirmedPlainUtxoMap))
	for outpoint, output := range addrUtxo.UnconfirmedPlainUtxoMap {
		if p.spentByOutpoint[outpoint] != "" {
			continue
		}
		if _, confirmed := p.confirmedSpent[outpoint]; confirmed {
			continue
		}
		result[outpoint] = output.Clone()
	}
	return result
}

func (p *MiniMemPool) GetUnconfirmedSpentUtxoByAddress(address string) map[uint64]*common.TxOutput {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	addrUtxo := p.addrUtxoMap[address]
	if addrUtxo == nil {
		return nil
	}
	invalidID := uint64(common.INVALID_ID)
	result := make(map[uint64]*common.TxOutput, len(addrUtxo.SpentUtxo))
	for outpoint, output := range addrUtxo.SpentUtxo {
		if p.spentByOutpoint[outpoint] == "" {
			if _, confirmed := p.confirmedSpent[outpoint]; !confirmed {
				continue
			}
		}
		id := output.UtxoId
		if id == common.INVALID_ID {
			id = invalidID
			invalidID--
		}
		result[id] = output.Clone()
	}
	return result
}
