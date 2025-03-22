package dkvs

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/avast/retry-go"
	"github.com/gogo/protobuf/proto"
	u "github.com/ipfs/boxo/util"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	record "github.com/libp2p/go-libp2p-record"
	recpb "github.com/libp2p/go-libp2p-record/pb"
	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/multiformats/go-base32"
	"github.com/sat20-labs/indexer/dkvs/kaddht"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	lock sync.Mutex
)

type PeerInfo struct {
	PeerID        peer.ID
	ConnectStatus network.Connectedness
}
type PeerInfoList map[string]*PeerInfo

func (d *Dkvs) asyncPut(key string, value []byte) (err error) {
	ctx1, cancel := context.WithCancel(context.Background())
	defer cancel()
	//先put到本地
	err = d.putKeyToLocalNode(ctx1, key, value)
	if err != nil {
		return err
	}
	//put本地成功后，然后再异步put到网络上去
	go func() {
		//执行异步操作
		ctx2, cancel := context.WithCancel(context.Background())
		defer cancel()
		rec, err := d.getLocal(ctx2, key)
		if err != nil {
			Logger.Errorf("asyncPut--->Failed to read key from local node {key: %s} err: %s", key, err.Error())
			return
		}
		if rec == nil {
			Logger.Errorf("asyncPut--->Failed to read key from local node {key: %s}", key)
			return
		}
		err = d.putKeyToNetNode(ctx2, key, rec)
		if err != nil && err.Error() == ErrLookupFailure.Error() {
			d.saveUnsyncedKey(ctx2, key) //Save the key to the local unsynchronized database
			Logger.Errorf("asyncPut--->put key to network failed, currently only put to the local node: {key: %s} at local", key)
		} else {
			Logger.Infof("asyncPut success! --->put key to network success--->{key: %s}", key)
		}
	}()
	return nil
}

func (d *Dkvs) putKeyToLocalNode(ctx context.Context, key string, value []byte, opts ...routing.Option) (err error) {
	dht := d.idht
	ctx, span := d.startSpan(ctx, "IpfsDHT.PutValue", trace.WithAttributes(attribute.String("Key", key)))
	defer span.End()

	Logger.Infof("putting value", "key", kaddht.LoggableRecordKeyString(key))
	Logger.Debugf("putting value {dskey: %v}", d.mkDsKey(key))

	// don't even allow local users to put bad values.
	if err := dht.Validator.Validate(key, value); err != nil {
		return err
	}

	old, err := d.getLocal(ctx, key)
	if err != nil {
		// Means something is wrong with the datastore.
		return err
	}

	// Check if we have an old value that's not the same as the new one.
	if old != nil && !bytes.Equal(old.GetValue(), value) {
		// Check to see if the new one is better.
		i, err := dht.Validator.Select(key, [][]byte{value, old.GetValue()})
		if err != nil {
			return err
		}
		if i != 0 {
			return fmt.Errorf("can't replace a newer value with an older value")
		}
	}

	rec := record.MakePutRecord(key, value)
	rec.TimeReceived = u.FormatRFC3339(time.Now())
	err = d.putLocal(ctx, key, rec)
	if err != nil {
		return err
	}
	return nil
}

func (d *Dkvs) getLocal(ctx context.Context, key string) (*recpb.Record, error) {
	Logger.Infof("finding value in datastore", "key", kaddht.LoggableRecordKeyString(key))

	rec, err := d.getRecordFromDatastore(ctx, key)
	if err != nil {
		Logger.Warnln("get local failed", "key", kaddht.LoggableRecordKeyString(key), "error", err)
		return nil, err
	}

	// Double check the key. Can't hurt.
	if rec != nil && string(rec.GetKey()) != key {
		Logger.Errorln("BUG: found a DHT record that didn't match it's key", "expected", kaddht.LoggableRecordKeyString(key), "got", rec.GetKey())
		return nil, nil

	}
	return rec, nil
}

