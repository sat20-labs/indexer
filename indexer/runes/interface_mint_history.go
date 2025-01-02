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
func (s *Indexer) GetMintHistory(ticker string, start, limit uint64) ([]runestone.Txid, uint64) {
	spaceRune, err := runestone.SpacedRuneFromString(ticker)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetMintHistory-> runestone.SpacedRuneFromString(%s) err:%s", ticker, err.Error())
		return nil, 0
	}
	runeId := s.runeToIdTbl.GetFromDB(&spaceRune.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetMintHistory-> runeToIdTbl.GetFromDB(%s) rune not found, ticker: %s", spaceRune.String(), ticker)
		return nil, 0
	}
	ret := s.runeIdToMintHistoryTbl.GetTxidsFromDB(runeId)
	if len(ret) == 0 {
		return nil, 0
	}

	total := uint64(len(ret))
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	return ret[start:end], end
}

/*
*
desc: 根据地址获取指定ticker的铸造历史 (新增数据表)
数据: key = arm-%address%-%runeid.string()%-%utxo% value = nil
实现: 通过address和runeid得到所有utxo(一个txid(1/n个utxo)即一个铸造历史)
*/
func (s *Indexer) GetAddressMintHistory(ticker, address string, start, limit uint64) ([]runestone.Txid, uint64) {
	spaceRune, err := runestone.SpacedRuneFromString(ticker)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAddressMintHistory-> runestone.SpacedRuneFromString(%s) err:%s", ticker, err.Error())
		return nil, 0
	}
	runeId := s.runeToIdTbl.GetFromDB(&spaceRune.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetAddressMintHistory-> runeToIdTbl.GetFromDB(%s) rune not found, ticker: %s", spaceRune.String(), ticker)
		return nil, 0
	}
	ret := s.addressRuneIdToMintHistoryTbl.GetTxidsFromDB(runestone.Address(address), runeId)
	if len(ret) == 0 {
		return nil, 0
	}

	total := uint64(len(ret))
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	return ret[start:end], end
}
