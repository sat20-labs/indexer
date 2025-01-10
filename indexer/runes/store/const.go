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

	// 表: runeid映射mint的utxo (new)
	// 存储: key = rm-%runeid.string()%-%utxo% value = nil
	// 存放时机:
	RUNEID_TO_MINT_HISTORYS = "d-"

	// 表: address和runeid映射mint的utxo (new)
	// 存储: key = arm-%address%-%runeid.string()%-%utxo% value = nil
	ADDRESS_RUNEID_TO_MINT_HISTORYS = "e-"

	// 表: runeid和outpoint映射balance
	// 存储: key = rob-%runeid%-%outpoint%-%lot% value = nil
	RUNEID_OUTPOINT_TO_BALANCE = "f-"

	// 表: runeid和address和outpoint映射balance
	// 存储: key = roab-%runeid%-%addressid%-%lot% value = address & addressid & lot
	RUNEID_ADDRESS_TO_BALANCE = "g-"

	// 表: address和outpoint映射balance
	// 存储: key = roab-%addressid%-%outpoint%-%lot% value = address & runeid & lot
	ADDRESS_OUTPOINT_TO_BALANCE = "h-"
)