func (d *Dkvs) getRecordFromDatastore(ctx context.Context, key string) (*recpb.Record, error) {
	dskey := d.mkDsKey(key)
	dht := d.idht
	buf, err := d.dhtDatastore.Get(ctx, dskey)
	if err == ds.ErrNotFound {
		Logger.Debugf("finding value in datastore {key: %s, error: %s}", dskey, err.Error())
		return nil, nil
	}
	if err != nil {
		Logger.Errorln("error retrieving record from datastore", "key", dskey, "error", err)
		return nil, err
	}
	rec := new(recpb.Record)
	err = proto.Unmarshal(buf, rec)
	if err != nil {
		// Bad data in datastore, log it but don't return an error, we'll just overwrite it
		Logger.Errorln("failed to unmarshal record from datastore", "key", dskey, "error", err)
		return nil, nil
	}

	err = dht.Validator.Validate(string(rec.GetKey()), rec.GetValue())
	if err != nil {
		// Invalid record in datastore, probably expired but don't return an error,
		// we'll just overwrite it
		Logger.Infof("local record verify failed", "key", rec.GetKey(), "error", err)
		return nil, nil
	}

	return rec, nil
}

// putLocal stores the key value pair in the datastore
func (d *Dkvs) putLocal(ctx context.Context, key string, rec *recpb.Record) error {
	data, err := proto.Marshal(rec)
	if err != nil {
		Logger.Warnln("failed to put marshal record for local put", "error", err, "key", kaddht.LoggableRecordKeyString(key))
		return err
	}

	return d.dhtDatastore.Put(ctx, d.mkDsKey(key), data)
}

func (d *Dkvs) startSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("go-libp2p-kad-dht").Start(ctx, fmt.Sprintf("KademliaDHT.%s", name), opts...)
}

func (d *Dkvs) putAllUnsyncKeyToNetwork(peerID peer.ID) error {
	lock.Lock() //加锁防止多线程同时操作putAllKeyToPeers
	defer lock.Unlock()
	if d.getUnsyncedDbSize() == 0 {
		Logger.Debugf("putAllUnsyncKeyToNetwork---> unsyncDb content is empty, no keys need to be synced to the network")
		return nil
	}
	log.Printf("peerID: %s", peerID)
	Logger.Info("putAllUnsyncKeyToNetwork---> start")
	Logger.Info("sync start unsyncDb content: ---")
	d.printUnsyncedDb()
	d.putAllKeysToPeers() //将db中未同步的key再一次put到其他节点
	Logger.Info("putAllUnsyncKeyToNetwork---> end")
	if d.getUnsyncedDbSize() > 0 {
		Logger.Info("sync end unsyncDb content: ---")
		d.printUnsyncedDb()
	}
	return nil
}

func (d *Dkvs) putKeyToNetNode(ctx context.Context, key string, rec *recpb.Record) error {
	peers, err := d.getAvailablePeerList(ctx, key)
	if err != nil {
		Logger.Error("d.baseService.GetAvailableServicePeerList(key) not return any connected node")
		serviceNodeCount := d.tryToConnectNetPeers() //当GetAvailableServicePeerList返回为空时将主动连接网络节点以提高连接节的稳定性
		if serviceNodeCount == 0 {                   //没有连接到任何服务节点
			return err
		}
	}
	isInDhtNet := false
	wg := sync.WaitGroup{}
	for _, p := range peers {
		wg.Add(1)
		go func(p peer.ID) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			defer wg.Done()
			routing.PublishQueryEvent(ctx, &routing.QueryEvent{
				Type: routing.Value,
				ID:   p,
			})

			err := d.protoMessenger.PutValue(ctx, p, rec)
			if err != nil {
				Logger.Warnf("putKeyToNetNode--> failed putting value to this peer failed--->{key: %v, pee: %v, err: %v} ", key, p, err)
			} else {
				isInDhtNet = true
				Logger.Debugf("putKeyToNetNode--> success putting value to this peer--->{key: %v, pee: %v} ", key, p)
				err := d.updateProvider(ctx, p, key)
				if err != nil {
					Logger.Debugf("putKeyToNetNode--> d.updateProvider(ctx, p, key) failed--->{key: %v, pee: %v, err: %v} ", key, p, err)
				}
			}
		}(p)
	}
	wg.Wait()
	if isInDhtNet {
		d.enableProvider(ctx, key) //makes this node announce that it can provide a value for the given key
	} else {
		Logger.Errorf("putKeyToNetNode--> key not put any other node--->{key: %v}", key)
	}
	return nil
}

// 传入业务的key转换成数据库的key格式
func (d *Dkvs) mkDsKey(s string) ds.Key {
	return ds.NewKey(base32.RawStdEncoding.EncodeToString([]byte(s)))
}

// 传入数据库的key的字符串格式（dsk字符串格式）可以decode到原来的业务的key
func (d *Dkvs) dsKeyDcode(s string) ([]byte, error) {
	return base32.RawStdEncoding.DecodeString(s)
}

