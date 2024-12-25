package runestone

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/pb"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"lukechampine.com/uint128"
)

type RuneAsset struct {
	Balance   uint128.Uint128
	IsEtching bool //Indicates if this address is etching
	Mints     []*OutPoint
	Transfers []*Edict
}

type RuneLedger struct {
	Assets map[Rune]*RuneAsset
}

type Address string
type RuneLedgers map[Address]*RuneLedger

func (s *RuneLedger) ToPb() (ret *pb.RuneLedger) {
	ret = &pb.RuneLedger{
		Assets: make(map[string]*pb.RuneAsset, len(s.Assets)),
	}
	for r, asset := range s.Assets {
		key := r.String()
		runeAsset := &pb.RuneAsset{
			Balance:   &pb.Uint128{Lo: asset.Balance.Lo, Hi: asset.Balance.Hi},
			IsEtching: asset.IsEtching,
			Mints:     make([]*pb.OutPoint, len(asset.Mints)),
			Transfers: make([]*pb.Edict, len(asset.Transfers)),
		}

		for i, mint := range asset.Mints {
			outpoint := &pb.OutPoint{
				Txid: mint.Txid,
				Vout: mint.Vout,
			}
			runeAsset.Mints[i] = outpoint
		}

		for i, transfer := range asset.Transfers {
			edict := &pb.Edict{
				Id:     &pb.RuneId{Block: transfer.ID.Block, Tx: transfer.ID.Tx},
				Amount: &pb.Uint128{Lo: transfer.Amount.Lo, Hi: transfer.Amount.Hi},
				Output: transfer.Output,
			}
			runeAsset.Transfers[i] = edict
		}
		ret.Assets[key] = runeAsset
	}
	return ret
}

func (s *RuneLedger) FromPb(pbVal *pb.RuneLedger) {
	s.Assets = make(map[Rune]*RuneAsset, len(pbVal.Assets))
	for k, v := range pbVal.Assets {
		prune, err := RuneFromString(k)
		if err != nil {
			common.Log.Panicf("RuneLedger->FromPb: err: %v", err.Error())
		}
		r := *prune
		s.Assets[r] = &RuneAsset{
			Balance:   uint128.Uint128{Lo: v.Balance.Lo, Hi: v.Balance.Hi},
			IsEtching: v.IsEtching,
			Mints:     make([]*OutPoint, len(v.Mints)),
			Transfers: make([]*Edict, len(v.Transfers)),
		}
		for i, mint := range v.Mints {
			s.Assets[r].Mints[i] = &OutPoint{Txid: mint.Txid, Vout: mint.Vout}
		}
		for i, transfer := range v.Transfers {
			s.Assets[r].Transfers[i] = &Edict{
				ID:     RuneId{Block: transfer.Id.Block, Tx: transfer.Id.Tx},
				Amount: uint128.Uint128{Lo: transfer.Amount.Lo, Hi: transfer.Amount.Hi},
				Output: transfer.Output,
			}
		}
	}
}

type RuneLedgerTable struct {
	Table[pb.RuneLedger]
}

func NewRuneLedgerTable(store *store.Store[pb.RuneLedger]) *RuneLedgerTable {
	return &RuneLedgerTable{Table: Table[pb.RuneLedger]{store: store}}
}

func (s *RuneLedgerTable) Get(key Address) (ret *RuneLedger) {
	tblKey := []byte(store.RUNE_LEDGER + key)
	pbVal := s.store.Get(tblKey)
	if pbVal != nil {
		ret = &RuneLedger{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneLedgerTable) GetNoTransaction(key Address) (ret *RuneLedger) {
	tblKey := []byte(store.RUNE_LEDGER + key)
	pbVal := s.store.GetNoTransaction(tblKey)
	if pbVal != nil {
		ret = &RuneLedger{}
		ret.FromPb(pbVal)
	}
	return
}

func (s *RuneLedgerTable) Insert(key Address, value *RuneLedger) (ret *RuneLedger) {
	tblKey := []byte(store.RUNE_LEDGER + key)
	pbVal := s.store.Insert(tblKey, value.ToPb())
	if pbVal != nil {
		ret = &RuneLedger{}
		ret.FromPb(pbVal)
	}
	return
}
