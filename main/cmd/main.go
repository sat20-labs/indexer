package main

import (
	"bytes"
	"encoding/hex"
	"math"
	"strconv"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/mempool"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
)

func main() {

	const (
		P2PKH int = iota
		P2WPKH
		P2TR // pttr
		P2SH_P2WPKH
		M44_P2WPKH // m44
		M44_P2TR   // m44_pttr
	)
	// Pay-to-Public-Key-Hash  （P2PKH）    OK  Legacy
	// pay-to-script-hash       (P2SH)    OK  Nested segwit
	// pay-to-witness-pubkey-hash (P2WKH) OK  native segwit
	// pay-to-witness-script-hash (P2WSH)  --- p2wsh Address  OK
	// pay-to-taproot (P2TR)  --- Taproot Address  OK

	var pkScript []byte
	var addrType int
	switch {
	case txscript.IsPayToPubKey(pkScript):
		common.Log.Debugf("The pkScript IsPayToPubKey: P2PK") // ok, 第一代，可能没人用
	case txscript.IsPayToPubKeyHash(pkScript):
		common.Log.Debugf("The pkScript IsPayToPubKeyHash: P2PKH") // ok
	case txscript.IsPayToScriptHash(pkScript):
		common.Log.Debugf("The pkScript IsPayToScriptHash: P2SH") // ok
	case txscript.IsPayToWitnessScriptHash(pkScript):
		common.Log.Debugf("The pkScript IsPayToWitnessScriptHash: P2WSH") // ok
	case txscript.IsPayToWitnessPubKeyHash(pkScript):
		common.Log.Debugf("The pkScript IsPayToWitnessPubKeyHash: P2WKH") // ok
	case txscript.IsPayToTaproot(pkScript):
		addrType = P2TR
		common.Log.Debugf("The pkScript IsTaprootKeyHash: PTTR, addrType: %d", addrType)
	case txscript.IsWitnessProgram(pkScript):
		addrType = P2WPKH // ?
		common.Log.Debugf("The pkScript IsWitnessProgram: P2WPKH, addrType: %d", addrType)
	case txscript.IsNullData(pkScript):
		common.Log.Debugf("The pkScript IsNullData: NullData")
	case txscript.IsPushOnlyScript(pkScript):
		common.Log.Debugf("The pkScript IsPushOnlyScript: PushOnlyScript")
	case txscript.IsUnspendable(pkScript):
		common.Log.Debugf("The pkScript IsUnspendable: Unspendable")
	default:
		common.Log.Debugf("The pkScript is invalid")

	}
}

func main1() {
	raw := "70736274ff0100b2020000000218de92a8e808a35d35637c0fbccc612ceb32c87418ab344595f0670c9f33d7bc0000000000ffffffff921bebb771fe518d1a27a45616132e3f75f52bc69c86970a46bdaa0f27e070a50100000000ffffffff02e8030000000000002251201eca94fc175e45d42a907e97eabf3ec76a3237653537cc0f11faf4dfd8c0e1003f2000000000000022512038bea4bdaef65d6dbb9444f7fcc65fd00270491cae166a069a117cd674ba20a3000000000001012be80300000000000022512038bea4bdaef65d6dbb9444f7fcc65fd00270491cae166a069a117cd674ba20a30108420140dc270d427238ec496d6c864c3401e425b28eafbde996d2f5c1358f4a9777a74e952b0287f96ab3e5a54b8391d88a8d8fd723ff4c3d8ed6844c4947b84b3699090001012b0f7300000000000022512038bea4bdaef65d6dbb9444f7fcc65fd00270491cae166a069a117cd674ba20a30108420140675748bbc6342abfa5417a33543703ff13c70139a603ac0018137408daf9c1a1e007cbea44a161c2f178f01dc189d9e00493363ab07b24d815289ed0e0816a54000000"
	psbtBytes, err := hex.DecodeString(raw)
	if err != nil {
		common.Log.Errorf("Invalid pending order raw: %v.", err)
	}
	pb, err := psbt.NewFromRawBytes(
		bytes.NewReader(psbtBytes), false,
	)
	if err != nil {
		common.Log.Errorf("Invalid pending order raw: %v.", err)
	}

	msgTx, err := psbt.Extract(pb)
	if err != nil {
		common.Log.Errorf("Invalid pending order raw: %v.", err)
		return
	}
	var buf bytes.Buffer
	err = msgTx.Serialize(&buf)
	if err != nil {
		common.Log.Errorf("Invalid pending order raw: %v.", err)
	}
	tx, err := btcutil.NewTxFromBytes(buf.Bytes())
	if err != nil {
		common.Log.Errorf("Invalid pending order raw: %v.", err)
		return
	}
	vsize := mempool.GetTxVirtualSize(tx)
	common.Log.Infof("vs: %d", vsize)

	fee, err := pb.GetTxFee()
	if err != nil {
		common.Log.Errorf("Invalid pending order raw: %v.", err)
		return
	}

	result := strconv.Itoa(int(math.Round(float64(fee) / float64(vsize))))
	common.Log.Infof("fee: %d， result: %s", fee, result)
	LogPsbt(pb)
}

