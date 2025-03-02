package indexer

import (
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"

	"github.com/sat20-labs/indexer/common"
	mpnCommon "github.com/sat20-labs/indexer/indexer/mpn/common"
)

// HaveBlock returns whether or not the chain instance has the block represented
// by the passed hash.  This includes checking the various places a block can
// be like part of the main chain, on a side chain, or in the orphan pool.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) HaveBlock(hash *chainhash.Hash) (bool, error) {
	return false, nil
}

// CalcSequenceLock computes a relative lock-time SequenceLock for the passed
// transaction using the passed UtxoViewpoint to obtain the past median time
// for blocks in which the referenced inputs of the transactions were included
// within. The generated SequenceLock lock can be used in conjunction with a
// block height, and adjusted median block time to determine if all the inputs
// referenced within a transaction have reached sufficient maturity allowing
// the candidate transaction to be included in a block.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) CalcSequenceLock(tx *btcutil.Tx,
	utxoView *mpnCommon.UtxoViewpoint, mempool bool) (*mpnCommon.SequenceLock, error) {

	return b.calcSequenceLock(nil, tx, utxoView, mempool)
}

// calcSequenceLock computes the relative lock-times for the passed
// transaction. See the exported version, CalcSequenceLock for further details.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *IndexerMgr) calcSequenceLock(node *common.Block, tx *btcutil.Tx,
	utxoView *mpnCommon.UtxoViewpoint, mempool bool) (*mpnCommon.SequenceLock, error) {
	// A value of -1 for each relative lock type represents a relative time
	// lock value that will allow a transaction to be included in a block
	// at any given height or time. This value is returned as the relative
	// lock time in the case that BIP 68 is disabled, or has not yet been
	// activated.
	sequenceLock := &mpnCommon.SequenceLock{Seconds: -1, BlockHeight: -1}

	// The sequence locks semantics are always active for transactions
	// within the mempool.
	csvSoftforkActive := mempool

	// If we're performing block validation, then we need to query the BIP9
	// state.
	//if !csvSoftforkActive {
	// Obtain the latest BIP9 version bits state for the
	// CSV-package soft-fork deployment. The adherence of sequence
	// locks depends on the current soft-fork state.
	// csvState, err := b.deploymentState(node.parent, chaincfg.DeploymentCSV)
	// if err != nil {
	// 	return nil, err
	// }
	// csvSoftforkActive = csvState == ThresholdActive
	//}

	// If the transaction's version is less than 2, and BIP 68 has not yet
	// been activated then sequence locks are disabled. Additionally,
	// sequence locks don't apply to coinbase transactions Therefore, we
	// return sequence lock values of -1 indicating that this transaction
	// can be included within a block at any given height or time.
	mTx := tx.MsgTx()
	sequenceLockActive := uint32(mTx.Version) >= 2 && csvSoftforkActive
	if !sequenceLockActive || mpnCommon.IsCoinBase(tx) {
		return sequenceLock, nil
	}

	// Grab the next height from the PoV of the passed blockNode to use for
	// inputs present in the mempool.
	nextHeight := int32(node.Height + 1)

	for txInIndex, txIn := range mTx.TxIn {
		utxo := utxoView.LookupEntry(txIn.PreviousOutPoint)
		if utxo == nil {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutPoint,
				tx.Hash(), txInIndex)
			return sequenceLock, mpnCommon.MakeRuleError(mpnCommon.ErrMissingTxOut, str)
		}

		// If the input height is set to the mempool height, then we
		// assume the transaction makes it into the next block when
		// evaluating its sequence blocks.
		inputHeight := utxo.BlockHeight()
		if inputHeight == 0x7fffffff {
			inputHeight = nextHeight
		}

		// Given a sequence number, we apply the relative time lock
		// mask in order to obtain the time lock delta required before
		// this input can be spent.
		sequenceNum := txIn.Sequence
		relativeLock := int64(sequenceNum & wire.SequenceLockTimeMask)

		switch {
		// Relative time locks are disabled for this input, so we can
		// skip any further calculation.
		case sequenceNum&wire.SequenceLockTimeDisabled == wire.SequenceLockTimeDisabled:
			continue
		case sequenceNum&wire.SequenceLockTimeIsSeconds == wire.SequenceLockTimeIsSeconds:
			// This input requires a relative time lock expressed
			// in seconds before it can be spent.  Therefore, we
			// need to query for the block prior to the one in
			// which this input was included within so we can
			// compute the past median time for the block prior to
			// the one which included this referenced output.
			prevInputHeight := inputHeight - 1
			if prevInputHeight < 0 {
				prevInputHeight = 0
			}
			// TODO
			// blockNode := node.Ancestor(prevInputHeight)
			// blockNode := b.BlockByHeight(prevInputHeight)
			medianTime := time.Time{} //mpnCommon.CalcPastMedianTime(blockNode)

			// Time based relative time-locks as defined by BIP 68
			// have a time granularity of RelativeLockSeconds, so
			// we shift left by this amount to convert to the
			// proper relative time-lock. We also subtract one from
			// the relative lock to maintain the original lockTime
			// semantics.
			timeLockSeconds := (relativeLock << wire.SequenceLockTimeGranularity) - 1
			timeLock := medianTime.Unix() + timeLockSeconds
			if timeLock > sequenceLock.Seconds {
				sequenceLock.Seconds = timeLock
			}
		default:
			// The relative lock-time for this input is expressed
			// in blocks so we calculate the relative offset from
			// the input's height as its converted absolute
			// lock-time. We subtract one from the relative lock in
			// order to maintain the original lockTime semantics.
			blockHeight := inputHeight + int32(relativeLock-1)
			if blockHeight > sequenceLock.BlockHeight {
				sequenceLock.BlockHeight = blockHeight
			}
		}
	}

	return sequenceLock, nil
}

