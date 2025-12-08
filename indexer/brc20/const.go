package brc20

const BRC20_DB_VERSION = "1.0.0"
const BRC20_DB_VER_KEY = "dbver"

const (
	DB_PREFIX_MINTHISTORY          = "a-"
	DB_PREFIX_TICKER               = "b-"
	DB_PREFIX_HOLDER_ASSET         = "c-" // asset in a holder
	DB_PREFIX_TRANSFER_HISTORY     = "d-" // BRC20ActionHistory
	DB_PREFIX_CURSE_INSCRIPTION_ID = "e-"
	DB_PREFIX_TICKER_HOLDER        = "f-" // holder in a ticker
	DB_PREFIX_UTXO_TRANSFER        = "g-" // utxo -> transfer nft
)
