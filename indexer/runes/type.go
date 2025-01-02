package runes

import (
	"encoding/json"

	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

type MintInfo struct {
	Start     string `json:"start"`
	End       string `json:"end"`
	Amount    string `json:"amount"`
	Mints     uint128.Uint128
	Cap       uint128.Uint128
	Remaining uint128.Uint128
	Mintable  bool   `json:"mintable"`
	Progress  string `json:"progress"`
}

func (s MintInfo) MarshalJSON() ([]byte, error) {
	type Alias MintInfo
	return json.Marshal(&struct {
		Alias
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
	Name               string
	Number             uint64
	Timestamp          uint64
	Id                 string
	EtchingBlock       uint64
	EtchingTransaction uint32
	MintInfo           *MintInfo
	Supply             uint128.Uint128
	Premine            string
	PreminePercentage  string
	Burned             uint128.Uint128
	Divisibility       uint8
	Symbol             string
	Turbo              bool
	Etching            string
	Parent             string
}

func (s RuneInfo) MarshalJSON() ([]byte, error) {
	type Alias RuneInfo
	return json.Marshal(&struct {
		Alias
		Supply string `json:"supply"`
		Burned string `json:"burned"`
	}{
		Alias:  Alias(s),
		Supply: s.Supply.String(),
		Burned: s.Burned.String(),
	})
}

func (s *RuneInfo) UnmarshalJSON(data []byte) error {
	type Alias RuneInfo
	aux := &struct {
		*Alias
		Supply string `json:"supply"`
		Burned string `json:"burned"`
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
	s.Burned, err = uint128.FromString(aux.Burned)
	if err != nil {
		return err
	}
	return nil
}

type AddressBalance struct {
	Address string
	Balance uint128.Uint128
}

func (s AddressBalance) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Address string `json:"address"`
		Balance string `json:"balance"`
	}{
		Address: s.Address,
		Balance: s.Balance.String(),
	})
}

func (s *AddressBalance) UnmarshalJSON(data []byte) error {
	aux := struct {
		Address string `json:"address"`
		Balance string `json:"balance"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.Address = aux.Address
	var err error
	s.Balance, err = uint128.FromString(aux.Balance)
	return err
}

type UtxoBalance struct {
	Utxo    string
	Balance uint128.Uint128
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
	SpacedRune *runestone.SpacedRune
	Balance    uint128.Uint128
}

func (s AddressAsset) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		SpacedRune string `json:"spacedrune"`
		Balance    string `json:"balance"`
	}{
		SpacedRune: s.SpacedRune.String(),
		Balance:    s.Balance.String(),
	})
}

func (s *AddressAsset) UnmarshalJSON(data []byte) error {
	aux := struct {
		SpacedRune string `json:"spacedrune"`
		Balance    string `json:"balance"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	s.SpacedRune, err = runestone.SpacedRuneFromString(aux.SpacedRune)
	if err != nil {
		return err
	}
	s.Balance, err = uint128.FromString(aux.Balance)
	return err
}

type UtxoAsset struct {
	SpacedRune *runestone.SpacedRune
	Balance    uint128.Uint128
}

type Edict struct {
	RuneName string
	Amount   uint128.Uint128
	Output   uint32
}