// IsCurrent returns whether or not the chain believes it is current.  Several
// factors are used to guess, but the key factors that allow the chain to
// believe it is current are:
//   - Latest block height is after the latest checkpoint (if enabled)
//   - Latest block has a timestamp newer than 24 hours ago
//
// This function is safe for concurrent access.
func (b *IndexerMgr) IsCurrent() bool {
	return true
}

// BestSnapshot returns information about the current best chain block and
// related state as of the current point in time.  The returned instance must be
// treated as immutable since it is shared by all callers.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) BestSnapshot() *mpnCommon.BestState {
	return nil
}

// BlockLocatorFromHash returns a block locator for the passed block hash.
// See utils.BlockLocator for details on the algorithm used to create a block locator.
//
// In addition to the general algorithm referenced above, this function will
// return the block locator for the latest known tip of the main (best) chain if
// the passed hash is not currently known.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) BlockLocatorFromHash(hash *chainhash.Hash) mpnCommon.BlockLocator {
	return nil
}

// LatestBlockLocator returns a block locator for the latest known tip of the
// main (best) chain.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) LatestBlockLocator() (mpnCommon.BlockLocator, error) {
	return nil, nil
}

// BlockHeightByHash returns the height of the block with the given hash in the
// main chain.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) BlockHeightByHash(hash *chainhash.Hash) (int32, error) {
	return 0, nil
}

// BlockHashByHeight returns the hash of the block at the given height in the
// main chain.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) BlockHashByHeight(blockHeight int32) (*chainhash.Hash, error) {
	return nil, nil
}

// LocateBlocks returns the hashes of the blocks after the first known block in
// the locator until the provided stop hash is reached, or up to the provided
// max number of block hashes.
//
// In addition, there are two special cases:
//
//   - When no locators are provided, the stop hash is treated as a request for
//     that block, so it will either return the stop hash itself if it is known,
//     or nil if it is unknown
//   - When locators are provided, but none of them are known, hashes starting
//     after the genesis block will be returned
//
// This function is safe for concurrent access.
func (b *IndexerMgr) LocateBlocks(locator mpnCommon.BlockLocator, hashStop *chainhash.Hash,
	maxHashes uint32) []chainhash.Hash {
	return nil
}

