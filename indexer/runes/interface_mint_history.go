package runes

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
)

/*
*
desc: 根据runeid获取铸造历史 (新增数据表)
数据: key = rm-%runeid.string()%-%txid% value = nil
实现: 通过runeid得到所有mint的txid(一个txid即一个铸造历史)
*/
func (s *Indexer) GetMintHistory(runeId string, start, limit uint64) ([]*MintHistory, uint64) {
	id, err := runestone.RuneIdFromString(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetMintHistory-> runestone.SpacedRuneFromString(%s) err:%s", runeId, err.Error())
		return nil, 0
	}
	utxos, err := s.runeIdToMintHistoryTbl.GetUtxos(id)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetMintHistory-> runeIdToMintHistoryTbl.GetUtxos(%s) err:%v", id.String(), err)
	}
	if len(utxos) == 0 {
		return nil, 0
	}

	runeEntry := s.idToEntryTbl.Get(id)
	if runeEntry == nil {
		common.Log.Errorf("RuneIndexer.GetMintHistory-> idToEntryTbl.Get(%s) rune not found, ticker: %s", id.String(), runeId)
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
desc: 根据地址获取指定nuneid的铸造历史 (新增数据表)
数据: key = arm-%address%-%runeid.string()%-%utxo% value = nil
实现: 通过address和runeid得到所有utxo(一个txid(1/n个utxo)即一个铸造历史)
*/
func (s *Indexer) GetAddressMintHistory(runeId string, addressId uint64, start, limit uint64) ([]*MintHistory, uint64) {
	id, err := runestone.RuneIdFromString(runeId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAddressMintHistory-> runestone.SpacedRuneFromString(%s) err:%s", runeId, err.Error())
	}
	address, err := s.RpcService.GetAddressByID(addressId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAddressMintHistory-> GetAddressByID(%d) err:%v", addressId, err)
	}
	utxos, err := s.addressRuneIdToMintHistoryTbl.GetUtxos(runestone.Address(address), id)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAddressMintHistory-> addressRuneIdToMintHistoryTbl.GetUtxos(%s, %s) err:%v", address, id.String(), err)
	}
	if len(utxos) == 0 {
		return nil, 0
	}

	runeEntry := s.idToEntryTbl.Get(id)
	if runeEntry == nil {
		common.Log.Errorf("RuneIndexer.GetAddressMintHistory-> idToEntryTbl.Get(%s) rune not found, runeIdStr: %s", id.String(), runeId)
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
