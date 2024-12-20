package store

const (
	DB_VERSION     = "0.1.0"
	DB_VERSION_KEY = "runes_db_version"
	STATUS_KEY     = "runes_status"
)

const (
	OUTPOINT_TO_BALANCES   = "o2b-"
	ID_TO_ENTRY            = "i2e-"
	RUNE_TO_ID             = "r2i-"
	TRANSACTION_ID_TO_RUNE = "ti2r-"
	RUNE_LEDGER            = "rl-"
)

const (
	DB_PREFIX_MINT_HISTORY = "m-"
	DB_PREFIX_RUNE         = "r-"
	DB_PREFIX_RUNE_HOLDER  = "h-"
)
