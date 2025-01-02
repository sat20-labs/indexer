package runes

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

/*
*
desc: 根据ticker名字获取所有持有者地址和持有数量 (新增数据表)
数据: key = rab-%runeid.string()-%address% value = nil
实现:
1 通过rune得到所有address
2 通过address拿到所有的utxo(RpcIndexer.GetUTXOs2(address))
3 根据utxo获取所有资产和持有数量 get_rune_balances_for_output(utxo)
4 对于相同的资产需要进行合并和汇总同一rune数量
*/
func (s *Indexer) GetAllAddressBalances(ticker string, start, limit uint64) ([]*AddressBalance, uint64) {
	runeSpace, err := runestone.SpacedRuneFromString(ticker)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllAddressBalances-> runestone.SpacedRuneFromString(%s) err:%v", ticker, err.Error())
		return nil, 0
	}
	runeId := s.runeToIdTbl.GetFromDB(&runeSpace.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetAllAddressBalances-> runeToIdTbl.GetFromDB(%s) rune not found, ticker: %s", runeSpace.String(), ticker)
		return nil, 0
	}

	addresses := s.runeIdToAddressTbl.GetAddressesFromDB(runeId)
	if len(addresses) == 0 {
		return nil, 0
	}

	type AddressLotMap map[runestone.Address]*runestone.Lot
	addressLotMap := make(AddressLotMap)
	for _, address := range addresses {
		utxos := s.rpcService.GetUTXOs2(string(address))
		for _, utxo := range utxos {
			outPoint := &runestone.OutPoint{}
			outPoint.FromString(utxo)
			blances := s.outpointToRuneBalancesTbl.GetFromDB(outPoint)
			for _, balance := range blances {
				if balance.RuneId.Block != runeId.Block || balance.RuneId.Tx != runeId.Tx {
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
	for address, lot := range addressLotMap {
		addressLot := &AddressBalance{
			Address: string(address),
			Balance: *lot.Value,
		}
		ret = append(ret, addressLot)
	}

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
desc: 根据ticker名字获取所有带有该ticker的utxo和该utxo中的资产数量 (新增数据表)
数据: key = rub-%runeid.string()%-%utxo% value = nil
实现:
1 通过rune得到所有utxo
2 根据utxo获取资产和持有数量 get_rune_balances_for_output(utxo)
*/
func (s *Indexer) GetAllUtxoBalances(ticker string, start, limit uint64) (*UtxoBalances, uint64) {
	spaceRune, err := runestone.SpacedRuneFromString(ticker)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllUtxoBalances-> runestone.SpacedRuneFromString(%s) err:%s", ticker, err.Error())
		return nil, 0
	}
	runeId := s.runeToIdTbl.GetFromDB(&spaceRune.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetAllUtxoBalances-> runeToIdTbl.GetFromDB(%s) rune not found, ticker: %s", spaceRune.String(), ticker)
		return nil, 0
	}
	outpoints := s.runeIdToOutpointTbl.GetOutpointsFromDB(runeId)
	if len(outpoints) == 0 {
		return nil, 0
	}

	type OutpointLotsMap map[runestone.OutPoint]*runestone.Lot
	outpointLotsMap := make(OutpointLotsMap)
	totalAmount := runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
	for _, outpoint := range outpoints {
		blances := s.outpointToRuneBalancesTbl.GetFromDB(outpoint)
		for _, balance := range blances {
			if balance.RuneId.Block != runeId.Block || balance.RuneId.Tx != runeId.Tx {
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
	for outpoint, lot := range outpointLotsMap {
		addressLot := &UtxoBalance{
			Utxo:    outpoint.String(),
			Balance: *lot.Value,
		}
		ret.Balances = append(ret.Balances, addressLot)
	}

	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	ret.Balances = ret.Balances[start:end]
	return ret, end
}

/*
*
desc: 根据地址获取该地址所有ticker和持有的数量
*/
func (s *Indexer) GetAddressAssets(address string, start, limit uint64) ([]*AddressAsset, uint64) {
	utxos := s.rpcService.GetUTXOs2(address)
	if len(utxos) == 0 {
		return nil, 0
	}

	type SpaceRuneLotMap map[runestone.SpacedRune]*runestone.Lot
	spaceRuneLotMap := make(SpaceRuneLotMap)
	for _, utxo := range utxos {
		outpoint := &runestone.OutPoint{}
		outpoint.FromString(utxo)
		balances := s.outpointToRuneBalancesTbl.GetFromDB(outpoint)
		for _, balance := range balances {
			runeEntry := s.idToEntryTbl.GetFromDB(&balance.RuneId)
			if spaceRuneLotMap[runeEntry.SpacedRune] == nil {
				spaceRuneLotMap[runeEntry.SpacedRune] = runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
			}
			spaceRuneLotMap[runeEntry.SpacedRune].AddAssign(&balance.Lot)
		}
	}

	total := uint64(len(spaceRuneLotMap))
	ret := make([]*AddressAsset, total)
	for spacedRune, lot := range spaceRuneLotMap {
		addressLot := &AddressAsset{
			SpacedRune: &spacedRune,
			Balance:    *lot.Value,
		}
		ret = append(ret, addressLot)
	}

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
desc: 根据utxo获取ticker名字和资产数量
*/
func (s *Indexer) GetUtxoAssets(utxo string, start, limit uint64) ([]*UtxoAsset, uint64) {
	outpoint := &runestone.OutPoint{}
	outpoint.FromString(utxo)
	balances := s.outpointToRuneBalancesTbl.GetFromDB(outpoint)
	ret := make([]*UtxoAsset, 0)
	for _, balance := range balances {
		runeEntry := s.idToEntryTbl.GetFromDB(&balance.RuneId)
		ret = append(ret, &UtxoAsset{
			SpacedRune: &runeEntry.SpacedRune,
			Balance:    *balance.Lot.Value,
		})
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
desc: 判断utxo中是否有runes资产
实现: balances = get_rune_balances_for_output(utxo); return len(balances) > 0
*/
func (s *Indexer) IsExistAsset(utxo string) bool {
	outpoint := &runestone.OutPoint{}
	outpoint.FromString(utxo)
	balances := s.outpointToRuneBalancesTbl.GetFromDB(outpoint)
	total := len(balances)
	return total > 0
}