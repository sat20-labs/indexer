package atom

import "github.com/sat20-labs/indexer/common"

type ActivationHeights struct {
	Activation     int
	Dmint          int
	Commitz        int
	Density        int
	Rollover       int
	CustomColoring int
}

type Status struct {
	Version     string
	Height      int
	TickerCount int64
	MintCount   int64
	ActionCount int64
}

func (s *Status) Clone() *Status {
	if s == nil {
		return &Status{Version: DB_VERSION}
	}
	n := *s
	return &n
}

type Ticker struct {
	Id             int64
	AtomicalId     string
	LocationId     string
	Name           string
	DisplayName    string
	Subtype        string
	MintMode       string
	MintAmount     int64
	MintHeight     int64
	MaxMints       int64
	MaxMintsGlobal int64
	MaxSupply      int64
	MintedTimes    int64
	MintedAmount   int64
	DeployHeight   int
	DeployTime     int64
	DeployTx       string
	DeployIndex    int
	CommitTx       string
	CommitTxIndex  int
	CommitIndex    int
	CommitHeight   int
	Bitworkc       string
	Bitworkr       string
	MintBitworkc   string
	MintBitworkr   string
	Bv             string
	Bci            int64
	Bri            int64
	Bcs            int64
	Brs            int64
	HolderCount    int
}

func (t *Ticker) Clone() *Ticker {
	if t == nil {
		return nil
	}
	n := *t
	return &n
}

type UtxoBalance struct {
	UtxoId     uint64
	AddressId  uint64
	Outpoint   string
	AtomicalId string
	Ticker     string
	Amount     int64
}

func (b *UtxoBalance) Clone() *UtxoBalance {
	if b == nil {
		return nil
	}
	n := *b
	return &n
}

type MintInfo struct {
	Id         int64
	AtomicalId string
	LocationId string
	Ticker     string
	AddressId  uint64
	UtxoId     uint64
	Outpoint   string
	Amount     int64
	Height     int
	TxIndex    int
	TxId       string
}

func (m *MintInfo) Clone() *MintInfo {
	if m == nil {
		return nil
	}
	n := *m
	return &n
}

func (m *MintInfo) ToCommon(address string) *common.MintInfo {
	return &common.MintInfo{
		Id:            m.Id,
		Address:       address,
		Amount:        common.NewDefaultDecimal(m.Amount).String(),
		Height:        m.Height,
		InscriptionId: m.LocationId,
	}
}

type Operation struct {
	Op          string
	Payload     *Payload
	InputIndex  int
	CommitTxId  string
	CommitIndex int
}

type Payload struct {
	Args map[string]any
	Raw  []byte
}

type HolderAction struct {
	Action    int
	Ticker    string
	UtxoId    uint64
	AddressId uint64
	Amount    int64
}

type ActionHistory struct {
	Id         int64
	Height     int
	TxIndex    int
	TxId       string
	Ticker     string
	AtomicalId string
	FromUtxo   uint64
	ToUtxo     uint64
	FromAddr   uint64
	ToAddr     uint64
	Amount     int64
	Action     string
}
