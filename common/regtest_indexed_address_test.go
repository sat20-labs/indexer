package common

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

func TestRegtestIndexedOutputAddress(t *testing.T) {
	previous := CHAIN
	t.Cleanup(func() { CHAIN = previous })
	for _, tc := range []struct {
		chain  string
		params *chaincfg.Params
	}{{"regtest", &chaincfg.RegressionNetParams}, {"testnet", &chaincfg.TestNet4Params}, {"mainnet", &chaincfg.MainNetParams}} {
		t.Run(tc.chain, func(t *testing.T) {
			CHAIN = tc.chain
			address, err := btcutil.NewAddressTaproot(bytes.Repeat([]byte{1}, 32), tc.params)
			if err != nil {
				t.Fatal(err)
			}
			script, err := txscript.PayToAddrScript(address)
			if err != nil {
				t.Fatal(err)
			}
			output := NewTxOutputV2(1234)
			output.OutValue.PkScript = script
			output.AddressType = int(txscript.WitnessV1TaprootTy)
			if got := output.GetAddress(); got != address.EncodeAddress() {
				t.Errorf("indexed address = %s, want %s", got, address.EncodeAddress())
			}
			got, err := GetPkScriptFromAddress(address.EncodeAddress())
			if err != nil || !bytes.Equal(got, script) {
				t.Errorf("stored address script roundtrip = %x, %v; want %x", got, err, script)
			}
		})
	}
}
