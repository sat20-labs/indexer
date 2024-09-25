package define

type PlainUtxo struct {
	Txid  string `json:"txid"`
	Vout  int    `json:"vout"`
	Value int64  `json:"value"`
}

type SatInfo struct {
	Sat        int64    `json:"sat"`
	Height     int64    `json:"height"`
	Cycle      int64    `json:"cycle"`
	Epoch      int64    `json:"epoch"`
	Period     int64    `json:"period"`
	Satributes []string `json:"satributes"`
}

type SpecificSatInUtxo struct {
	Utxo        string     `json:"utxo"`
	Value       int64      `json:"value"`
	SpecificSat int64      `json:"specificsat"`
	Sats        []SatRange `json:"sats"`
}

type SpecificSat struct {
	Address     string     `json:"address"`
	Utxo        string     `json:"utxo"`
	Value       int64      `json:"value"`
	SpecificSat int64      `json:"specificsat"`
	Sats        []SatRange `json:"sats"`
}

type SatRange struct {
	Start  int64 `json:"start"`
	Size   int64 `json:"size"`
	Offset int64 `json:"offset"`
}

type SatributeRange struct {
	SatRange
	Satributes []string `json:"satributes"`
}

type SatDetailInfo struct {
	SatributeRange
	Block int `json:"block"`
	// Time  int64 `json:"time"`
}

type ExoticSatRangeUtxo struct {
	Utxo  string          `json:"utxo"`
	Value int64           `json:"value"`
	Sats  []SatDetailInfo `json:"sats"`
}

type SpecificExoticUtxo struct {
	Utxo   string     `json:"utxo"`
	Value  int64      `json:"value"`
	Type   string     `json:"type"`
	Amount int64      `json:"amount"`
	Sats   []SatRange `json:"sats"`
}
