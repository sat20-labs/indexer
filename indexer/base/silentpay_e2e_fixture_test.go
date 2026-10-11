package base_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	coretypes "github.com/OLProtocol/go-bitcoind"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	baseindex "github.com/sat20-labs/indexer/indexer/base"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// The only substituted component is the Bitcoin RPC data source. Blocks are
// serialized wire blocks with real transaction IDs, mined regtest headers,
// witness commitments and script-verified spends. Base, DB and HTTP are real.
// This does not pretend to be a separately running Bitcoin Core process.
type spTestChain struct {
	bitcoin_rpc.BitcoinRPC // unsupported operations are never used by the harness
	mu sync.RWMutex
	chain []*wire.MsgBlock
	blocks map[string]*wire.MsgBlock
	txs map[string]*wire.MsgTx
	failRaw bool
	rawTxCalls int
}

func e2eKey(n byte) *btcec.PrivateKey {
	b := make([]byte, 32)
	b[31] = n
	key, _ := btcec.PrivKeyFromBytes(b)
	return key
}

func e2eP2W(key *btcec.PrivateKey) []byte {
	return append([]byte{0, 20}, btcutil.Hash160(key.PubKey().SerializeCompressed())...)
}

func e2eMerkle(hashes []chainhash.Hash) chainhash.Hash {
	for len(hashes) > 1 {
		next := make([]chainhash.Hash, 0, (len(hashes)+1)/2)
		for i := 0; i < len(hashes); i += 2 {
			j := i+1
			if j == len(hashes) { j = i }
			var pair [64]byte
			copy(pair[:32], hashes[i][:])
			copy(pair[32:], hashes[j][:])
			next = append(next, chainhash.DoubleHashH(pair[:]))
		}
		hashes = next
	}
	return hashes[0]
}

func (c *spTestChain) appendBlock(t *testing.T, transactions ...*wire.MsgTx) *wire.MsgBlock {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	height := len(c.chain)
	coinbase := wire.NewMsgTx(2)
	script, err := txscript.NewScriptBuilder().AddInt64(int64(height)).AddData([]byte{0x53, 0x50}).Script()
	if err != nil { t.Fatal(err) }
	coinbase.AddTxIn(wire.NewTxIn(&wire.OutPoint{Index: 0xffffffff}, script, nil))
	coinbase.AddTxOut(wire.NewTxOut(100000, e2eP2W(e2eKey(7))))
	block := &wire.MsgBlock{Header: wire.BlockHeader{
		Version: 0x20000000, PrevBlock: c.chain[height-1].BlockHash(),
		Timestamp: time.Unix(1700000000+int64(height)*600, 0), Bits: 0x207fffff,
	}, Transactions: append([]*wire.MsgTx{coinbase}, transactions...)}
	witness := false
	for _, tx := range transactions {
		if tx.HasWitness() { witness = true }
	}
	if witness {
		coinbase.TxIn[0].Witness = wire.TxWitness{make([]byte, 32)}
		leaves := make([]chainhash.Hash, len(block.Transactions))
		for i := 1; i < len(leaves); i++ { leaves[i] = block.Transactions[i].WitnessHash() }
		root := e2eMerkle(leaves)
		var preimage [64]byte
		copy(preimage[:32], root[:])
		commitment := chainhash.DoubleHashH(preimage[:])
		pkScript := append([]byte{0x6a, 0x24, 0xaa, 0x21, 0xa9, 0xed}, commitment[:]...)
		coinbase.AddTxOut(wire.NewTxOut(0, pkScript))
	}
	leaves := make([]chainhash.Hash, len(block.Transactions))
	for i, tx := range block.Transactions { leaves[i] = tx.TxHash() }
	block.Header.MerkleRoot = e2eMerkle(leaves)
	target := blockchain.CompactToBig(block.Header.Bits)
	for {
		hash := block.BlockHash()
		if blockchain.HashToBig(&hash).Cmp(target) <= 0 { break }
		block.Header.Nonce++
	}
	if err := blockchain.ValidateWitnessCommitment(btcutil.NewBlock(block)); err != nil { t.Fatal(err) }
	c.chain = append(c.chain, block)
	c.blocks[block.BlockHash().String()] = block
	for _, tx := range block.Transactions { c.txs[tx.TxHash().String()] = tx }
	return block
}

func newSPTestChain(t *testing.T) *spTestChain {
	t.Helper()
	genesis := chaincfg.RegressionNetParams.GenesisBlock
	c := &spTestChain{chain: []*wire.MsgBlock{genesis}, blocks: map[string]*wire.MsgBlock{genesis.BlockHash().String(): genesis}, txs: make(map[string]*wire.MsgTx)}
	for _, tx := range genesis.Transactions { c.txs[tx.TxHash().String()] = tx }
	for len(c.chain) <= 101 { c.appendBlock(t) }
	return c
}