func (d *Dkvs) getUnsyncedKey(cxt context.Context, key string) ([]byte, error) {
	value, err := d.dkvsdb.Get(cxt, d.mkDsKey(key))
	if err != nil {
		Logger.Error("LDB Get ", err)
		return nil, err
	}

	return value, nil
}

func (d *Dkvs) saveUnsyncedKey(cxt context.Context, key string) error {
	err := d.dkvsdb.Put(cxt, d.mkDsKey(key), []byte(key))
	if err != nil {
		Logger.Error("Unsync LDB Put ", err)
	}
	d.printUnsyncedDb() //for debug
	return err
}

func (d *Dkvs) deleteUnsyncedKey(cxt context.Context, key string) error {
	err := d.dkvsdb.Delete(cxt, d.mkDsKey(key))
	if err != nil {
		Logger.Error("Unsync LDB Delete ", err)
	}

	return err
}

func (d *Dkvs) closeUnsyncedDB(cxt context.Context) error {
	err := d.dkvsdb.Close()
	if err != nil {
		Logger.Error("LDB Close ", err)
	}

	return err
}

func (d *Dkvs) printUnsyncedDb() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q := query.Query{}
	results, err := d.dkvsdb.Query(ctx, q)
	if err != nil {
		Logger.Errorf("printUnsyncDb---> Error querying DataStore: %v", err)
	}
	defer results.Close()

	for result := range results.Next() {
		Logger.Debugf("printUnsyncDb {key: %s}", result.Value)
	}
}

func (d *Dkvs) getUnsyncedDbSize() uint64 {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q := query.Query{}
	var size uint64 = 0
	results, err := d.dkvsdb.Query(ctx, q)
	if err != nil {
		return size
	}
	defer results.Close()

	for range results.Next() {
		size++
	}
	return size
}

func (d *Dkvs) putAllKeysToPeers() error {
	// Query the DataStore for all keys and values
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q := query.Query{}
	results, err := d.dkvsdb.Query(ctx, q)
	if err != nil {
		Logger.Error("putAllKeysToPeers---> Error querying DataStore: ", err)
		return err
	}
	defer results.Close()

	delMap := make(map[string]string)
	for result := range results.Next() {
		Logger.Debugln("Key: ", result.Key)
		rec, err := d.getLocal(ctx, string(result.Value))
		if err != nil || rec == nil {
			Logger.Warnf("putAllKeysToPeers---> There is no such key in dht db {key: %s}", result.Value)
			d.deleteUnsyncedKey(ctx, string(result.Value))
			continue
		}
		err = d.putKeyToNetNode(ctx, string(result.Value), rec)
		if err != nil {
			Logger.Debugf("putAllKeysToPeers---> dhtPut %s return %s\n", result.Value, err.Error())
		}
		if err == nil || err.Error() == ErrBadRecord.Error() {
			delMap[result.Key] = string(result.Value)
		}
	}

	for _, v := range delMap {
		err := d.deleteUnsyncedKey(ctx, v)
		if err != nil {
			Logger.Errorf("putAllKeysToPeers---> DeleteUnsyncKey %v", err.Error())
		}
	}
	Logger.Debug("putAllKeysToPeers---> printUnsyncedDb")
	return nil
}

func (d *Dkvs) FindPeersByKey(ctx context.Context, key string, timeout time.Duration) []peer.AddrInfo {
	ctxT, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	dkvsKey := RecordKey(key)
	keyCid := ConvertKeyToCid(dkvsKey)
	var providers []peer.AddrInfo
	providers, err := d.idht.FindProviders(ctxT, keyCid)
	if err != nil {
		Logger.Warnf("findPeersByKey--> d.idht.FindProviders failed{key: %s, err: %s}", key, err.Error())
	}
	if providers == nil {
		Logger.Errorf("findPeersByKey--> {key: %s} is not saved to any node", key)
	} else {
		Logger.Debugf("findPeersByKey--> {key: %s} in {perrs: %v}", key, providers)
	}
	return providers
}

// makes this node announce that it can provide a value for the given key
func (d *Dkvs) enableProvider(ctx context.Context, key string) {
	keyCid := ConvertKeyToCid(key)
	err := d.idht.Provide(ctx, keyCid, true)
	if err != nil {
		Logger.Warnf("failed to set key to node  provider list {key: %s, cid: %s}", key, keyCid, err)
		return
	}
	Logger.Debugf("success to set key to node  provider list: {key: %s, cid: %s}", key, keyCid)
}