func LogPsbt(pb *psbt.Packet) {
	common.Log.Infof("psbt:")
	common.Log.Infof("UnsignedTx:")
	logMsgTx(pb.UnsignedTx)

	//common.Log.Infof("Inputs: %v", pb.Inputs)
	common.Log.Infof("pb inputs account: %d", len(pb.Inputs))
	for index, input := range pb.Inputs {
		common.Log.Infof("pb inputs index: %d", index)
		logpbInput(&input)
	}

	//common.Log.Infof("Outputs: %v", pb.Outputs)
	common.Log.Infof("pb outputs account: %d", len(pb.Outputs))
	for index, output := range pb.Outputs {
		common.Log.Infof("pb outputs index: %d", index)
		logpbOutout(&output)
	}

	common.Log.Infof("Unknowns: %+v", pb.Unknowns)

}

func logMsgTx(tx *wire.MsgTx) {
	common.Log.Infof("tx:")
	common.Log.Infof("txin: %d", len(tx.TxIn))
	for index, txin := range tx.TxIn {
		common.Log.Infof("      txin index: %d", index)
		common.Log.Infof("      txin utxo txid: %s", txin.PreviousOutPoint.Hash.String())
		common.Log.Infof("      txin utxo index: %d", txin.PreviousOutPoint.Index)
		common.Log.Infof("      txin utxo Wintness: ")
		common.Log.Infof("      {")
		for _, witness := range txin.Witness {
			common.Log.Infof("      %x", witness)
		}
		common.Log.Infof("      }")
		common.Log.Infof("      txin SignatureScript: %x", txin.SignatureScript)
		common.Log.Infof("      ---------------------------------")
	}

	common.Log.Infof("txout: %d", len(tx.TxOut))
	for index, txout := range tx.TxOut {
		common.Log.Infof("      txout index: %d", index)
		common.Log.Infof("      txout pkscript: %x", txout.PkScript)
		addr, err := PkScriptToAddr(txout.PkScript)
		if err != nil {
			common.Log.Errorf("     txout pkscript is an invalidaddress: %s", err)
		} else {
			common.Log.Infof("      txout address: %s", addr)
		}
		common.Log.Infof("      txout value: %d", txout.Value)
		common.Log.Infof("      ---------------------------------")
	}
}

func logpbInput(pbinput *psbt.PInput) {
	common.Log.Infof("pbinput:")
	if pbinput.WitnessUtxo != nil {
		common.Log.Infof("pbinput.WitnessUtxo:")
		common.Log.Infof("      pkScript: %x", pbinput.WitnessUtxo.PkScript)
		addr, err := PkScriptToAddr(pbinput.WitnessUtxo.PkScript)
		if err != nil {
			common.Log.Errorf("     txout pkscript is an invalidaddress: %s", err)
		} else {
			common.Log.Infof("      txout address: %s", addr)
		}
		common.Log.Infof("      txout value: %d", pbinput.WitnessUtxo.Value)
	}

	for index, partialSig := range pbinput.PartialSigs {
		common.Log.Infof("  index %d partialSig: %x", index, partialSig)
	}

	common.Log.Infof("      SighashType: %d", pbinput.SighashType)
	common.Log.Infof("      RedeemScript: %x", pbinput.RedeemScript)
	common.Log.Infof("      SighashType: %x", pbinput.SighashType)
	for index, bip32Derivation := range pbinput.Bip32Derivation {
		common.Log.Infof("  index %d Bip32Derivation: %x", index, bip32Derivation)
	}

	common.Log.Infof("      FinalScriptSig: %x", pbinput.FinalScriptSig)
	common.Log.Infof("      FinalScriptWitness: %x", pbinput.FinalScriptWitness)
	common.Log.Infof("      TaprootKeySpendSig: %x", pbinput.TaprootKeySpendSig)
	common.Log.Infof("      SighashType: %d", pbinput.SighashType)

}

func logpbOutout(pboutput *psbt.POutput) {
	common.Log.Infof("pboutput:")
	common.Log.Infof("      RedeemScript: %x", pboutput.RedeemScript)
	common.Log.Infof("      WitnessScript: %x", pboutput.WitnessScript)
	for index, bip32Derivation := range pboutput.Bip32Derivation {
		common.Log.Infof("  index %d Bip32Derivation: %x", index, bip32Derivation)
	}
	common.Log.Infof("      TaprootInternalKey: %x", pboutput.TaprootInternalKey)
	common.Log.Infof("      TaprootTapTree: %x", pboutput.TaprootTapTree)

	for index, bip32Derivation := range pboutput.TaprootBip32Derivation {
		common.Log.Infof("  index %d TaprootBip32Derivation: %x", index, bip32Derivation)
	}

}

func PkScriptToAddr(pkScript []byte) (string, error) {
	NetParams := &chaincfg.TestNet3Params
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(pkScript, NetParams)
	if err != nil {
		return "", err
	}

	return addrs[0].EncodeAddress(), nil
}
