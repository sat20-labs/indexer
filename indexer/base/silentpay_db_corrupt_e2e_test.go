package base_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/sat20-labs/indexer/common"
	indexdb "github.com/sat20-labs/indexer/indexer/db"
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

	tests := []struct {
		name string
		key []byte
		value []byte
		remove bool
	}{
		{"orphan_live", spLiveOrphan, []byte{1}, false},
		{"orphan_archive", spArchiveOrphan, archive, false},
		{"missing_sp_block", spBlock, nil, true},
		{"missing_base_utxo", utxoKey, nil, true},
		{"invalid_base_address_script", addressKey, []byte("not-a-bitcoin-address"), false},
		{"malformed_archive_header", spArchive, []byte{0}, false},
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
			// The DB-only self-check must still reject damaged local SP/Base
			// data even when no Bitcoin Core is available.
			expectSilentPayPanic(t, func() { b.CheckSelf() })
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
