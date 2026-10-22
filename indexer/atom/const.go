package atom

import "github.com/sat20-labs/indexer/indexer/atom/coloring"

const DB_VERSION = "1.1.0"
const DB_VER_KEY = "dbver"
const DB_STATUS_KEY = "status"

const (
	DB_PREFIX_TICKER              = "a-"
	DB_PREFIX_ID_TO_TICKER        = "b-"
	DB_PREFIX_UTXO_BALANCE        = "c-"
	DB_PREFIX_TICKER_UTXO         = "d-"
	DB_PREFIX_HOLDER_ASSET        = "e-"
	DB_PREFIX_TICKER_HOLDER       = "f-"
	DB_PREFIX_MINTHISTORY         = "g-"
	DB_PREFIX_ACTION              = "h-"
	DB_PREFIX_ADDRESS_MINTHISTORY = "i-"
)

const (
	OpDirectFT    = "ft"
	OpDeployDFT   = "dft"
	OpMintDFT     = "dmt"
	OpSplit       = "y"
	OpCustomColor = "z"
)

const (
	AtomicalsActivationMainnet         = coloring.ActivationMainnet
	AtomicalsActivationDmintMainnet    = coloring.DmintMainnet
	AtomicalsActivationCommitzMainnet  = coloring.CommitzMainnet
	AtomicalsActivationDensityMainnet  = coloring.DensityMainnet
	AtomicalsActivationRolloverMainnet = coloring.RolloverMainnet
	AtomicalsActivationColoringMainnet = coloring.CustomColoringMainnet
	AtomicalsActivationTestnet4        = coloring.ActivationTestnet4
)

const (
	MintGeneralDelayBlocks = 100
	MintTickerDelayBlocks  = 3
	DFTMintAmountMin       = 546
	DFTMintAmountMax       = 100000000
	DFTMaxMintsMin         = 1
	DFTMaxMintsLegacy      = 500000
	DFTMaxMintsDensity     = 21000000
	DFTMintHeightMin       = 0
	DFTMintHeightMax       = 10000000
)
