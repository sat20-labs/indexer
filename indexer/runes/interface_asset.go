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
	r := s.idToEntryTbl.Get(rid)
	if r == nil {
		common.Log.Errorf("RuneIndexer.GetHoldersWithTick-> idToEntryTbl.Get(%s) rune not found, runeId: %s", rid.Hex(), runeId)
		return nil
	}
	balances, err := s.runeIdAddressToBalanceTbl.GetBalances(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetHoldersWithTick-> runeIdAddressToBalanceTbl.GetBalances(%s) err:%v", rid.Hex(), err.Error())
	}

	type AddressLot struct {
		Address string
		Amount  *runestone.Lot
	}
	type AddressIdToAddressLotMap map[uint64]*AddressLot
	addressIdToAddressLotMap := make(AddressIdToAddressLotMap)

	for _, balance := range balances {
		if addressIdToAddressLotMap[balance.AddressId] == nil {
			addressIdToAddressLotMap[balance.AddressId] = &AddressLot{
				Address: string(balance.Address),
				Amount:  runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0}),
			}
		}
		lot := addressIdToAddressLotMap[balance.AddressId].Amount.Add(runestone.NewLot(balance.Balance.Value))
		addressIdToAddressLotMap[balance.AddressId].Amount = &lot
	}

	total := uint64(len(addressIdToAddressLotMap))
	ret = make(map[uint64]*common.Decimal, total)
	var i = 0

	for addressId, addressLot := range addressIdToAddressLotMap {
		decimal := common.NewDecimalFromUint128(*addressLot.Amount.Value, 0)
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
	rid, err := runestone.RuneIdFromString(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllAddressBalances-> runestone.RuneIdFromString(%s) err:%v", runeId, err.Error())
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

	balances, err := s.runeIdAddressToBalanceTbl.GetBalances(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> runeIdAddressToBalanceTbl.GetBalances(%s) err:%v", rid.Hex(), err.Error())
	}

	type AddressLot struct {
		Address string
		Amount  *runestone.Lot
	}
	type AddressIdToAddressLotMap map[uint64]*AddressLot
	addressIdToAddressLotMap := make(AddressIdToAddressLotMap)

	for _, balance := range balances {
		if addressIdToAddressLotMap[balance.AddressId] == nil {
			addressIdToAddressLotMap[balance.AddressId] = &AddressLot{
				Address: string(balance.Address),
				Amount:  runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0}),
			}
		}
		lot := addressIdToAddressLotMap[balance.AddressId].Amount.Add(runestone.NewLot(balance.Balance.Value))
		addressIdToAddressLotMap[balance.AddressId].Amount = &lot
	}

	total := uint64(len(addressIdToAddressLotMap))
	ret := make([]*AddressBalance, total)
	var i = 0
	for addressId, addressLot := range addressIdToAddressLotMap {
		// pile := r.Pile(*addressLot.Amount.Value)
		addressLot := &AddressBalance{
			AddressId: addressId,
			Address:   addressLot.Address,
			Balance:   *addressLot.Amount.Value,
			// Pile:      &pile,
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
	rid, err := runestone.RuneIdFromHex(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetAllUtxoBalances-> runestone.RuneIdFromHex(%s) err:%s", runeId, err.Error())
		return nil, 0
	}

	balances, err := s.runeIdOutpointToBalanceTbl.GetBalances(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAllUtxoBalances-> runeIdOutpointToBalanceTbl.GetBalances(%s) err:%s", rid.Hex(), err.Error())
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
			Utxo:     balance.OutPoint.Hex(),
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
	rid, err := runestone.RuneIdFromHex(runeId)
	if err != nil {
		common.Log.Infof("RuneIndexer.SlowGetAllUtxoBalances-> runestone.RuneIdFromHex(%s) err:%s", runeId, err.Error())
		return nil, 0
	}
	outpoints, err := s.runeIdToOutpointTbl.GetOutpoints(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.SlowGetAllUtxoBalances-> runeIdToOutpointTbl.GetOutpoints(%s) err:%s", rid.Hex(), err.Error())
	}
	if len(outpoints) == 0 {
		return nil, 0
	}

	type OutpointLotsMap map[runestone.OutPoint]*runestone.Lot
	outpointLotsMap := make(OutpointLotsMap)
	totalAmount := runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
	for _, outpoint := range outpoints {
		outpointToBalancesValue := s.outpointToBalancesTbl.Get(outpoint)
		for _, balance := range outpointToBalancesValue.RuneIdLots {
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
			Utxo:     outpoint.Hex(),
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
	type RuneBalance struct {
		Balance      *runestone.Lot
		Divisibility uint8
		Symbol       rune
		RuneId       string
	}
	type SpaceRuneLotMap map[runestone.SpacedRune]*RuneBalance
	spaceRuneLotMap := make(SpaceRuneLotMap)

	balances, err := s.addressOutpointToBalancesTbl.GetBalances(addressId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAddressAssets-> GetBalances(%d) err:%v", addressId, err)
	}

	for _, balance := range balances {
		runeEntry := s.idToEntryTbl.Get(balance.RuneId)
		if spaceRuneLotMap[runeEntry.SpacedRune] == nil {
			symbol := defaultRuneSymbol
			if runeEntry.Symbol != nil {
				symbol = *runeEntry.Symbol
			}
			spaceRuneLotMap[runeEntry.SpacedRune] = &RuneBalance{
				Balance:      runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0}),
				Divisibility: runeEntry.Divisibility,
				Symbol:       symbol,
				RuneId:       runeEntry.RuneId.String(),
			}
		}
		amount, err := runeEntry.Pile(*balance.Balance.Value).Uint128()
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetAddressAssets-> runeEntry.Pile(%s).Uint128() err:%s", balance.Balance.Value.String(), err.Error())
		}
		spaceRuneLotMap[runeEntry.SpacedRune].Balance.AddAssign(runestone.NewLot(amount))
	}

	total := uint64(len(spaceRuneLotMap))
	ret := make([]*AddressAsset, total)
	var i = 0
	for spacedRune, runBalance := range spaceRuneLotMap {
		addressLot := &AddressAsset{
			Rune:         spacedRune.String(),
			RuneId:       runBalance.RuneId,
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
	outpoint := runestone.OutPointFromUtxoId(utxoId)
	outpointToBalancesValue := s.outpointToBalancesTbl.Get(outpoint)
	ret := make([]*UtxoAsset, len(outpointToBalancesValue.RuneIdLots))
	for i, runeIdLot := range outpointToBalancesValue.RuneIdLots {
		runeEntry := s.idToEntryTbl.Get(&runeIdLot.RuneId)
		symbol := defaultRuneSymbol
		if runeEntry.Symbol != nil {
			symbol = *runeEntry.Symbol
		}
		ret[i] = &UtxoAsset{
			Rune:         runeEntry.SpacedRune.String(),
			RuneId:       runeEntry.RuneId.String(),
			Balance:      *runeIdLot.Lot.Value,
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
	outpoint := runestone.OutPointFromUtxoId(utxoId)
	outpointToBalancesValue := s.outpointToBalancesTbl.Get(outpoint)
	total := len(outpointToBalancesValue.RuneIdLots)
	return total > 0
}