func (c *spTestChain) GetBlockCount() (uint64, error) {
	c.mu.RLock(); defer c.mu.RUnlock()
	return uint64(len(c.chain)-1), nil
}
func (c *spTestChain) GetBestBlockHash() (string, error) {
	c.mu.RLock(); defer c.mu.RUnlock()
	return c.chain[len(c.chain)-1].BlockHash().String(), nil
}
func (c *spTestChain) GetBlockHash(height uint64) (string, error) {
	c.mu.RLock(); defer c.mu.RUnlock()
	if height >= uint64(len(c.chain)) { return "", fmt.Errorf("fixture height unavailable") }
	return c.chain[height].BlockHash().String(), nil
}
func (c *spTestChain) GetRawBlock(hash string) (string, error) {
	c.mu.RLock(); defer c.mu.RUnlock()
	block := c.blocks[hash]
	if block == nil || c.failRaw { return "", fmt.Errorf("fixture block unavailable") }
	var buf bytes.Buffer
	if err := block.Serialize(&buf); err != nil { return "", err }
	return hex.EncodeToString(buf.Bytes()), nil
}
func (c *spTestChain) GetRawTx(id string) (string, error) {
	c.mu.Lock(); defer c.mu.Unlock()
	c.rawTxCalls++
	tx := c.txs[id]
	if tx == nil || c.failRaw { return "", fmt.Errorf("fixture transaction unavailable") }
	var buf bytes.Buffer
	if err := tx.Serialize(&buf); err != nil { return "", err }
	return hex.EncodeToString(buf.Bytes()), nil
}
func (c *spTestChain) GetTx(id string) (*coretypes.RawTransaction, error) {
	return nil, fmt.Errorf("verbose transaction not part of SP scan fixture")
}
func (c *spTestChain) GetTxOut(string, uint32, bool) (*coretypes.UTransactionOut, error) {
	return nil, fmt.Errorf("current gettxout must not be used for SP snapshots")
}
func (c *spTestChain) SendTx(string) (string, error) {
	return "", fmt.Errorf("fixture never broadcasts")
}
func (c *spTestChain) truncate(height int) {
	c.mu.Lock(); defer c.mu.Unlock()
	c.chain = c.chain[:height+1]
}

