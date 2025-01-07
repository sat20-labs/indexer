package runes

import (
	"encoding/json"

	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

const defaultRuneSymbol = '\u29C9'

type MintInfo struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Amount    uint128.Uint128
	Mints     uint128.Uint128
	Cap       uint128.Uint128
	Remaining uint128.Uint128
	Mintable  bool    `json:"mintable"`
	Progress  float64 `json:"progress"`
}

func (s MintInfo) MarshalJSON() ([]byte, error) {
	type Alias MintInfo
	return json.Marshal(&struct {
		Alias
		Amount    string `json:"amount"`
		Mints     string `json:"mints"`
		Cap       string `json:"cap"`
		Remaining string `json:"remaining"`
	}{
		Alias:     Alias(s),
		Mints:     s.Mints.String(),
		Cap:       s.Cap.String(),
		Remaining: s.Remaining.String(),
	})
}

func (s *MintInfo) UnmarshalJSON(data []byte) error {
	type Alias MintInfo
	aux := &struct {
		*Alias
		Amount    string `json:"amount"`
		Mints     string `json:"mints"`
		Cap       string `json:"cap"`
		Remaining string `json:"remaining"`
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	s.Mints, err = uint128.FromString(aux.Mints)
	if err != nil {
		return err
	}
	s.Cap, err = uint128.FromString(aux.Cap)
	if err != nil {
		return err
	}
	s.Remaining, err = uint128.FromString(aux.Remaining)
	if err != nil {
		return err
	}
	return nil
}

type RuneInfo struct {
	Name              string
	Number            uint64
	Timestamp         uint64
	Id                string
	MintInfo          *MintInfo
	Supply            uint128.Uint128
	MaxSupply         uint128.Uint128
	Premine           uint128.Uint128
	PreminePercentage float64
	Burned            uint128.Uint128
	Divisibility      uint8
	Symbol            string
	Turbo             bool
	Etching           string
	Parent            string
}

func (s RuneInfo) MarshalJSON() ([]byte, error) {
	type Alias RuneInfo
	return json.Marshal(&struct {
		Alias
		Supply    string `json:"supply"`
		MaxSupply string `json:"maxSupply"`
		Premine   string `json:"premine"`
		Burned    string `json:"burned"`
	}{
		Alias:     Alias(s),
		Supply:    s.Supply.String(),
		MaxSupply: s.MaxSupply.String(),
		Premine:   s.Premine.String(),
		Burned:    s.Burned.String(),
	})
}

func (s *RuneInfo) UnmarshalJSON(data []byte) error {
	type Alias RuneInfo
	aux := &struct {
		*Alias
		Supply    string `json:"supply"`
		MaxSupply string `json:"maxSupply"`
		Premine   string `json:"premine"`
		Burned    string `json:"burned"`
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	s.Supply, err = uint128.FromString(aux.Supply)
	if err != nil {
		return err
	}
	s.MaxSupply, err = uint128.FromString(aux.MaxSupply)
	if err != nil {
		return err
	}
	s.Premine, err = uint128.FromString(aux.Premine)
	if err != nil {
		return err
	}
	s.Burned, err = uint128.FromString(aux.Burned)
	if err != nil {
		return err
	}
	return nil
}

func (s *RuneInfo) BlockHeight() int {
	runeId, err := runestone.RuneIdFromHex(s.Id)
	if err != nil {
		return -1
	}
	return int(runeId.Block)
}

type AddressBalance struct {
	AddressId uint64
	Address   string
	Balance   uint128.Uint128
	Pile      *runestone.Pile
}

func (s AddressBalance) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		AddressId uint64 `json:"addressid"`
		Balance   string `json:"balance"`
	}{
		AddressId: s.AddressId,
		Balance:   s.Balance.String(),
	})
}

func (s *AddressBalance) UnmarshalJSON(data []byte) error {
	aux := struct {
		AddressId uint64 `json:"addressid"`
		Balance   string `json:"balance"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.AddressId = aux.AddressId
	var err error
	s.Balance, err = uint128.FromString(aux.Balance)
	return err
}

type UtxoBalance struct {
	Utxo     string
	Outpoint *runestone.OutPoint
	Balance  uint128.Uint128
}

func (s UtxoBalance) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Utxo    string `json:"utxo"`
		Balance string `json:"balance"`
	}{
		Utxo:    s.Utxo,
		Balance: s.Balance.String(),
	})
}

func (s *UtxoBalance) UnmarshalJSON(data []byte) error {
	aux := struct {
		Utxo    string `json:"utxo"`
		Balance string `json:"balance"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.Utxo = aux.Utxo
	var err error
	s.Balance, err = uint128.FromString(aux.Balance)
	return err
}

type UtxoBalances struct {
	Total    uint128.Uint128
	Balances []*UtxoBalance
}

func (s UtxoBalances) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Total    string         `json:"total"`
		Balances []*UtxoBalance `json:"balances"`
	}{
		Total:    s.Total.String(),
		Balances: s.Balances,
	})
}

func (s *UtxoBalances) UnmarshalJSON(data []byte) error {
	aux := struct {
		Total    string         `json:"total"`
		Balances []*UtxoBalance `json:"balances"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.Balances = aux.Balances
	var err error
	s.Total, err = uint128.FromString(aux.Total)
	return err
}

type AddressAsset struct {
	Rune         string
	Balance      uint128.Uint128
	Divisibility uint8
	Symbol       rune
}

func (s AddressAsset) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Rune    string `json:"rune"`
		Balance string `json:"balance"`
	}{
		Rune:    s.Rune,
		Balance: s.Balance.String(),
	})
}

func (s *AddressAsset) UnmarshalJSON(data []byte) error {
	aux := struct {
		Rune    string `json:"rune"`
		Balance string `json:"balance"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	_, err = runestone.SpacedRuneFromString(aux.Rune)
	if err != nil {
		return err
	}
	s.Rune = aux.Rune
	s.Balance, err = uint128.FromString(aux.Balance)
	return err
}

type UtxoAsset struct {
	Rune         string
	Balance      uint128.Uint128
	Divisibility uint8
	Symbol       rune
}

type MintHistory struct {
	Utxo      string
	Amount    uint128.Uint128
	AddressId uint64
	Height    uint64
	Number    uint64
}

type Edict struct {
	RuneId string
	Amount uint128.Uint128
	Output uint32
}
