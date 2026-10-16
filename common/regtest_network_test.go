package common

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func TestRegtestAddressAndNodeIdentities(t *testing.T) {
	previousChain := CHAIN
	t.Cleanup(func() { CHAIN = previousChain })
	key := bytes.Repeat([]byte{1}, 32)
	for _, tc := range []struct {
		chain  string
		params *chaincfg.Params
	}{
		{"regtest", &chaincfg.RegressionNetParams},
		{"mainnet", &chaincfg.MainNetParams},
		{"testnet", &chaincfg.TestNet4Params},
		{"testnet4", &chaincfg.TestNet4Params},
	} {
		t.Run(tc.chain, func(t *testing.T) {
			address, err := btcutil.NewAddressTaproot(key, tc.params)
			if err != nil {
				t.Fatal(err)
			}
			valid, err := IsValidAddr(address.EncodeAddress(), tc.chain)
			if err != nil || !valid {
				t.Fatalf("valid address rejected: valid=%v err=%v", valid, err)
			}
			script, err := AddrToPkScriptV2(address.EncodeAddress(), tc.chain)
			if err != nil {
				t.Fatal(err)
			}
			roundtrip, err := PkScriptToAddr(script, tc.params)
			if err != nil || roundtrip != address.EncodeAddress() {
				t.Fatalf("address roundtrip=%s err=%v", roundtrip, err)
			}
		})
	}
	CHAIN = "regtest"
	if IsMainnet() || GetBootstrapPubKey() != _bootstrapPubKey_testnet || GetCoreNodePubKey() != _coreNodePubKey_testnet {
		t.Fatal("regtest selected production node identities")
	}
}
