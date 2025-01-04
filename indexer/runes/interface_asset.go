package runes

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

/*
*
desc: 根据runeid获取所有持有者地址和持有数量 (新增数据表)
数据: key = rab-%runeid.string()-%address% value = nil
实现:
1 通过rune得到所有address
2 通过address拿到所有的utxo(RpcIndexer.GetUTXOs2(address))
3 根据utxo获取所有资产和持有数量 get_rune_balances_for_output(utxo)
4 对于相同的资产需要进行合并和汇总同一rune数量
*/
func (s *Indexer) GetAllAddressBalances(runeId string, start, limit uint64) ([]*AddressBalance, uint64) {
	rid, err := runestone.RuneIdFromString(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllAddressBalances-> runestone.SpacedRuneFromString(%s) err:%v", runeId, err.Error())
		return nil, 0
	}

	addresses := s.runeIdToAddressTbl.GetAddresses(rid)
	if len(addresses) == 0 {
		return nil, 0
	}

	r := s.idToEntryTbl.Get(rid)
	if r == nil {
		common.Log.Errorf("RuneIndexer.GetAllAddressBalances-> idToEntryTbl.Get(%s) rune not found, runeId: %s", rid.String(), runeId)
		return nil, 0
	}

	type AddressLotMap map[runestone.Address]*runestone.Lot
	addressLotMap := make(AddressLotMap)
	for _, address := range addresses {
		utxos := s.RpcService.GetUTXOs2(string(address))
		for _, utxo := range utxos {
			outpoint := &runestone.OutPoint{}
			utxoInfo, err := s.RpcService.GetUtxoInfo(utxo)
			if err != nil {
				common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> GetUtxoInfo(%s) err:%v", utxo, err)
			}
			utxoId := utxoInfo.UtxoId
			outpoint.FromUtxo(utxo, utxoId)
			blances := s.outpointToRuneBalancesTbl.Get(outpoint)
			for _, balance := range blances {
				if balance.RuneId.Block != rid.Block || balance.RuneId.Tx != rid.Tx {
					continue
				}
				if addressLotMap[address] == nil {
					addressLotMap[address] = runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
				}
				addressLotMap[address].AddAssign(&balance.Lot)
			}
		}
	}

	total := uint64(len(addressLotMap))
	ret := make([]*AddressBalance, total)
	var i = 0
	for address, lot := range addressLotMap {
		pile := r.Pile(*lot.Value)
		addressLot := &AddressBalance{
			Address: string(address),
			Balance: *lot.Value,
			Pile:    &pile,
		}
		ret[i] = addressLot
		i++
	}

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
desc: 根据runeId获取所有带有该rune的utxo和该utxo中的资产数量 (新增数据表)
数据: key = rub-%runeid.string()%-%utxo% value = nil
实现:
1 通过rune得到所有utxo
2 根据utxo获取资产和持有数量 get_rune_balances_for_output(utxo)
*/
func (s *Indexer) GetAllUtxoBalances(runeId string, start, limit uint64) (*UtxoBalances, uint64) {
	rid, err := runestone.RuneIdFromString(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllUtxoBalances-> runestone.SpacedRuneFromString(%s) err:%s", runeId, err.Error())
		return nil, 0
	}
	outpoints := s.runeIdToOutpointTbl.GetOutpoints(rid)
	if len(outpoints) == 0 {
		return nil, 0
	}

	type OutpointLotsMap map[runestone.OutPoint]*runestone.Lot
	outpointLotsMap := make(OutpointLotsMap)
	totalAmount := runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
	for _, outpoint := range outpoints {
		blances := s.outpointToRuneBalancesTbl.Get(outpoint)
		for _, balance := range blances {
			if balance.RuneId.Block != rid.Block || balance.RuneId.Tx != rid.Tx {
				continue
			}
			if outpointLotsMap[*outpoint] == nil {
				outpointLotsMap[*outpoint] = runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
			}
			outpointLotsMap[*outpoint].AddAssign(&balance.Lot)
			totalAmount.AddAssign(&balance.Lot)
		}

	}

	total := uint64(len(outpointLotsMap))
	ret := &UtxoBalances{
		Total:    *totalAmount.Value,
		Balances: make([]*UtxoBalance, total),
	}
	var i = 0
	for outpoint, lot := range outpointLotsMap {
		addressLot := &UtxoBalance{
			Utxo:    outpoint.String(),
			Balance: *lot.Value,
		}
		ret.Balances[i] = addressLot
		i++
	}

	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	ret.Balances = ret.Balances[start:end]
	return ret, total
}

/*
*
desc: 根据地址获取该地址所有ticker和持有的数量
*/
func (s *Indexer) GetAddressAssets(addressId uint64) []*AddressAsset {
	address, err := s.RpcService.GetAddressByID(addressId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAddressAssets-> GetAddressByID(%d) err:%v", addressId, err)
	}
	utxos := s.RpcService.GetUTXOs2(address)
	if len(utxos) == 0 {
		return nil
	}

	type SpaceRuneLotMap map[runestone.SpacedRune]*runestone.Lot
	spaceRuneLotMap := make(SpaceRuneLotMap)
	for _, utxo := range utxos {
		outpoint := &runestone.OutPoint{}
		utxoInfo, err := s.RpcService.GetUtxoInfo(utxo)
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> GetUtxoInfo(%s) err:%v", utxo, err)
		}
		utxoId := utxoInfo.UtxoId
		outpoint.FromUtxo(utxo, utxoId)
		balances := s.outpointToRuneBalancesTbl.Get(outpoint)
		for _, balance := range balances {
			runeEntry := s.idToEntryTbl.Get(&balance.RuneId)
			if spaceRuneLotMap[runeEntry.SpacedRune] == nil {
				spaceRuneLotMap[runeEntry.SpacedRune] = runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
			}
			spaceRuneLotMap[runeEntry.SpacedRune].AddAssign(&balance.Lot)
		}
	}

	total := uint64(len(spaceRuneLotMap))
	ret := make([]*AddressAsset, total)
	var i = 0
	for spacedRune, lot := range spaceRuneLotMap {
		addressLot := &AddressAsset{
			Rune:    spacedRune.String(),
			Balance: *lot.Value,
		}
		ret[i] = addressLot
		i++
	}
	return ret
}