// LocateHeaders returns the headers of the blocks after the first known block
// in the locator until the provided stop hash is reached, or up to a max of
// wire.MaxBlockHeadersPerMsg headers.
//
// In addition, there are two special cases:
//
//   - When no locators are provided, the stop hash is treated as a request for
//     that header, so it will either return the header for the stop hash itself
//     if it is known, or nil if it is unknown
//   - When locators are provided, but none of them are known, headers starting
//     after the genesis block will be returned
//
// This function is safe for concurrent access.
func (b *IndexerMgr) LocateHeaders(locator mpnCommon.BlockLocator,
	hashStop *chainhash.Hash) []wire.BlockHeader {
	return nil
}

// BlockByHash returns the block from the main chain with the given hash with
// the appropriate chain height set.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) BlockByHash(hash *chainhash.Hash) (*btcutil.Block, error) {
	// Lookup the block hash in block index and ensure it is in the best
	// chain.

	// Load the block from the database and return it.
	var block *btcutil.Block
	// err := b.db.View(func(dbTx database.Tx) error {
	// 	var err error
	// 	block, err = dbFetchBlockByNode(dbTx, node)
	// 	return err
	// })
	return block, nil
}

func (b *IndexerMgr) BlockByHeight(height int32) (*btcutil.Block, error) {
	// Lookup the block hash in block index and ensure it is in the best
	// chain.

	// Load the block from the database and return it.
	var block *btcutil.Block
	// err := b.db.View(func(dbTx database.Tx) error {
	// 	var err error
	// 	block, err = dbFetchBlockByNode(dbTx, node)
	// 	return err
	// })
	return block, nil
}