func e2eTagged(tag string, parts ...[]byte) [32]byte {
	tagHash := sha256.Sum256([]byte(tag))
	h := sha256.New()
	h.Write(tagHash[:]); h.Write(tagHash[:])
	for _, p := range parts { h.Write(p) }
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

func e2ePointMul(t *testing.T, scalar *btcec.ModNScalar, pub *btcec.PublicKey) *btcec.PublicKey {
	t.Helper()
	var p, result btcec.JacobianPoint
	pub.AsJacobian(&p)
	btcec.ScalarMultNonConst(scalar, &p, &result)
	if result.Z.IsZero() { t.Fatal("fixture point at infinity") }
	result.ToAffine()
	return btcec.NewPublicKey(&result.X, &result.Y)
}

func e2eScalar(t *testing.T, bytes [32]byte) btcec.ModNScalar {
	t.Helper()
	var scalar btcec.ModNScalar
	if scalar.SetByteSlice(bytes[:]) || scalar.IsZero() { t.Fatal("invalid fixture scalar") }
	return scalar
}

func e2ePayment(t *testing.T, parent *wire.MsgTx, scan, spend *btcec.PrivateKey, count int) (*wire.MsgTx, []*btcec.PrivateKey) {
	t.Helper()
	payer := e2eKey(7)
	op := wire.OutPoint{Hash: parent.TxHash(), Index: 0}
	var serialized [36]byte
	copy(serialized[:32], op.Hash[:])
	binary.LittleEndian.PutUint32(serialized[32:], op.Index)
	// Independent sender-side equation, not SPComputeTweak (the index under test).
	inputHash := e2eScalar(t, e2eTagged("BIP0352/Inputs", serialized[:], payer.PubKey().SerializeCompressed()))
	inputHash.Mul(&payer.Key)
	secret := e2ePointMul(t, &inputHash, scan.PubKey()).SerializeCompressed()
	tx := wire.NewMsgTx(2)
	tx.AddTxIn(wire.NewTxIn(&op, nil, nil))
	keys := make([]*btcec.PrivateKey, 0, count)
	for k := 0; k < count; k++ {
		var counter [4]byte
		binary.BigEndian.PutUint32(counter[:], uint32(k))
		tweak := e2eScalar(t, e2eTagged("BIP0352/SharedSecret", secret, counter[:]))
		tweak.Add(&spend.Key)
		key := btcec.PrivKeyFromScalar(&tweak)
		keys = append(keys, key)
		tx.AddTxOut(wire.NewTxOut(20000, append([]byte{0x51, 0x20}, schnorr.SerializePubKey(key.PubKey())...)))
	}
	// Ordinary change deliberately separates the true vout and SP-output index.
	if count > 1 {
		change := wire.NewTxOut(10000, e2eP2W(e2eKey(8)))
		tx.TxOut = append(tx.TxOut[:1], append([]*wire.TxOut{change}, tx.TxOut[1:]...)...)
	}
	previous := parent.TxOut[0]
	fetcher := txscript.NewCannedPrevOutputFetcher(previous.PkScript, previous.Value)
	hashes := txscript.NewTxSigHashes(tx, fetcher)
	scriptCode := append(append([]byte{0x76, 0xa9, 20}, btcutil.Hash160(payer.PubKey().SerializeCompressed())...), 0x88, 0xac)
	witness, err := txscript.WitnessSignature(tx, hashes, 0, previous.Value, scriptCode, txscript.SigHashAll, payer, true)
	if err != nil { t.Fatal(err) }
	tx.TxIn[0].Witness = witness
	e2eVerifySpend(t, tx, previous)
	return tx, keys
}

func e2eVerifySpend(t *testing.T, tx *wire.MsgTx, previous *wire.TxOut) {
	t.Helper()
	fetcher := txscript.NewCannedPrevOutputFetcher(previous.PkScript, previous.Value)
	hashes := txscript.NewTxSigHashes(tx, fetcher)
	engine, err := txscript.NewEngine(previous.PkScript, tx, 0, txscript.StandardVerifyFlags, nil, hashes, previous.Value, fetcher)
	if err != nil { t.Fatal(err) }
	if err := engine.Execute(); err != nil { t.Fatalf("fixture spend script verification failed: %v", err) }
}

func e2eSpendSP(t *testing.T, parent *wire.MsgTx, vout uint32, key *btcec.PrivateKey) *wire.MsgTx {
	t.Helper()
	tx := wire.NewMsgTx(2)
	op := wire.OutPoint{Hash: parent.TxHash(), Index: vout}
	tx.AddTxIn(wire.NewTxIn(&op, nil, nil))
	previous := parent.TxOut[vout]
	tx.AddTxOut(wire.NewTxOut(previous.Value-1000, e2eP2W(e2eKey(9))))
	fetcher := txscript.NewCannedPrevOutputFetcher(previous.PkScript, previous.Value)
	hashes := txscript.NewTxSigHashes(tx, fetcher)
	digest, err := txscript.CalcTaprootSignatureHash(hashes, txscript.SigHashDefault, tx, 0, fetcher)
	if err != nil { t.Fatal(err) }
	signature, err := schnorr.Sign(key, digest)
	if err != nil { t.Fatal(err) }
	tx.TxIn[0].Witness = wire.TxWitness{signature.Serialize()}
	e2eVerifySpend(t, tx, previous)
	return tx
}

func e2eRecover(t *testing.T, transaction baseindex.SPTransaction, scan, spend *btcec.PrivateKey) map[uint32]*btcec.PrivateKey {
	t.Helper()
	point, err := btcec.ParsePubKey(transaction.Tweak)
	if err != nil { t.Fatal(err) }
	secret := e2ePointMul(t, &scan.Key, point).SerializeCompressed()
	found := make(map[uint32]*btcec.PrivateKey)
	for k := uint32(0); k < 2323; k++ {
		var counter [4]byte
		binary.BigEndian.PutUint32(counter[:], k)
		tweak := e2eScalar(t, e2eTagged("BIP0352/SharedSecret", secret, counter[:]))
		tweak.Add(&spend.Key)
		private := btcec.PrivKeyFromScalar(&tweak)
		pub := schnorr.SerializePubKey(private.PubKey())
		matched := false
		for _, output := range transaction.P2TROutputs {
			if bytes.Equal(output.OutputKey, pub) {
				found[output.Vout] = private
				matched = true
			}
		}
		if !matched { break }
	}
	return found
}

func e2eBase(t *testing.T, chain *spTestChain) (*baseindex.BaseIndexer, common.KVDB) {
	t.Helper()
	previous, previousChain := bitcoin_rpc.ShareBitconRpc, common.CHAIN
	bitcoin_rpc.ShareBitconRpc, common.CHAIN = chain, "testnet"
	t.Cleanup(func() { bitcoin_rpc.ShareBitconRpc, common.CHAIN = previous, previousChain })
	kv := indexdb.NewKVDBWithCache(filepath.Join(t.TempDir(), "base"), 1)
	if kv == nil { t.Fatal("open isolated E2E database") }
	b := baseindex.NewBaseIndexer(kv, &chaincfg.RegressionNetParams, 0, 1000)
	b.Init()
	b.SetBlockCallback(func(*common.Block, []*common.Range) {})
	// No automatic durable tip commit in the fixture: tests explicitly choose H.
	b.SetUpdateDBCallback(nil)
	t.Cleanup(func() { b.CloseSilentPayViews(); if err := kv.Close(); err != nil { t.Error(err) } })
	return b, kv
}

func e2eSync(t *testing.T, b *baseindex.BaseIndexer, commit bool) {
	t.Helper()
	if result := b.SyncToChainTip(make(chan struct{})); result != 0 { t.Fatalf("Base sync failed: %d", result) }
	if commit { b.UpdateDB() }
}
