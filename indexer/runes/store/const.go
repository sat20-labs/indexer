package store

const (
	DB_VERSION     = "0.1.0"
	DB_VERSION_KEY = "runes_db_version"
	STATUS_KEY     = "runes_status"
)

const (
	// 表: runeId映射runeEntry表
	// 存储: key = i2r-%runeId% value = runeEntry
	ID_TO_ENTRY = "a-"

	// 表: runeEntry映射RuneId表
	// 存储: key = r2i-%runeEntry% value = runeId
	RUNE_TO_ID = "b-"

	// 表: utox映射rune资产表
	// 存储: key = o2b-%outpoint% value = runeIdLot
	OUTPOINT_TO_BALANCES = "c-"

	// 表: runeid映射拥有rune资产的address (new)
	// 存储: key = rab-%runeid.string()-%address% value = nil
	// 存放时机: 当rune资产发生变化时,需要更新这个表
	RUNEID_TO_ADDRESS = "d-"

	// 表: runeid映射拥有rune资产的utxo (new)
	// 存储: key = rub-%runeid.string()%-%utxo% value = nil
	// 存放时机: 当rune资产发生变化时,需要更新这个表
	RUNEID_TO_UTXO = "e-"

	// 表: runeid映射mint的utxo (new)
	// 存储: key = rm-%runeid.string()%-%utxo% value = nil
	// 存放时机:
	RUNEID_TO_MINT_HISTORYS = "f-"

	// 表: address和runeid映射mint的utxo (new)
	// 存储: key = arm-%address%-%runeid.string()%-%utxo% value = nil
	ADDRESS_RUNEID_TO_MINT_HISTORYS = "g-"

	// 表: runeid和outpoint映射balance (new)
	// 存储: key = rob-%runeid%-%outpoint%-%lot% value = nil
	RUNEID_TO_OUTPOINT_TO_BALANCE = "h-"
)
