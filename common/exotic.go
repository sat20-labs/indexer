package common

const MaxSupply = 2099999997690000

type ExoticRange struct {
	Range      *Range `json:"range"`
	Offset     int64         `json:"offset"`
	Satributes []string      `json:"satributes"`
}


type SatAttr struct {
	Rarity       string `json:"rar,omitempty"` // string
	TrailingZero int    `json:"trz,omitempty"`
	Template     string `json:"tmpl,omitempty"`
	RegularExp   string `json:"reg,omitempty"`
}

