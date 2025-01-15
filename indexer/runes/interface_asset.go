package runes

import (
	"sort"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

type AddressLot struct {
	Address string
	Amount  *uint128.Uint128
}
type AddressIdToAddressLotMap map[uint64]*AddressLot

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

	addressIdToAddressLotMap := make(AddressIdToAddressLotMap)
	for _, balance := range balances {
		if addressIdToAddressLotMap[balance.AddressId] == nil {
			addressIdToAddressLotMap[balance.AddressId] = &AddressLot{
				Address: string(balance.Address),
				Amount:  &uint128.Uint128{Lo: 0, Hi: 0},
			}
		}
		v128 := addressIdToAddressLotMap[balance.AddressId].Amount.Add(balance.Balance.Value)
		addressIdToAddressLotMap[balance.AddressId].Amount = &v128
	}

	total := uint64(len(addressIdToAddressLotMap))
	ret = make(map[uint64]*common.Decimal, total)
	var i = 0

	for addressId, addressLot := range addressIdToAddressLotMap {
		decimal := common.NewDecimalFromUint128(*addressLot.Amount, 0)
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

	r := s.idToEntryTbl.Get(rid)
	if r == nil {
		common.Log.Errorf("RuneIndexer.GetAllAddressBalances-> idToEntryTbl.Get(%s) rune not found, runeId: %s", rid.Hex(), runeId)
		return nil, 0
	}

	balances, err := s.runeIdAddressToBalanceTbl.GetBalances(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAllAddressBalances-> runeIdAddressToBalanceTbl.GetBalances(%s) err:%v", rid.Hex(), err.Error())
	}

	addressIdToAddressLotMap := make(AddressIdToAddressLotMap)
	for _, balance := range balances {
		if addressIdToAddressLotMap[balance.AddressId] == nil {
			addressIdToAddressLotMap[balance.AddressId] = &AddressLot{
				Address: string(balance.Address),
				Amount:  &uint128.Uint128{Lo: 0, Hi: 0},
			}
		}
		v128 := addressIdToAddressLotMap[balance.AddressId].Amount.Add(balance.Balance.Value)
		addressIdToAddressLotMap[balance.AddressId].Amount = &v128
	}

	total := uint64(len(addressIdToAddressLotMap))
	ret := make([]*AddressBalance, total)
	var i = 0
	for addressId, addressLot := range addressIdToAddressLotMap {
		addressLot := &AddressBalance{
			AddressId:    addressId,
			Address:      addressLot.Address,
			Balance:      *addressLot.Amount,
			Divisibility: r.Divisibility,
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
		common.Log.Infof("RuneIndexer.GetAllUtxoBalances-> runestone.RuneIdFromHex(%s) err:%s", runeId, err.Error())
		return nil, 0
	}

	r := s.idToEntryTbl.Get(rid)
	if r == nil {
		common.Log.Errorf("RuneIndexer.GetAllUtxoBalances-> idToEntryTbl.Get(%s) rune not found, runeId: %s", rid.Hex(), runeId)
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
		totalAmount.AddAssign(&balance.Balance)
		addressLot := &UtxoBalance{
			UtxoId:       balance.OutPoint.UtxoId,
			Balance:      balance.Balance.Value,
			Divisibility: r.Divisibility,
		}
		ret.Balances[i] = addressLot
		i++
	}
	ret.Total = totalAmount.Value

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
		RuneEntry *runestone.RuneEntry
		Balance   uint128.Uint128
	}
	type SpaceRuneLotMap map[runestone.SpacedRune]*RuneBalance
	spaceRuneLotMap := make(SpaceRuneLotMap)

	balances, err := s.addressOutpointToBalancesTbl.GetBalances(addressId)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetAddressAssets-> GetBalances(%d) err:%v", addressId, err)
	}
	for _, balance := range balances {
		runeEntry := s.idToEntryTbl.Get(balance.RuneId)
		sr := runeEntry.SpacedRune
		common.Log.Infof("runeEntry.SpacedRune: %v", sr.String())
		if spaceRuneLotMap[sr] == nil {
			spaceRuneLotMap[sr] = &RuneBalance{
				Balance:   uint128.Uint128{Lo: 0, Hi: 0},
				RuneEntry: runeEntry,
			}
		}
		amount := spaceRuneLotMap[sr].Balance.Add(balance.Balance.Value)
		spaceRuneLotMap[sr].Balance = amount
	}

	total := uint64(len(spaceRuneLotMap))
	ret := make([]*AddressAsset, total)
	var i = 0
	for spacedRune, runeBalance := range spaceRuneLotMap {
		runeEntry := runeBalance.RuneEntry
		amount := runeBalance.Balance
		symbol := defaultRuneSymbol
		if runeEntry.Symbol != nil {
			symbol = *runeEntry.Symbol
		}
		addressLot := &AddressAsset{
			Rune:         spacedRune.String(),
			RuneId:       runeEntry.RuneId.String(),
			Balance:      amount,
			Divisibility: runeEntry.Divisibility,
			Symbol:       symbol,
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
		r := s.idToEntryTbl.Get(&runeIdLot.RuneId)
		symbol := defaultRuneSymbol
		if r.Symbol != nil {
			symbol = *r.Symbol
		}
		ret[i] = &UtxoAsset{
			Rune:         r.SpacedRune.String(),
			RuneId:       r.RuneId.String(),
			Balance:      runeIdLot.Lot.Value,
			Divisibility: r.Divisibility,
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
