package runes

import (
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

type RuneMint struct {
	Start     uint64 //
	End       uint64
	Amount    uint128.Uint128
	Mints     uint128.Uint128
	Cap       uint128.Uint128 // runestone.RuneEntry.Terms.Cap
	Remaining uint128.Uint128
	Mintable  bool
	Progress  float32
}

type RuneInfo struct {
	Name               string // SpacedRune   runestone.SpacedRune
	Number             uint64 // RuneEntry.Number
	Timestamp          uint64 // RuneEntry.Timestamp
	Id                 string // runestone.RuneId.Block:runestone.RuneId.Tx
	EtchingBlock       uint64 // runestone.RuneId.Block
	EtchingTransaction uint32 // runestone.RuneId.Tx
	Supply             uint128.Uint128
	Premine            uint128.Uint128 // runestone.Etching.Premine
	PreminePercentage  float32
	Burned             uint128.Uint128 // runestone.RuneEntry.Burned
	Divisibility       uint8           // runestone.Etching.Divisibility
	Symbol             string          // runestone.Etching.Symbol
	Turbo              bool            // runestone.RuneEntry.Turbo
	Etching            string          // Txid, runestone.RuneEntry.Etching
	Parent             runestone.InscriptionId

	Terms *runestone.Terms
}
