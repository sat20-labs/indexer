package atom

const DB_VERSION = "1.0.0"
const DB_VER_KEY = "dbver"
const DB_STATUS_KEY = "status"

const (
	DB_PREFIX_TICKER        = "a-"
	DB_PREFIX_ID_TO_TICKER  = "b-"
	DB_PREFIX_UTXO_BALANCE  = "c-"
	DB_PREFIX_TICKER_UTXO   = "d-"
	DB_PREFIX_HOLDER_ASSET  = "e-"
	DB_PREFIX_TICKER_HOLDER = "f-"
	DB_PREFIX_MINTHISTORY   = "g-"
	DB_PREFIX_ACTION        = "h-"
)

const (
	OpDirectFT    = "ft"
	OpDeployDFT   = "dft"
	OpMintDFT     = "dmt"
	OpSplit       = "y"
	OpCustomColor = "z"
)

const (
	AtomicalsActivationMainnet         = 808080
	AtomicalsActivationDmintMainnet    = 819181
	AtomicalsActivationCommitzMainnet  = 822800
	AtomicalsActivationDensityMainnet  = 828128
	AtomicalsActivationRolloverMainnet = 828628
	AtomicalsActivationColoringMainnet = 848484
	AtomicalsActivationTestnet4        = 27000
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
