package runes

import (
	"lukechampine.com/uint128"
)

type MintInfo struct {
	Start     string
	End       string
	Amount    string
	Mints     uint128.Uint128
	Cap       uint128.Uint128
	Remaining uint128.Uint128
	Mintable  bool
	Progress  string
}

type RuneInfo struct {
	Name               string // SpacedRune   runestone.SpacedRune
	Number             uint64 // RuneEntry.Number
	Timestamp          uint64 // RuneEntry.Timestamp
	Id                 string // runestone.RuneId.Block:runestone.RuneId.Tx
	EtchingBlock       uint64 // runestone.RuneId.Block
	EtchingTransaction uint32 // runestone.RuneId.Tx
	MintInfo           *MintInfo
	Supply             uint128.Uint128
	Premine            string // runestone.Etching.Premine
	PreminePercentage  string
	Burned             uint128.Uint128 // runestone.RuneEntry.Burned
	Divisibility       uint8           // runestone.Etching.Divisibility
	Symbol             string          // runestone.Etching.Symbol
	Turbo              bool            // runestone.RuneEntry.Turbo
	Etching            string          // Txid, runestone.RuneEntry.Etching
	Parent             string
}