// ProcessBlock is the main workhorse for handling insertion of new blocks into
// the block chain.  It includes functionality such as rejecting duplicate
// blocks, ensuring blocks follow all rules, orphan handling, and insertion into
// the block chain along with best chain selection and reorganization.
//
// When no errors occurred during processing, the first return value indicates
// whether or not the block is on the main chain and the second indicates
// whether or not the block is an orphan.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) ProcessBlock(block *btcutil.Block, flags mpnCommon.BehaviorFlags) (bool, bool, error) {
	return true, true, nil
	// b.chainLock.Lock()
	// defer b.chainLock.Unlock()

	// fastAdd := flags&BFFastAdd == BFFastAdd

	// blockHash := block.Hash()
	// common.Log.Tracef("Processing block %v", blockHash)

	// // The block must not already exist in the main chain or side chains.
	// exists, err := b.blockExists(blockHash)
	// if err != nil {
	// 	return false, false, err
	// }
	// if exists {
	// 	str := fmt.Sprintf("already have block %v", blockHash)
	// 	return false, false, MakeRuleError(ErrDuplicateBlock, str)
	// }

	// // The block must not already exist as an orphan.
	// if _, exists := b.orphans[*blockHash]; exists {
	// 	str := fmt.Sprintf("already have block (orphan) %v", blockHash)
	// 	return false, false, MakeRuleError(ErrDuplicateBlock, str)
	// }

	// // Perform preliminary sanity checks on the block and its transactions.
	// err = checkBlockSanity(block, b.chainParams.PowLimit, b.timeSource, flags)
	// if err != nil {
	// 	return false, false, err
	// }

	// // Find the previous checkpoint and perform some additional checks based
	// // on the checkpoint.  This provides a few nice properties such as
	// // preventing old side chain blocks before the last checkpoint,
	// // rejecting easy to mine, but otherwise bogus, blocks that could be
	// // used to eat memory, and ensuring expected (versus claimed) proof of
	// // work requirements since the previous checkpoint are met.
	// blockHeader := &block.MsgBlock().Header
	// checkpointNode, err := b.findPreviousCheckpoint()
	// if err != nil {
	// 	return false, false, err
	// }
	// if checkpointNode != nil {
	// 	// Ensure the block timestamp is after the checkpoint timestamp.
	// 	checkpointTime := time.Unix(checkpointNode.timestamp, 0)
	// 	if blockHeader.Timestamp.Before(checkpointTime) {
	// 		str := fmt.Sprintf("block %v has timestamp %v before "+
	// 			"last checkpoint timestamp %v", blockHash,
	// 			blockHeader.Timestamp, checkpointTime)
	// 		return false, false, MakeRuleError(ErrCheckpointTimeTooOld, str)
	// 	}
	// 	if !fastAdd {
	// 		// Even though the checks prior to now have already ensured the
	// 		// proof of work exceeds the claimed amount, the claimed amount
	// 		// is a field in the block header which could be forged.  This
	// 		// check ensures the proof of work is at least the minimum
	// 		// expected based on elapsed time since the last checkpoint and
	// 		// maximum adjustment allowed by the retarget rules.
	// 		// duration := blockHeader.Timestamp.Sub(checkpointTime)
	// 		// requiredTarget := CompactToBig(b.calcEasiestDifficulty(
	// 		// 	checkpointNode.bits, duration))
	// 		// currentTarget := CompactToBig(blockHeader.Bits)
	// 		// if currentTarget.Cmp(requiredTarget) > 0 {
	// 		// 	str := fmt.Sprintf("block target difficulty of %064x "+
	// 		// 		"is too low when compared to the previous "+
	// 		// 		"checkpoint", currentTarget)
	// 		// 	return false, false, MakeRuleError(ErrDifficultyTooLow, str)
	// 		// }
	// 	}
	// }

	// // Handle orphan blocks.
	// prevHash := &blockHeader.PrevBlock
	// prevHashExists, err := b.blockExists(prevHash)
	// if err != nil {
	// 	return false, false, err
	// }
	// if !prevHashExists {
	// 	common.Log.Infof("Adding orphan block %v with parent %v", blockHash, prevHash)
	// 	b.addOrphanBlock(block)

	// 	return false, true, nil
	// }

	// // The block has passed all context independent checks and appears sane
	// // enough to potentially accept it into the block chain.
	// isMainChain, err := b.maybeAcceptBlock(block, flags)
	// if err != nil {
	// 	return false, false, err
	// }

	// // Accept any orphan blocks that depend on this block (they are
	// // no longer orphans) and repeat for those accepted blocks until
	// // there are no more.
	// err = b.processOrphans(blockHash, flags)
	// if err != nil {
	// 	return false, false, err
	// }

	// common.Log.Debugf("Accepted block %v", blockHash)

	// return isMainChain, false, nil
}

// IsDeploymentActive returns true if the target deploymentID is active, and
// false otherwise.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) IsDeploymentActive(deploymentID uint32) (bool, error) {
	return true, nil

	// b.chainLock.Lock()
	// state, err := b.deploymentState(b.bestChain.Tip(), deploymentID)
	// b.chainLock.Unlock()
	// if err != nil {
	// 	return false, err
	// }

	// return state == ThresholdActive, nil
}

