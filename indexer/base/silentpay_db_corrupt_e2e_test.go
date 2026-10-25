package base_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/sat20-labs/indexer/common"
	baseindex "github.com/sat20-labs/indexer/indexer/base"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
	"google.golang.org/protobuf/proto"
)

// Exercising the public Base CheckSelf call proves DB-only auditing detects
// incoherence without consulting bitcoind or repairing the broken records.
func TestSilentPayDBOnlyAuditDetectsLocalInconsistency(t *testing.T) {
	chain := newSPTestChain(t)
	payment, _ := e2ePayment(t, chain.chain[1].Transactions[0], e2eKey(31), e2eKey(32), 2)
	chain.appendBlock(t, payment)
	b, kv := e2eBase(t, chain)
	e2eSync(t, b, true)

	spArchive := e2eSPDiskKey("sp/a/", 102, 1)
	spBlock := e2eSPDiskKey("sp/b/", 102, 0)[:len("sp/b/")+4]
	spLiveOrphan := e2eSPDiskKey("sp/l/", 102, 9)
	spArchiveOrphan := e2eSPDiskKey("sp/a/", 102, 9)
	utxoKey := indexdb.GetUTXODBKey(payment.TxHash().String() + ":0")

	archive, err := kv.Read(spArchive)
	if err != nil {
		t.Fatal(err)
	}
	block, err := kv.Read(spBlock)
	if err != nil {
		t.Fatal(err)
	}
	utxoRaw, err := kv.Read(utxoKey)
	if err != nil {
		t.Fatal(err)
	}
	var utxo common.UtxoValueInDB
	if err := indexdb.DecodeBytesWithProto3(utxoRaw, &utxo); err != nil {
		t.Fatal(err)
	}
	addressKey := indexdb.GetAddressIdKey(utxo.AddressId)
	address, err := kv.Read(addressKey)
	if err != nil {
		t.Fatal(err)
	}
	badPoint := append([]byte(nil), archive...)
	for i := 34; i < 66; i++ {
		badPoint[i] = 0xff
	}
	badOutput := append([]byte(nil), archive...)
	badOutput[len(badOutput)-1] ^= 1
	wrongUTXO := proto.Clone(&utxo).(*common.UtxoValueInDB)
	wrongUTXO.UtxoId ^= 1
	badUTXO, err := proto.Marshal(wrongUTXO)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		key      []byte
		value    []byte
		remove   bool
		fullOnly bool
	}{
		{"orphan_live", spLiveOrphan, []byte{1}, false, false},
		{"orphan_archive", spArchiveOrphan, archive, false, false},
		{"missing_sp_block", spBlock, nil, true, false},
		{"missing_base_utxo", utxoKey, nil, true, false},
		{"wrong_base_utxo_id", utxoKey, badUTXO, false, false},
		{"invalid_base_address_script", addressKey, []byte("not-a-bitcoin-address"), false, true},
		{"missing_base_address", addressKey, nil, true, true},
		{"invalid_curve_point", spArchive, badPoint, false, true},
		{"mismatched_output_key", spArchive, badOutput, false, true},
		{"malformed_archive_header", spArchive, []byte{0}, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			old, previousErr := kv.Read(test.key)
			if test.remove {
				if err := kv.Delete(test.key); err != nil {
					t.Fatal(err)
				}
			} else if err := kv.Write(test.key, test.value); err != nil {
				t.Fatal(err)
			}
			// Daily checks retain the identity/liveness closure; full checks
			// additionally reject malformed curve points and address/key damage.
			if test.fullOnly {
				if !b.CheckSelf() {
					t.Fatal("content-only damage failed daily check")
				}
			} else {
				expectSilentPayPanic(t, func() { b.CheckSelf() })
			}
			expectSilentPayPanic(t, func() { _ = b.CheckSilentPayFull(context.Background()) })
			if test.name == "invalid_curve_point" {
				// Public reads must retain strict parsing after a daily check.
				expectSilentPayPanic(t, func() {
					_, _ = b.SilentPayLookup(context.Background(), []baseindex.SPPosition{{Height: 102, BlockHash: chain.chain[102].BlockHash().String(), TxIndex: 1, Vout: 0}})
				})
			}
			after, afterErr := kv.Read(test.key)
			if test.remove {
				if !errors.Is(afterErr, common.ErrKeyNotFound) {
					t.Fatalf("audit rewrote missing row: %v", afterErr)
				}
			} else if afterErr != nil || !bytes.Equal(after, test.value) {
				t.Fatalf("audit changed corrupt row: %v", afterErr)
			}
			if previousErr == nil {
				if err := kv.Write(test.key, old); err != nil {
					t.Fatal(err)
				}
			} else if errors.Is(previousErr, common.ErrKeyNotFound) {
				if err := kv.Delete(test.key); err != nil {
					t.Fatal(err)
				}
			} else {
				t.Fatal(previousErr)
			}
			if !b.CheckSelf() {
				t.Fatal("restored local SP database did not pass self-check")
			}
		})
	}

	// Under the user-approved DB-only contract, a valid but incorrect T
	// cannot be detected without the original transaction input evidence.
	mutated := append([]byte(nil), archive...)
	copy(mutated[33:66], e2eKey(99).PubKey().SerializeCompressed())
	if bytes.Equal(mutated, archive) {
		t.Fatal("fixture did not alter the stored T")
	}
	if err := kv.Write(spArchive, mutated); err != nil {
		t.Fatal(err)
	}
	if !b.CheckSelf() {
		t.Fatal("DB-only mode unexpectedly claims to independently verify BIP352 T")
	}
	if err := b.CheckSilentPayFull(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := kv.Write(spArchive, archive); err != nil {
		t.Fatal(err)
	}
	if !b.CheckSelf() {
		t.Fatal("recovered unmodified records failed DB-only check")
	}
	if _, err := b.SilentPayStatus(context.Background()); err != nil {
		t.Fatalf("read-only audit changed SP service state: %v", err)
	}
	_ = block
	_ = utxoRaw
	_ = address
}
