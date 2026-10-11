package common

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

func spUnhex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil { t.Fatal(err) }
	return b
}

// Golden data: bitcoin/bips bip-0352/send_and_receive_test_vectors.json,
// "Simple send: two inputs", BIP352 1.1.1 (BSD-2-Clause).
func TestSilentPayOfficialTwoInputs(t *testing.T) {
	ids := []string{"f4184fc596403b9d638783cf57adfe4c75c605f6356fbc91338530e9831e9e16", "a1075db55d416d3ca199f55b6084e2115b9345e16c5cf302fc80e9d5fbf5d48d"}
	scripts := []string{
		"483046022100ad79e6801dd9a8727f342f31c71c4912866f59dc6e7981878e92c5844a0ce929022100fb0d2393e813968648b9753b7e9871d90ab3d815ebf91820d704b19f4ed224d621025a1e61f898173040e20616d43e9f496fba90338a39faa1ed98fcbaeee4dd9be5",
		"48304602210086783ded73e961037e77d49d9deee4edc2b23136e9728d56e4491c80015c3a63022100fda4c0f21ea18de29edbce57f7134d613e044ee150a89e2e64700de2d4e83d4e2103bd85685d03d111699b15d046319febe77f8de5286e9e512703cdee1bf3be3792",
	}
	prev := []string{"76a91419c2f3ae0ca3b642bd3e49598b8da89f50c1416188ac", "76a914d9317c66f54ff0a152ec50b1d19c25be50c8e15988ac"}
	inputs := make([]SPInput, 2)
	for i := range inputs {
		h, err := chainhash.NewHashFromStr(ids[i]); if err != nil { t.Fatal(err) }
		inputs[i] = SPInput{Outpoint: wire.OutPoint{Hash:*h}, SignatureScript:spUnhex(t,scripts[i]), Prevout:&wire.TxOut{PkScript:spUnhex(t,prev[i])}}
	}
	want := "024ac253c216532e961988e2a8ce266a447c894c781e52ef6cee902361db960004"
	for n := 0; n < 2; n++ {
		got, ok, err := SPComputeTweak(inputs)
		if err != nil || !ok || hex.EncodeToString(got[:]) != want { t.Fatalf("tweak=%x eligible=%v err=%v",got,ok,err) }
		inputs[0],inputs[1] = inputs[1],inputs[0]
	}
	inputs[0].SignatureScript = append([]byte{0x00,0x75}, inputs[0].SignatureScript...)
	got,ok,err:=SPComputeTweak(inputs)
	if err!=nil || !ok || hex.EncodeToString(got[:])!=want { t.Fatalf("non-template scriptSig: %x %v %v",got,ok,err) }
}

func spKeyInput(n byte, xonly bool) SPInput {
	k:=make([]byte,32); k[31]=n
	_,p:=btcec.PrivKeyFromBytes(k)
	pk:=p.SerializeCompressed()
	in:=SPInput{Outpoint:wire.OutPoint{Hash:chainhash.Hash{n}},Prevout:&wire.TxOut{},Witness:wire.TxWitness{[]byte{1}}}
	if xonly { in.Prevout.PkScript=append([]byte{0x51,0x20},pk[1:]...) } else {
		in.Prevout.PkScript=append(append([]byte{0x76,0xa9,0x14},btcutil.Hash160(pk)...),0x88,0xac)
		in.SignatureScript=append([]byte{0,0x21},pk...)
	}
	return in
}

func TestSilentPayIntermediateInfinityAndDuplicateKeys(t *testing.T) {
	a:=spKeyInput(7,false)
	b:=a; b.Outpoint.Hash[0]=8
	b.SignatureScript=append([]byte(nil),a.SignatureScript...)
	b.SignatureScript[2]^=1
	b.Prevout=&wire.TxOut{PkScript:append(append([]byte{0x76,0xa9,0x14},btcutil.Hash160(b.SignatureScript[2:])...),0x88,0xac)}
	c:=spKeyInput(9,false)
	zero,ok,err:=SPComputeTweak([]SPInput{a,b})
	if err!=nil || ok || zero!=([33]byte{}) { t.Fatalf("final infinity: %x %v %v",zero,ok,err) }
	got,ok,err:=SPComputeTweak([]SPInput{a,b,c})
	if err!=nil || !ok { t.Fatalf("intermediate infinity rejected: %v",err) }
	pub,err:=SPInputPublicKey(c); if err!=nil { t.Fatal(err) }
	want,err:=spTweakFromAggregate([]wire.OutPoint{a.Outpoint,b.Outpoint,c.Outpoint},pub)
	if err!=nil || got!=want { t.Fatalf("intermediate infinity: got=%x want=%x err=%v",got,want,err) }
	d:=a; d.Outpoint.Hash[0]=10
	dup,ok,err:=SPComputeTweak([]SPInput{a,d})
	if err!=nil || !ok || dup==([33]byte{}) { t.Fatalf("duplicate pubkeys must both contribute: %v",err) }
	pk,_:=SPInputPublicKey(a)
	one,err:=spTweakFromAggregate([]wire.OutPoint{a.Outpoint,d.Outpoint},pk)
	if err!=nil || dup==one { t.Fatal("duplicate pubkeys were deduplicated") }
}

