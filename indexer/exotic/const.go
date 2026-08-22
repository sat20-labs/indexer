package exotic

const ORDX_DB_VERSION = "1.2.0"
const ORDX_DB_VER_KEY = "dbver"

const (
	DB_PREFIX_MINTHISTORY    = "mh-"
	DB_PREFIX_TICKER         = "tick-"
	DB_PREFIX_TICKER_HOLDER  = "th-" // legacy name: utxo -> HolderInfo
	DB_PREFIX_TICKER_UTXO    = "tu-" // ticker -> utxo -> amount
	DB_PREFIX_HOLDER_BALANCE = "hb-" // ticker -> address -> aggregate amount
	DB_PREFIX_IMAGE          = "img-"
	DB_PREFIX_TICKER_INFO    = "ti-"
)
