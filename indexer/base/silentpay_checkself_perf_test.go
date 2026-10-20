package base_test

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

// Audit must use only the persisted Base/SP DB; fail the test on any Bitcoin
// RPC method access, including historical block and parent-transaction reads.
type spDelayedParentSource struct {
	bitcoin_rpc.BitcoinRPC
	inFlight atomic.Int32
	maxInFlight atomic.Int32
	calls atomic.Int32
	delay time.Duration
	failID string
	blockCalls atomic.Int32
}

func (s *spDelayedParentSource) GetBlockHash(height uint64) (string, error) {
	s.blockCalls.Add(1)
	return "", errors.New("SP CheckSelf attempted Bitcoin Core block-hash RPC")
}
func (s *spDelayedParentSource) GetRawBlock(hash string) (string, error) {
	s.blockCalls.Add(1)
	return "", errors.New("SP CheckSelf attempted Bitcoin Core block-data RPC")
}
func (s *spDelayedParentSource) GetRawTx(id string) (string, error) {
	n := s.inFlight.Add(1)
	defer s.inFlight.Add(-1)
	s.calls.Add(1)
	for max := s.maxInFlight.Load(); n > max; max = s.maxInFlight.Load() {
		if s.maxInFlight.CompareAndSwap(max, n) {
			break
		}
	}
	time.Sleep(s.delay)
	if id == s.failID {
		return "", errors.New("injected missing Bitcoin parent transaction")
	}
	return s.BitcoinRPC.GetRawTx(id)
}

// All source methods are disabled after the fixture is committed: successful
// CheckSelf proves it did not ask for missing Bitcoin network evidence.
func TestSilentPayDBOnlyCheckSelfNeverCallsBitcoinRPC(t *testing.T) {
	chain := newSPTestChain(t)
	payments := make([]*wire.MsgTx, 0, 8)
	for h := 1; h <= 8; h++ {
		tx, _ := e2ePayment(t, chain.chain[h].Transactions[0], e2eKey(31), e2eKey(32), 1)
		payments = append(payments, tx)
	}
	chain.appendBlock(t, payments...)
	b, _ := e2eBase(t, chain)
	e2eSync(t, b, true)

	source := &spDelayedParentSource{BitcoinRPC: chain, delay: 15 * time.Millisecond}
	bitcoin_rpc.ShareBitconRpc = source
	if !b.CheckSelf() {
		t.Fatal("database-only SP self-check failed")
	}
	if source.calls.Load() != 0 || source.blockCalls.Load() != 0 {
		t.Fatalf("database-only audit performed %d transaction and %d block RPCs", source.calls.Load(), source.blockCalls.Load())
	}
}

func TestSilentPayDBOnlyAuditWorksWithoutHistoricalRawTransaction(t *testing.T) {
	chain := newSPTestChain(t)
	payments := make([]*wire.MsgTx, 0, 8)
	for h := 1; h <= 8; h++ {
		tx, _ := e2ePayment(t, chain.chain[h].Transactions[0], e2eKey(31), e2eKey(32), 1)
		payments = append(payments, tx)
	}
	chain.appendBlock(t, payments...)
	b, _ := e2eBase(t, chain)
	e2eSync(t, b, true)
	source := &spDelayedParentSource{
		BitcoinRPC: chain,
		failID: chain.chain[4].Transactions[0].TxHash().String(),
	}
	bitcoin_rpc.ShareBitconRpc = source
	if !b.CheckSelf() {
		t.Fatal("SP database audit should not request unavailable raw transactions")
	}
	if source.calls.Load() != 0 || source.blockCalls.Load() != 0 {
		t.Fatal("database audit still consulted unavailable Bitcoin RPC data")
	}
}
