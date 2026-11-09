package btclucky

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	btcblockchain "github.com/btcsuite/btcd/blockchain"
	btcbtcec "github.com/btcsuite/btcd/btcec/v2"
	btcbtcjson "github.com/btcsuite/btcd/btcjson"
	btcbtcutil "github.com/btcsuite/btcd/btcutil"
	btcchaincfg "github.com/btcsuite/btcd/chaincfg"
	btcchainhash "github.com/btcsuite/btcd/chaincfg/chainhash"
	btcwire "github.com/btcsuite/btcd/wire"
)

func testTemplate() *btcbtcjson.GetBlockTemplateResult {
	coinbaseValue := int64(5000000000)
	return &btcbtcjson.GetBlockTemplateResult{
		Bits:          "207fffff",
		CurTime:       1700000000,
		Height:        1,
		PreviousHash:  "0000000000000000000000000000000000000000000000000000000000000000",
		Transactions:  nil,
		Version:       1,
		CoinbaseValue: &coinbaseValue,
		Target:        "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		WorkID:        "test-work",
	}
}

func templateWithTxs(t *testing.T) (*btcbtcjson.GetBlockTemplateResult, [][]byte) {
	t.Helper()
	tpl := testTemplate()
	rawTxs := make([][]byte, 2)
	for i := range rawTxs {
		tx := btcwire.NewMsgTx(2)
		var prev btcchainhash.Hash
		prev[0] = byte(i + 1)
		tx.AddTxIn(&btcwire.TxIn{
			PreviousOutPoint: btcwire.OutPoint{Hash: prev, Index: uint32(i)},
			SignatureScript:  []byte{0x51},
			Sequence:         ^uint32(0),
			Witness:          btcwire.TxWitness{{byte(i + 1), 0x02}},
		})
		tx.AddTxOut(btcwire.NewTxOut(int64(1000+i), []byte{0x51}))
		var buf bytes.Buffer
		if err := tx.Serialize(&buf); err != nil {
			t.Fatalf("serialize template transaction: %v", err)
		}
		rawTxs[i] = append([]byte(nil), buf.Bytes()...)
		tpl.Transactions = append(tpl.Transactions, btcbtcjson.GetBlockTemplateResultTx{
			Data: hex.EncodeToString(rawTxs[i]),
			Hash: tx.WitnessHash().String(),
			TxID: tx.TxHash().String(),
		})
	}
	return tpl, rawTxs
}

func testRewardAddress(t *testing.T) string {
	t.Helper()
	privKey, _ := btcbtcec.PrivKeyFromBytes(bytes.Repeat([]byte{1}, 32))
	priv, err := btcbtcutil.NewWIF(privKey, &btcchaincfg.MainNetParams, true)
	if err != nil {
		t.Fatalf("NewWIF: %v", err)
	}
	addr, err := btcbtcutil.NewAddressPubKey(priv.PrivKey.PubKey().SerializeCompressed(), &btcchaincfg.MainNetParams)
	if err != nil {
		t.Fatalf("NewAddressPubKey: %v", err)
	}
	return addr.EncodeAddress()
}

func TestResolveJobCount(t *testing.T) {
	n, err := ResolveJobCount("2", 0)
	if err != nil {
		t.Fatalf("ResolveJobCount explicit: %v", err)
	}
	if n != 2 {
		t.Fatalf("explicit jobs = %d, want 2", n)
	}

	n, err = ResolveJobCount("auto", 1<<30)
	if err != nil {
		t.Fatalf("ResolveJobCount auto: %v", err)
	}
	if n != 1 {
		t.Fatalf("auto with huge reserve = %d, want 1", n)
	}

	if _, err := ResolveJobCount("0", 0); err == nil {
		t.Fatalf("expected zero jobs to fail")
	}
}

