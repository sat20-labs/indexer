package runes

import (
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
)

type MintHistoryInfo struct {
	LastTimestamp int64
	MintHistory   []*MintHistory
}

const mintHistoryCacheDuration = 6 * time.Minute

var (
	runeMintHistoryCache cmap.ConcurrentMap[string, *MintHistoryInfo]
)

func init() {
	runeMintHistoryCache = cmap.New[*MintHistoryInfo]()
}

/*
*
desc: 根据runeid获取铸造历史 (新增数据表)
数据: key = rm-%runeid.string()%-%txid% value = nil
实现: 通过runeid得到所有mint的txid(一个txid即一个铸造历史)
*/
func (s *Indexer) GetAllMintHistory(runeId string) []*MintHistory {
	if mintHistoryInfo, exist := runeMintHistoryCache.Get(runeId); exist {
		if time.Since(time.Unix(mintHistoryInfo.LastTimestamp, 0)) < mintHistoryCacheDuration {
			return mintHistoryInfo.MintHistory
		}
	}

	id, err := runestone.RuneIdFromString(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllMintHistory-> runestone.RuneIdFromString(%s) err:%s", runeId, err.Error())
		return nil
	}
	mintHistorys, err := s.runeIdToMintHistoryTbl.GetList(id)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAllMintHistory-> runeIdToMintHistoryTbl.GetList(%s) err:%v", id.Hex(), err)
	}
	if len(mintHistorys) == 0 {
		return nil
	}

	r := s.idToEntryTbl.Get(id)
	if r == nil {
		common.Log.Errorf("RuneIndexer.GetAllMintHistory-> idToEntryTbl.Get(%s) rune not found, ticker: %s", id.Hex(), runeId)
		return nil
	}

	ret := make([]*MintHistory, len(mintHistorys))
	for i, history := range mintHistorys {
		ret[i] = &MintHistory{
			UtxoId:    history.UtxoId,
			Amount:    *r.Terms.Amount,
			AddressId: history.AddressId,
			Height:    r.RuneId.Block,
			Number:    r.Number,
		}
	}

	mintHistoryInfo := &MintHistoryInfo{
		LastTimestamp: time.Now().Unix(),
		MintHistory:   ret,
	}
	runeMintHistoryCache.Set(runeId, mintHistoryInfo)

	return ret
}

func (s *Indexer) GetMintHistory(runeId string, start, limit uint64) ([]*MintHistory, uint64) {
	ret := s.GetAllMintHistory(runeId)
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
	mintHistorys, err := s.addressRuneIdToMintHistoryTbl.GetList(addressId, id)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAddressMintHistory-> addressRuneIdToMintHistoryTbl.GetList(%d, %s) err:%v", addressId, id.Hex(), err)
	}
	if len(mintHistorys) == 0 {
		return nil, 0
	}

	runeEntry := s.idToEntryTbl.Get(id)
	if runeEntry == nil {
		common.Log.Errorf("RuneIndexer.GetAddressMintHistory-> idToEntryTbl.Get(%s) rune not found, runeIdStr: %s", id.Hex(), runeId)
		return nil, 0
	}

	ret := make([]*MintHistory, len(mintHistorys))
	for i, mintHistory := range mintHistorys {
		ret[i] = &MintHistory{
			UtxoId:    mintHistory.OutPoint.UtxoId,
			Amount:    *runeEntry.Terms.Amount,
			AddressId: mintHistory.AddressId,
			Height:    runeEntry.RuneId.Block,
			Number:    runeEntry.Number,
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
