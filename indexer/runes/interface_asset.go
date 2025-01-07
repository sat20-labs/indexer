package runes

import (
	"sort"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

// key: addressId, value: amount
func (s *Indexer) GetHoldersWithTick(runeId string) (ret map[uint64]*common.Decimal) {
	rid, err := runestone.RuneIdFromString(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetHoldersWithTick-> runestone.RuneIdFromString(%s) err:%v", runeId, err.Error())
		return nil
	}
	runeIdToAddresses, err := s.runeIdToAddressTbl.GetList(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetHoldersWithTick-> runeIdToAddressTbl.GetList(%s) err:%v", rid.Hex(), err.Error())
	}
	if len(runeIdToAddresses) == 0 {
		return nil
	}

	r := s.idToEntryTbl.Get(rid)
	if r == nil {
		common.Log.Errorf("RuneIndexer.GetHoldersWithTick-> idToEntryTbl.Get(%s) rune not found, runeId: %s", rid.Hex(), runeId)
		return nil
	}

	type AddressLot struct {
		Address string
		Amount  *runestone.Lot
	}
	type AddressIdToAddressLotMap map[uint64]*AddressLot
	addressIdToAddressLotMap := make(AddressIdToAddressLotMap)
	for _, address := range runeIdToAddresses {
		utxos := s.RpcService.GetUTXOs2(string(address.Address))
		for _, utxo := range utxos {
			utxoInfo, err := s.RpcService.GetUtxoInfo(utxo)
			if err != nil {
				common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> GetUtxoInfo(%s) err:%v", utxo, err)
			}
			utxoId := utxoInfo.UtxoId
			outpoint, err := runestone.OutPointFromUtxo(utxo, utxoId)
			if err != nil {
				common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> runestone.OutPointFromUtxo(%s, %d) err:%v", utxo, utxoId, err)
			}
			blances := s.outpointToBalancesTbl.Get(outpoint)
			for _, balance := range blances {
				if balance.RuneId.Block != rid.Block || balance.RuneId.Tx != rid.Tx {
					continue
				}

				if addressIdToAddressLotMap[address.AddressId] == nil {
					addressIdToAddressLotMap[address.AddressId] = &AddressLot{
						Address: string(address.Address),
						Amount:  runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0}),
					}
				}
				addressIdToAddressLotMap[address.AddressId].Amount.AddAssign(&balance.Lot)
			}
		}
	}

	total := uint64(len(addressIdToAddressLotMap))
	ret = make(map[uint64]*common.Decimal, total)
	var i = 0

	for addressId, addressLot := range addressIdToAddressLotMap {
		decimal := common.NewDecimalFromUint128(*addressLot.Amount.Value, int(r.Divisibility))
		ret[addressId] = decimal
		i++
	}
	return
}

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
	return nil, 0
	rid, err := runestone.RuneIdFromDec(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllAddressBalances-> runestone.SpacedRuneFromString(%s) err:%v", runeId, err.Error())
		return nil, 0
	}

	runeIdToAddresses, err := s.runeIdToAddressTbl.GetList(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> runeIdToAddressTbl.GetList(%s) err:%v", rid.Hex(), err.Error())
	}
	if len(runeIdToAddresses) == 0 {
		return nil, 0
	}

	r := s.idToEntryTbl.Get(rid)
	if r == nil {
		common.Log.Errorf("RuneIndexer.GetAllAddressBalances-> idToEntryTbl.Get(%s) rune not found, runeId: %s", rid.Hex(), runeId)
		return nil, 0
	}

	type AddressLot struct {
		Address string
		Amount  *runestone.Lot
	}
	type AddressIdToAddressLotMap map[uint64]*AddressLot
	addressIdToAddressLotMap := make(AddressIdToAddressLotMap)
	for _, address := range runeIdToAddresses {
		utxos := s.RpcService.GetUTXOs2(string(address.Address))
		for _, utxo := range utxos {
			utxoInfo, err := s.RpcService.GetUtxoInfo(utxo)
			if err != nil {
				common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> GetUtxoInfo(%s) err:%v", utxo, err)
			}
			utxoId := utxoInfo.UtxoId
			outpoint, err := runestone.OutPointFromUtxo(utxo, utxoId)
			if err != nil {
				common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> runestone.OutPointFromUtxo(%s, %d) err:%v", utxo, utxoId, err)
			}
			blances := s.outpointToBalancesTbl.Get(outpoint)
			for _, balance := range blances {
				if balance.RuneId.Block != rid.Block || balance.RuneId.Tx != rid.Tx {
					continue
				}

				if addressIdToAddressLotMap[address.AddressId] == nil {
					addressIdToAddressLotMap[address.AddressId] = &AddressLot{
						Address: string(address.Address),
						Amount:  runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0}),
					}
				}
				addressIdToAddressLotMap[address.AddressId].Amount.AddAssign(&balance.Lot)
			}
		}
	}

	total := uint64(len(addressIdToAddressLotMap))
	ret := make([]*AddressBalance, total)
	var i = 0
	for addressId, addressLot := range addressIdToAddressLotMap {
		pile := r.Pile(*addressLot.Amount.Value)
		addressLot := &AddressBalance{
			AddressId: addressId,
			Address:   addressLot.Address,
			Balance:   *addressLot.Amount.Value,
			Pile:      &pile,
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

	balances, err := s.runeIdOutpointToBalanceTbl.GetBalances(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAllUtxoBalances-> runeIdToOutpointTbl.GetOutpoints(%s) err:%s", rid.Hex(), err.Error())
	}

	if len(balances) == 0 {
		return nil, 0
	}

	sort.Slice(balances, func(i, j int) bool {
		return balances[i].OutPoint.UtxoId < balances[j].OutPoint.UtxoId
	})

	total := uint64(len(balances))

	ret := &UtxoBalances{
		Total:    uint128.Zero,
		Balances: make([]*UtxoBalance, len(balances)),
	}

	totalAmount := runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
	var i = 0
	for _, balance := range balances {
		totalAmount.AddAssign(balance.Balance)
		addressLot := &UtxoBalance{
			Utxo:     balance.OutPoint.String(),
			Outpoint: balance.OutPoint,
			Balance:  *balance.Balance.Value,
		}
		ret.Balances[i] = addressLot
		i++
	}
	ret.Total = *totalAmount.Value

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

func (s *Indexer) SlowGetAllUtxoBalances(runeId string, start, limit uint64) (*UtxoBalances, uint64) {
	rid, err := runestone.RuneIdFromString(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllUtxoBalances-> runestone.SpacedRuneFromString(%s) err:%s", runeId, err.Error())
		return nil, 0
	}
	outpoints, err := s.runeIdToOutpointTbl.GetOutpoints(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAllUtxoBalances-> runeIdToOutpointTbl.GetOutpoints(%s) err:%s", rid.Hex(), err.Error())
	}
	if len(outpoints) == 0 {
		return nil, 0
	}

	type OutpointLotsMap map[runestone.OutPoint]*runestone.Lot
	outpointLotsMap := make(OutpointLotsMap)
	totalAmount := runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
	for _, outpoint := range outpoints {
		balances := s.outpointToBalancesTbl.Get(outpoint)
		for _, balance := range balances {
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
			Utxo:     outpoint.String(),
			Outpoint: &outpoint,
			Balance:  *lot.Value,
		}
		ret.Balances[i] = addressLot
		i++
	}

	sort.Slice(ret.Balances, func(i, j int) bool {
		return ret.Balances[i].Outpoint.UtxoId < ret.Balances[j].Outpoint.UtxoId
	})

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

	type RuneBalance struct {
		Balance      *runestone.Lot
		Divisibility uint8
		Symbol       rune
	}
	type SpaceRuneLotMap map[runestone.SpacedRune]*RuneBalance
	spaceRuneLotMap := make(SpaceRuneLotMap)
	for _, utxo := range utxos {
		utxoInfo, err := s.RpcService.GetUtxoInfo(utxo)
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetAddressAssets-> GetUtxoInfo(%s) err:%v", utxo, err)
		}
		utxoId := utxoInfo.UtxoId
		outpoint, err := runestone.OutPointFromUtxo(utxo, utxoId)
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetAddressAssets-> runestone.OutPointFromUtxo(%s, %d) err:%v", utxo, utxoId, err)
		}
		balances := s.outpointToBalancesTbl.Get(outpoint)
		for _, balance := range balances {
			runeEntry := s.idToEntryTbl.Get(&balance.RuneId)

			if spaceRuneLotMap[runeEntry.SpacedRune] == nil {
				symbol := defaultRuneSymbol
				if runeEntry.Symbol != nil {
					symbol = *runeEntry.Symbol
				}
				spaceRuneLotMap[runeEntry.SpacedRune] = &RuneBalance{
					Balance:      runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0}),
					Divisibility: runeEntry.Divisibility,
					Symbol:       symbol,
				}
			}
			spaceRuneLotMap[runeEntry.SpacedRune].Balance.AddAssign(&balance.Lot)
		}
	}

	total := uint64(len(spaceRuneLotMap))
	ret := make([]*AddressAsset, total)
	var i = 0
	for spacedRune, runBalance := range spaceRuneLotMap {
		addressLot := &AddressAsset{
			Rune:         spacedRune.String(),
			Balance:      *runBalance.Balance.Value,
			Divisibility: runBalance.Divisibility,
			Symbol:       runBalance.Symbol,
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
	utxo, err := s.RpcService.GetUtxoByID(utxoId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetUtxoAssets-> GetUtxoByID(%d) err:%v", utxoId, err)
	}
	outpoint, err := runestone.OutPointFromUtxo(utxo, utxoId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetUtxoAssets-> runestone.OutPointFromUtxo(%s, %d) err:%v", utxo, utxoId, err)
	}
	balances := s.outpointToBalancesTbl.Get(outpoint)
	ret := make([]*UtxoAsset, len(balances))
	for i, balance := range balances {
		runeEntry := s.idToEntryTbl.Get(&balance.RuneId)
		symbol := defaultRuneSymbol
		if runeEntry.Symbol != nil {
			symbol = *runeEntry.Symbol
		}
		ret[i] = &UtxoAsset{
			Rune:         runeEntry.SpacedRune.String(),
			Balance:      *balance.Lot.Value,
			Divisibility: runeEntry.Divisibility,
			Symbol:       symbol,
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
	utxo, err := s.RpcService.GetUtxoByID(utxoId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.IsExistAsset-> GetUtxoByID(%d) err:%v", utxoId, err)
	}
	outpoint, err := runestone.OutPointFromUtxo(utxo, utxoId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.IsExistAsset-> runestone.OutPointFromUtxo(%s, %d) err:%v", utxo, utxoId, err)
	}
	balances := s.outpointToBalancesTbl.Get(outpoint)
	total := len(balances)
	return total > 0
}