// FetchUtxoView loads unspent transaction outputs for the inputs referenced by
// the passed transaction from the point of view of the end of the main chain.
// It also attempts to fetch the utxos for the outputs of the transaction itself
// so the returned view can be examined for duplicate transactions.
//
// This function is safe for concurrent access however the returned view is NOT.
func (b *IndexerMgr) FetchUtxoView(tx *btcutil.Tx) (*mpnCommon.UtxoViewpoint, error) {
	return nil, nil
	// Create a set of needed outputs based on those referenced by the
	// inputs of the passed transaction and the outputs of the transaction
	// itself.
	// neededLen := len(tx.MsgTx().TxOut)
	// if !IsCoinBase(tx) {
	// 	neededLen += len(tx.MsgTx().TxIn)
	// }
	// needed := make([]wire.OutPoint, 0, neededLen)
	// prevOut := wire.OutPoint{Hash: *tx.Hash()}
	// for txOutIdx := range tx.MsgTx().TxOut {
	// 	prevOut.Index = uint32(txOutIdx)
	// 	needed = append(needed, prevOut)
	// }
	// if !IsCoinBase(tx) {
	// 	for _, txIn := range tx.MsgTx().TxIn {
	// 		needed = append(needed, txIn.PreviousOutPoint)
	// 	}
	// }

	// // Request the utxos from the point of view of the end of the main
	// // chain.
	// view := NewUtxoViewpoint()
	// b.chainLock.RLock()
	// err := view.fetchUtxosFromCache(b.utxoCache, needed)
	// b.chainLock.RUnlock()
	// return view, err
}

// FlushUtxoCache flushes the UTXO state to the database if a flush is needed with the
// given flush mode.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) FlushUtxoCache(mode mpnCommon.FlushMode) error {
	// b.chainLock.Lock()
	// defer b.chainLock.Unlock()

	// return b.db.Update(func(dbTx database.Tx) error {
	// 	return b.utxoCache.flush(dbTx, mode, b.BestSnapshot())
	// })
	return nil
}

// FetchUtxoEntry loads and returns the requested unspent transaction output
// from the point of view of the end of the main chain.
//
// NOTE: Requesting an output for which there is no data will NOT return an
// error.  Instead both the entry and the error will be nil.  This is done to
// allow pruning of spent transaction outputs.  In practice this means the
// caller must check if the returned entry is nil before invoking methods on it.
//
// This function is safe for concurrent access however the returned entry (if
// any) is NOT.
func (b *IndexerMgr) FetchUtxoEntry(outpoint wire.OutPoint) (*mpnCommon.UtxoEntry, error) {
	return nil, nil
}

// Checkpoints returns a slice of checkpoints (regardless of whether they are
// already known).  When there are no checkpoints for the chain, it will return
// nil.
//
// This function is safe for concurrent access.
func (b *IndexerMgr) Checkpoints() []chaincfg.Checkpoint {
	return nil
}

// Subscribe to block chain notifications. Registers a callback to be executed
// when various events take place. See the documentation on Notification and
// NotificationType for details on the types and contents of notifications.
func (b *IndexerMgr) Subscribe(callback mpnCommon.NotificationCallback) {

}

// ConnectBlock must be invoked when a block is extending the main chain.  It
// keeps track of the state of each index it is managing, performs some sanity
// checks, and invokes each indexer.
//
// This is part of the mpnCommon.IndexManager interface.
func (m *IndexerMgr) ConnectBlock( block *btcutil.Block,
	stxos []mpnCommon.SpentTxOut) error {

	// Call each of the currently active optional indexes with the block
	// being connected so they can update accordingly.
	common.Log.Infof("ConnectBlock %v", block)

	return nil
}

// DisconnectBlock must be invoked when a block is being disconnected from the
// end of the main chain.  It keeps track of the state of each index it is
// managing, performs some sanity checks, and invokes each indexer to remove
// the index entries associated with the block.
//
// This is part of the mpnCommon.IndexManager interface.
func (m *IndexerMgr) DisconnectBlock( block *btcutil.Block,
	stxo []mpnCommon.SpentTxOut) error {

	// Call each of the currently active optional indexes with the block
	// being disconnected so they can update accordingly.
	common.Log.Infof("DisconnectBlock %v", block)
	
	return nil
}

// Ensure the Manager type implements the mpnCommon.IndexManager interface.
var _ mpnCommon.IndexManager = (*IndexerMgr)(nil)