func TestBlockBytesWireEncoding(t *testing.T) {
	rewardAddr := testRewardAddress(t)
	tplResult, rawTxs := templateWithTxs(t)
	tpl, err := newBlockTemplate(tplResult, "mainnet")
	if err != nil {
		t.Fatalf("newBlockTemplate: %v", err)
	}
	coinbase, header, err := tpl.work(&btcchaincfg.MainNetParams, rewardAddr, 7)
	if err != nil {
		t.Fatalf("BlockTemplate.work: %v", err)
	}
	if got := coinbase.TxOut[0].Value; got != 5000000000 {
		t.Fatalf("coinbase value = %d", got)
	}

	var txBuf bytes.Buffer
	if err := coinbase.Serialize(&txBuf); err != nil {
		t.Fatalf("serialize coinbase: %v", err)
	}
	decoded, err := btcbtcutil.NewTxFromBytes(txBuf.Bytes())
	if err != nil {
		t.Fatalf("decode serialized coinbase as btc tx: %v", err)
	}
	if decoded.MsgTx().TxOut[0].Value != coinbase.TxOut[0].Value {
		t.Fatalf("decoded coinbase value mismatch")
	}
	if !bytes.Contains(coinbase.TxIn[0].SignatureScript, []byte(coinbaseTag)) {
		t.Fatalf("coinbase signature script missing tag %q", coinbaseTag)
	}

	header.Nonce = 11
	blockBytes, err := tpl.blockBytes(coinbase, header)
	if err != nil {
		t.Fatalf("BlockTemplate.blockBytes: %v", err)
	}
	block, err := btcbtcutil.NewBlockFromBytes(blockBytes)
	if err != nil {
		t.Fatalf("decode serialized block as btc block: %v", err)
	}
	if got, want := len(block.Transactions()), len(rawTxs)+1; got != want {
		t.Fatalf("block transaction count = %d, want %d", got, want)
	}
	var rawTxSuffix bytes.Buffer
	for i, tx := range rawTxs {
		got := block.MsgBlock().Transactions[i+1]
		if got.TxHash() != *block.Transactions()[i+1].Hash() {
			t.Fatalf("template transaction %d txid mismatch", i)
		}
		if len(got.TxIn[0].Witness) != 1 || !bytes.Equal(got.TxIn[0].Witness[0], []byte{byte(i + 1), 0x02}) {
			t.Fatalf("template transaction %d witness changed", i)
		}
		_, _ = rawTxSuffix.Write(tx)
	}
	if !bytes.HasSuffix(blockBytes, rawTxSuffix.Bytes()) {
		t.Fatal("block bytes do not preserve template transaction bytes")
	}
	if got, want := header.MerkleRoot, btcblockchain.CalcMerkleRoot(block.Transactions(), false); got != want {
		t.Fatalf("merkle root = %s, want %s", got, want)
	}
}

func TestWorkHeaderHash(t *testing.T) {
	tplResult, _ := templateWithTxs(t)
	tplResult.Transactions = tplResult.Transactions[:1]
	tpl, err := newBlockTemplate(tplResult, "mainnet")
	if err != nil {
		t.Fatalf("newBlockTemplate: %v", err)
	}
	if len(tpl.MerkleBranch) != 1 || tpl.MerkleBranch[0] != tplResult.Transactions[0].TxID {
		t.Fatalf("two-leaf merkle branch = %v, want sibling %s", tpl.MerkleBranch, tplResult.Transactions[0].TxID)
	}
	rewardAddr := testRewardAddress(t)
	coinbase, header, err := tpl.work(&btcchaincfg.MainNetParams, rewardAddr, 7)
	if err != nil {
		t.Fatalf("BlockTemplate.work: %v", err)
	}
	blockBytes, err := tpl.blockBytes(coinbase, header)
	if err != nil {
		t.Fatalf("BlockTemplate.blockBytes: %v", err)
	}
	block, err := btcbtcutil.NewBlockFromBytes(blockBytes)
	if err != nil {
		t.Fatalf("decode serialized block as btc block: %v", err)
	}
	if got, want := header.MerkleRoot, btcblockchain.CalcMerkleRoot(block.Transactions(), false); got != want {
		t.Fatalf("two-leaf merkle root = %s, want %s", got, want)
	}
	header.Nonce = 11
	var data miningHeader
	if err := header.Serialize(bytes.NewBuffer(data[:0])); err != nil {
		t.Fatalf("serialize header: %v", err)
	}
	got, want := data.hashNonce(header.Nonce), header.BlockHash()
	if got != want {
		t.Fatalf("mining header hash = %s, want %s", got, want)
	}
}

func TestTargetFromTemplate(t *testing.T) {
	target, err := targetFromTemplate(testTemplate())
	if err != nil {
		t.Fatalf("targetFromTemplate: %v", err)
	}
	want, _ := new(big.Int).SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	if target.Cmp(want) != 0 {
		t.Fatalf("target = %x, want %x", target, want)
	}

	tpl := testTemplate()
	tpl.Target = ""
	target, err = targetFromTemplate(tpl)
	if err != nil {
		t.Fatalf("targetFromTemplate bits: %v", err)
	}
	if target.Sign() <= 0 {
		t.Fatalf("target from bits must be positive")
	}
}
