package indexer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/atom"
	"github.com/sat20-labs/indexer/indexer/brc20"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/indexer/ft"
	"github.com/sat20-labs/indexer/indexer/nft"
	"github.com/sat20-labs/indexer/indexer/ns"
	"github.com/sat20-labs/indexer/indexer/runes"
)

func openAtomicalsInterfaceDB(t *testing.T) common.KVDB {
	t.Helper()
	kv := indexdb.NewKVDBWithCache(t.TempDir(), 1)
	if kv == nil {
		t.Fatal("open protocol test database")
	}
	t.Cleanup(func() {
		if err := kv.Close(); err != nil {
			t.Errorf("close protocol database: %v", err)
		}
	})
	return kv
}

func TestAtomicalsExistingQueriesKeepNativeQuantities(t *testing.T) {
	previousChain := common.CHAIN
	common.CHAIN = common.ChainRegtest
	t.Cleanup(func() { common.CHAIN = previousChain })
	mgr := newExactOutputTestManager(t)
	mgr.chaincfgParam = &chaincfg.RegressionNetParams
	mgr.atomIndexer = atom.NewIndexer(openAtomicalsInterfaceDB(t), mgr.chaincfgParam)
	mgr.ftIndexer = ft.NewOrdxIndexer(openAtomicalsInterfaceDB(t))
	mgr.nft = nft.NewNftIndexer(openAtomicalsInterfaceDB(t))
	mgr.nft.Init(mgr.base, nil)
	mgr.ns = ns.NewNameService(openAtomicalsInterfaceDB(t))
	mgr.ns.Init(mgr.nft)
	mgr.exotic = exotic.NewExoticIndexer(openAtomicalsInterfaceDB(t))
	mgr.exotic.Init(mgr.base)
	mgr.brc20Indexer = brc20.NewIndexer(openAtomicalsInterfaceDB(t), false)
	mgr.brc20Indexer.Init(mgr.nft)
	mgr.RunesIndexer = runes.NewIndexer(openAtomicalsInterfaceDB(t), mgr.chaincfgParam, false)
	mgr.RunesIndexer.Init(mgr.base)

	address, err := btcutil.NewAddressWitnessPubKeyHash(bytes.Repeat([]byte{1}, 20), mgr.chaincfgParam)
	if err != nil {
		t.Fatal(err)
	}
	pkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		t.Fatal(err)
	}
	const utxoID, addressID = uint64(99), uint64(7)
	const value = int64(600)
	outpoint := strings.Repeat("1", 64) + ":0"
	nativeID := strings.Repeat("2", 64) + "i0"
	name := common.AssetName{Protocol: common.PROTOCOL_NAME_ATOM, Type: common.ASSET_TYPE_FT, Ticker: "quark"}

	baseBatch := mgr.base.GetBaseDB().NewWriteBatch()
	defer baseBatch.Close()
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	check(indexdb.SetDBWithProto3(indexdb.GetUTXODBKey(outpoint), &common.UtxoValueInDB{UtxoId: utxoID, Value: value, AddressId: addressID}, baseBatch))
	check(indexdb.BindUtxoDBKeyToId(indexdb.GetUTXODBKey(outpoint), utxoID, baseBatch))
	check(indexdb.BindAddressDBKeyToId(address.EncodeAddress(), addressID, baseBatch))
	check(indexdb.SetDBWithProto3(indexdb.GetAddressDBKeyV2(address.EncodeAddress()), &common.AddressValueInDBV2{AddressId: addressID, AddressType: int32(txscript.WitnessV0PubKeyHashTy)}, baseBatch))
	check(baseBatch.Put(indexdb.GetAddressValueDBKeyCompact(addressID, utxoID), common.Uint64ToBytes(uint64(value))))
	check(baseBatch.Put(indexdb.GetAddressUtxoCountKey(addressID), common.Uint64ToBytes(1)))
	check(baseBatch.Flush())

	// Persist authoritative ARC20 state, then read it through the ordinary
	// public query methods. Bitcoin value exceeds the native token quantity.
	atomDB := openAtomicalsInterfaceDB(t)
	atomBatch := atomDB.NewWriteBatch()
	defer atomBatch.Close()
	check(indexdb.SetDB([]byte(atom.GetTickerKey("quark")), &atom.Ticker{
		Name: "quark", DisplayName: "QUARK", AtomicalId: nativeID, MintedAmount: 100, MintedTimes: 1, HolderCount: 1,
	}, atomBatch))
	check(indexdb.SetDB([]byte(atom.GetUtxoBalanceKey(utxoID, nativeID)), &atom.UtxoBalance{
		UtxoId: utxoID, AddressId: addressID, Outpoint: outpoint, AtomicalId: nativeID, Ticker: "quark", Amount: 100,
	}, atomBatch))
	check(atomBatch.Flush())
	mgr.atomIndexer = atom.NewIndexer(atomDB, mgr.chaincfgParam)
	mgr.atomIndexer.Init(mgr.base)

	ticker := mgr.GetTickerInfo(&name)
	if ticker == nil || ticker.AssetName != name || ticker.InscriptionId != nativeID || ticker.N != 1 || ticker.Divisibility != 0 {
		t.Fatalf("native metadata lost in existing ticker fields: %+v", ticker)
	}
	if mgr.GetBindingSat(&name) != 1 {
		t.Fatal("ARC20 capacity lost")
	}
	unsupported := name
	unsupported.Type = common.ASSET_TYPE_NFT
	if mgr.GetTickerInfo(&unsupported) != nil || mgr.GetBindingSat(&unsupported) != 0 {
		t.Fatal("non-FT Atomicals accepted by fungible asset queries")
	}

	utxos, err := mgr.GetAssetUTXOsInAddressWithTick(address.EncodeAddress(), &name)
	if err != nil || len(utxos) != 1 || utxos[utxoID] != 100 {
		t.Fatalf("existing address ticker query returned %+v, %v", utxos, err)
	}
	view := mgr.GetTxOutputWithUtxoV3(outpoint, true)
	if view == nil || view.Value != value || !bytes.Equal(view.PkScript, pkScript) || len(view.Assets) != 1 {
		t.Fatalf("existing UTXO query lost output facts: %+v", view)
	}
	asset := view.Assets[0]
	if asset.AssetName != name || asset.Amount != "100" || asset.Precision != 0 || asset.BindingSat != 1 || len(asset.Offsets) != 0 || len(asset.OffsetToAmts) != 0 {
		t.Fatalf("native quantity misrepresented as ordinal ranges: %+v", asset)
	}
	v3, err := mgr.GetAssetUTXOsInAddressWithTickV3(address.EncodeAddress(), &name, false)
	if err != nil || len(v3) != 1 || v3[0].OutPoint != outpoint || v3[0].Assets[0].Amount != "100" || len(v3[0].Assets[0].Offsets) != 0 {
		t.Fatalf("existing V3 address query returned %+v, %v", v3, err)
	}
}
