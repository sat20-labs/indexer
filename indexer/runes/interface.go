package runes

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
)

func (s *Indexer) GetRuneInfo(ticker string) *runestone.RuneEntry {
	r, err := runestone.RuneFromString(ticker)
	if err != nil {
		common.Log.Debugf("RuneIndexer.GetRuneInfo-> runestone.RuneFromString(%s) err:%s", ticker, err.Error())
		return nil
	}
	runeId := s.runeToIdTbl.GetFromDB(r)
	if runeId == nil {
		common.Log.Infof("RuneIndexer.GetRuneInfo-> runeToIdTbl.GetFromDB(%s) rune not found, ticker: %s", r.String(), ticker)
		return nil
	}
	runeEntry := s.idToEntryTbl.GetFromDB(runeId)
	return runeEntry
}

func (s *Indexer) GetHolders(ticker string, start, limit int) ([]*runestone.RuneHolder, int) {
	r, err := runestone.RuneFromString(ticker)
	if err != nil {
		common.Log.Debugf("RuneIndexer.GetHolders-> runestone.RuneFromString(%s) err:%v", ticker, err.Error())
		return nil, 0
	}
	holders := s.runeHolderTbl.GetFromDB(r)
	return holders, 0
}

func (s *Indexer) GetMintHistory(ticker string, start, limit int) (runestone.RuneMintHistorys, int) {
	r, err := runestone.RuneFromString(ticker)
	if err != nil {
		common.Log.Debugf("RuneIndexer.GetMintHistory-> runestone.RuneFromString(%s) err:%v", ticker, err.Error())
		return nil, 0
	}
	mintHistorys := s.runeMintHistorysTbl.GetFromDB(r)
	if mintHistorys == nil {
		return nil, 0
	}
	end := len(mintHistorys)
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	return mintHistorys[start:end], end
}

func (s *Indexer) GetAddressMintHistory(address runestone.Address, ticker string, start, limit int) (runestone.RuneMintHistorys, int) {
	r, err := runestone.RuneFromString(ticker)
	if err != nil {
		common.Log.Debugf("RuneIndexer.GetAddressMintHistory-> runestone.RuneFromString(%s) err:%v", ticker, err.Error())
		return nil, 0
	}
	ledger := s.runeLedgerTbl.GetFromDB(address)
	if ledger == nil {
		common.Log.Infof("RuneIndexer.GetAddressMintHistory-> runeLedgerTbl.GetFromDB(%s) rune not found, ticker: %s", address, ticker)
		return nil, 0
	}

	mintHistorys := make(runestone.RuneMintHistorys, len(ledger.Assets[*r].Mints))
	mints := ledger.Assets[*r].Mints
	for i, mint := range mints {
		mintHistory := &runestone.RuneMintHistory{
			Address: address,
			Rune:    *r,
			Utxo:    mint.String(),
		}
		mintHistorys[i] = mintHistory
	}

	total := len(mintHistorys)
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}

	return mintHistorys[start:end], total
}

func (s *Indexer) GetMintAmount(ticker string) (mint uint64, supply uint64) {
	runeEntry := s.GetRuneInfo(ticker)
	if runeEntry == nil {
		return 0, 0
	}
	mint = runeEntry.Mints.Big().Uint64()
	supply = runeEntry.Supply().Big().Uint64()
	return mint, supply
}