/*
*
desc: 根据utxo获取ticker名字和资产数量
*/
func (s *Indexer) GetUtxoAssets(utxoId uint64) []*UtxoAsset {
	outpoint := &runestone.OutPoint{}
	utxo, err := s.RpcService.GetUtxoByID(utxoId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetUtxoAssets-> GetUtxoByID(%d) err:%v", utxoId, err)
	}
	outpoint.FromUtxo(utxo, utxoId)
	balances := s.outpointToRuneBalancesTbl.Get(outpoint)
	ret := make([]*UtxoAsset, len(balances))
	for i, balance := range balances {
		runeEntry := s.idToEntryTbl.Get(&balance.RuneId)
		ret[i] = &UtxoAsset{
			Rune:    runeEntry.SpacedRune.String(),
			Balance: *balance.Lot.Value,
		}
	}
	return ret
}

/*
*
desc: 判断utxo中是否有runes资产
实现: balances = get_rune_balances_for_output(utxo); return len(balances) > 0
*/
func (s *Indexer) IsExistAsset(utxoId uint64) bool {
	outpoint := &runestone.OutPoint{}
	utxo, err := s.RpcService.GetUtxoByID(utxoId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetUtxoAssets-> GetUtxoByID(%d) err:%v", utxoId, err)
	}
	outpoint.FromUtxo(utxo, utxoId)
	balances := s.outpointToRuneBalancesTbl.GetFromDB(outpoint)
	total := len(balances)
	return total > 0
}
