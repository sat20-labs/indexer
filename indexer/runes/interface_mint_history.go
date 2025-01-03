package runes

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
)

/*
*
desc: 根据ticker名字获取铸造历史 (新增数据表)
数据: key = rm-%runeid.string()%-%txid% value = nil
实现: 通过runeid得到所有mint的txid(一个txid即一个铸造历史)
*/
func (s *Indexer) GetMintHistory(ticker string, start, limit uint64) ([]*MintHistory, uint64) {
	spaceRune, err := runestone.SpacedRuneFromString(ticker)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetMintHistory-> runestone.SpacedRuneFromString(%s) err:%s", ticker, err.Error())
		return nil, 0
	}
	runeId := s.runeToIdTbl.Get(&spaceRune.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetMintHistory-> runeToIdTbl.Get(%s) rune not found, ticker: %s", spaceRune.String(), ticker)
		return nil, 0
	}
	utxos := s.runeIdToMintHistoryTbl.GetUtxos(runeId)
	if len(utxos) == 0 {
		return nil, 0
	}

	runeEntry := s.idToEntryTbl.Get(runeId)
	if runeEntry == nil {
		common.Log.Errorf("RuneIndexer.GetMintHistory-> idToEntryTbl.Get(%s) rune not found, ticker: %s", runeId.String(), ticker)
		return nil, 0
	}

	ret := make([]*MintHistory, len(utxos))
	for i, utxo := range utxos {
		ret[i] = &MintHistory{
			Utxo:   string(utxo),
			Amount: *runeEntry.Terms.Amount,
		}
	}

	total := uint64(len(ret))
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	return ret[start:end], total
}

/*
*
desc: 根据地址获取指定ticker的铸造历史 (新增数据表)
数据: key = arm-%address%-%runeid.string()%-%utxo% value = nil
实现: 通过address和runeid得到所有utxo(一个txid(1/n个utxo)即一个铸造历史)
*/
func (s *Indexer) GetAddressMintHistory(ticker, address string, start, limit uint64) ([]*MintHistory, uint64) {
	spaceRune, err := runestone.SpacedRuneFromString(ticker)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAddressMintHistory-> runestone.SpacedRuneFromString(%s) err:%s", ticker, err.Error())
		return nil, 0
	}
	runeId := s.runeToIdTbl.Get(&spaceRune.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetAddressMintHistory-> runeToIdTbl.Get(%s) rune not found, ticker: %s", spaceRune.String(), ticker)
		return nil, 0
	}
	utxos := s.addressRuneIdToMintHistoryTbl.GetUtxos(runestone.Address(address), runeId)
	if len(utxos) == 0 {
		return nil, 0
	}

	runeEntry := s.idToEntryTbl.Get(runeId)
	if runeEntry == nil {
		common.Log.Errorf("RuneIndexer.GetAddressMintHistory-> idToEntryTbl.Get(%s) rune not found, ticker: %s", runeId.String(), ticker)
		return nil, 0
	}

	ret := make([]*MintHistory, len(utxos))
	for i, utxo := range utxos {
		ret[i] = &MintHistory{
			Utxo:   string(utxo),
			Amount: *runeEntry.Terms.Amount,
		}
	}

	total := uint64(len(ret))
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	return ret[start:end], total
}