func TestSilentPayInputTypesAndNUMS(t *testing.T) {
	in:=spKeyInput(3,false)
	pub:=append([]byte(nil),in.SignatureScript[2:]...)
	p2w:=in; p2w.SignatureScript=nil; p2w.Witness=wire.TxWitness{[]byte{1},pub}; p2w.Prevout=&wire.TxOut{PkScript:append([]byte{0,20},btcutil.Hash160(pub)...)}
	p2sh:=p2w; p2sh.SignatureScript=append([]byte{22},p2w.Prevout.PkScript...); p2sh.Prevout=&wire.TxOut{PkScript:append(append([]byte{0xa9,20},btcutil.Hash160(p2w.Prevout.PkScript)...),0x87)}
	for _,v:=range []SPInput{in,p2w,p2sh} {
		pk,err:=SPInputPublicKey(v); if err!=nil || pk==nil || !bytes.Equal(pk.SerializeCompressed(),pub) { t.Fatalf("key extraction: %v",err) }
	}
	tr:=spKeyInput(5,true)
	control:=append([]byte{0xc0},SPNUMSInternalKey[:]...)
	tr.Witness=wire.TxWitness{[]byte{0x51},control,[]byte{0x50,1,2}}
	copyBefore:=append([]byte(nil),control...)
	pk,err:=SPInputPublicKey(tr)
	if err!=nil || pk!=nil { t.Fatalf("NUMS scriptpath not excluded: %v",err) }
	if len(tr.Witness)!=3 || !bytes.Equal(control,copyBefore) { t.Fatal("input witness mutated") }
	tr.Witness=wire.TxWitness{[]byte{1}}
	pk,err=SPInputPublicKey(tr); if err!=nil || pk==nil || pk.SerializeCompressed()[0]!=2 { t.Fatalf("taproot parity: %v",err) }
	tr.Witness=wire.TxWitness{[]byte{1},[]byte{0x51},append([]byte{0xc0},pub[1:]...)}
	pk,err=SPInputPublicKey(tr); if err!=nil || pk==nil { t.Fatalf("non-NUMS scriptpath excluded: %v",err) }
}

func TestSilentPayMissingDataAndFutureWitness(t *testing.T) {
	in:=spKeyInput(1,true)
	future:=SPInput{Outpoint:wire.OutPoint{Hash:chainhash.Hash{2}},Prevout:&wire.TxOut{PkScript:append([]byte{0x52,0x20},make([]byte,32)...)}}
	_,ok,err:=SPComputeTweak([]SPInput{in,future}); if err!=nil || ok { t.Fatalf("future witness included: %v",err) }
	missing:=future; missing.Prevout=nil
	_,_,err=SPComputeTweak([]SPInput{in,missing}); if err==nil { t.Fatal("missing prevout silently skipped") }
	in.Outpoint=wire.OutPoint{Index:0xffffffff}
	_,ok,err=SPComputeTweak([]SPInput{in}); if err!=nil || ok { t.Fatalf("coinbase eligible: %v",err) }
	// A valid uncompressed P2PKH spend is not an eligible SP input.
	legacy:=spKeyInput(2,false); k:=make([]byte,32);k[31]=2; _,p:=btcec.PrivKeyFromBytes(k)
	uncompressed:=p.SerializeUncompressed()
	legacy.SignatureScript=append([]byte{0,65},uncompressed...)
	legacy.Prevout=&wire.TxOut{PkScript:append(append([]byte{0x76,0xa9,20},btcutil.Hash160(uncompressed)...),0x88,0xac)}
	_,ok,err=SPComputeTweak([]SPInput{legacy}); if err!=nil || ok { t.Fatalf("legacy uncompressed: %v",err) }
}
