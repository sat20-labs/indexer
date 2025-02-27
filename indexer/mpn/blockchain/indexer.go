package blockchain

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	utils "github.com/sat20-labs/indexer/indexer/mpn/utils"
)

// IndexManager provides a generic interface that the is called when blocks are
// connected and disconnected to and from the tip of the main chain for the
// purpose of supporting optional indexes.
type IndexManager interface {

	// ConnectBlock is invoked when a new block has been connected to the
	// main chain. The set of output spent within a block is also passed in
	// so indexers can access the previous output scripts input spent if
	// required.
	ConnectBlock(*btcutil.Block, []SpentTxOut) error

	// DisconnectBlock is invoked when a block has been disconnected from
	// the main chain. The set of outputs scripts that were spent within
	// this block is also returned so indexers can clean up the prior index
	// state for this block.
	DisconnectBlock(*btcutil.Block, []SpentTxOut) error

	HaveBlock(hash *chainhash.Hash) (bool, error)
	CalcSequenceLock(tx *btcutil.Tx, utxoView *UtxoViewpoint, mempool bool) (*utils.SequenceLock, error)
	IsCurrent() bool
	BestSnapshot() *utils.BestState
	BlockLocatorFromHash(hash *chainhash.Hash) utils.BlockLocator
	LatestBlockLocator() (utils.BlockLocator, error)
	BlockHeightByHash(hash *chainhash.Hash) (int32, error)
	BlockHashByHeight(blockHeight int32) (*chainhash.Hash, error)
	LocateBlocks(locator utils.BlockLocator, hashStop *chainhash.Hash, 
		maxHashes uint32) []chainhash.Hash
	LocateHeaders(locator utils.BlockLocator, hashStop *chainhash.Hash) []wire.BlockHeader
	BlockByHash(hash *chainhash.Hash) (*btcutil.Block, error)
	BlockByHeight(height int32) (*btcutil.Block, error)
	ProcessBlock(block *btcutil.Block, flags BehaviorFlags) (bool, bool, error)
	IsDeploymentActive(deploymentID uint32) (bool, error)
}
