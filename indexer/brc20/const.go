package brc20

const BRC20_DB_VERSION = "1.0.0"
const BRC20_DB_VER_KEY = "dbver"
const BRC20_DB_STATUS_KEY = "status"

const (
	DB_PREFIX_MINTHISTORY          = "a-"
	DB_PREFIX_TICKER               = "b-"
	DB_PREFIX_HOLDER_ASSET         = "c-" // asset in a holder
	DB_PREFIX_TRANSFER_HISTORY     = "d-" // BRC20ActionHistory 按区块顺序排序
	DB_PREFIX_CURSE_INSCRIPTION_ID = "e-"
	DB_PREFIX_TICKER_HOLDER        = "f-" // holder in a ticker
	DB_PREFIX_UTXO_TRANSFER        = "g-" // utxo -> transfer nft
	DB_PREFIX_TRANSFER_HISTORY_HOLDER  = "h-" // +addressId+ticker+nftId 个人历史数据，value: inscribe utxoId + transfer utxoId, 用于构造 DB_PREFIX_TRANSFER_HISTORY
)