func ConvertKeyToCid(key string) cid.Cid {
	mhv := u.Hash([]byte(key))
	keyCid := cid.NewCidV1(cid.Raw, mhv)
	return keyCid
}

func (d *Dkvs) getDhtClosestPeers(ctx context.Context, key string) ([]peer.ID, error) {
	var peers []peer.ID
	retryStrategy := []retry.Option{
		retry.Delay(500 * time.Millisecond), // delay 500 ms
		retry.Attempts(3),                   // max retry times 3
		retry.LastErrorOnly(true),           // Return last error only
		retry.RetryIf(func(err error) bool { // Retries based on error type
			switch err.Error() {
			case ErrLookupFailure.Error():
				return true
			default:
				return false
			}
		}),
	}
	err := retry.Do(
		func() error {
			var err error
			peers, err = d.idht.GetClosestPeers(ctx, key) // 自定义的函数
			if err != nil {
				Logger.Errorf("d.idht.GetClosestPeers failed andr retry: %v", err)
			}
			return err // 返回错误
		},
		retryStrategy...,
	)
	if err != nil {
		Logger.Errorf("d.idht.GetClosestPeers retry completed: %v", err)
		return nil, err
	}
	return peers, nil
}

// updateProvider asks a peer to store that we are a provider for the given key.
func (d *Dkvs) updateProvider(ctx context.Context, p peer.ID, key string) error {
	keyCid := ConvertKeyToCid(key)
	keyMH := keyCid.Hash()
	Logger.Debugf("updateProvider---> {peer: %v}", p)
	err := d.idht.ProviderStore().AddProvider(ctx, keyMH, peer.AddrInfo{ID: p})
	if err != nil {
		Logger.Debugf("updateProvider--->d.idht.ProviderStore().AddProvider failed {key: %s, peer: %s, err: %s}", key, p, err.Error())
		return err
	}
	return nil
}

// Periodically process unsynchronized keys
func (d *Dkvs) periodicallyProcessUnsyncKey() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			host := d.idht.Host()
			peers := host.Network().Peers()
			if len(peers) == 0 { // if len(ppers) equal 0 indicates that it is not connected to any node
				Logger.Debugf("periodicallyProcessUnsyncKey()--->the current node is not connected to the network")
				continue
			}
			Logger.Debug("periodicallyProcessUnsyncKey()--->the current node is connected to the network and call putAllUnsyncKeyToNetwork()")
			err := d.putAllUnsyncKeyToNetwork(d.idht.PeerID())
			if err != nil {
				Logger.Debugf("periodicallyProcessUnsyncKey()--->call putAllUnsyncKeyToNetwork() failed {err: s%}", err.Error())
			}
		}
	}()
}

// 主动连接之前连接过网络服务节点以提高网络的稳定性
func (d *Dkvs) tryToConnectNetPeers() int { //主动连接网络节点
	return 0
}


func (m *Dkvs) getAvailablePeerList(ctx context.Context, key string) ([]libp2pPeer.ID, error) {
	var findedPeerList []libp2pPeer.ID
	closestPeerList, err := m.idht.GetClosestPeers(ctx, key)
	if err != nil {
		Logger.Errorf("getAvailablePeerList: no find peers, err :%v", err)
		return findedPeerList, err
	}

	var peerList PeerInfoList
	// switch nodeMode {
	// case config.ServiceMode:
	// 	peerList = m.servicePeerList
	// case config.LightMode:
	// 	peerList = m.lightPeerList
	// default:
	// 	return nil, fmt.Errorf("getAvailablePeerList: invalid node mode")
	// }

	for _, closestPeer := range closestPeerList {
		closestPeerId := closestPeer.String()
		peerInfo := peerList[closestPeerId]
		if peerInfo == nil {
			Logger.Debugf("getAvailablePeerList: peer %v is not exist in peerList", closestPeer)
			continue
		}
		// if peerInfo.PeerID == m.host.ID() {
		// 	continue
		// }
		if peerInfo.ConnectStatus != network.Connected {
			Logger.Debugf("getAvailablePeerList: peer %v is not connected", closestPeer)
		}
		findedPeerList = append(findedPeerList, closestPeer)
	}
	if len(findedPeerList) == 0 {
		Logger.Error("getAvailablePeerList: no available peer found")
		return findedPeerList, fmt.Errorf("getAvailablePeerList: no available peer found")
	}
	return findedPeerList, nil
}
