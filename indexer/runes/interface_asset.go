package runes

import (
	"sort"
	// "time"

	// cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/table"
	"lukechampine.com/uint128"
)

type AddressLot struct {
	AddressId uint64
	Amount    *uint128.Uint128
}
type AddressIdToAddressLotMap map[uint64]*AddressLot



// key: addressId, value: amount
func (s *Indexer) GetHoldersWithTick(runeId string) (ret map[uint64]*common.Decimal) {
	runeInfo := s.GetRuneInfo(runeId)
	if runeInfo == nil {
		common.Log.Errorf("%s not found", runeId)
		return nil
	}
	rid, err := runestone.RuneIdFromString(runeInfo.Id)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetHoldersWithTick-> runestone.RuneIdFromString(%s) err:%v", runeId, err.Error())
		return nil
	}
	// r := s.idToEntryTbl.Get(rid)
	// if r == nil {
	// 	common.Log.Errorf("RuneIndexer.GetHoldersWithTick-> idToEntryTbl.Get(%s) rune not found, runeId: %s", rid.Hex(), runeId)
	// 	return nil
	// }
	balances, err := s.runeIdAddressToBalanceTbl.GetBalances(rid)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetHoldersWithTick-> runeIdAddressToBalanceTbl.GetBalances(%s) err:%v", rid.Hex(), err.Error())
	}

	addressIdToAddressLotMap := make(AddressIdToAddressLotMap)
	for _, balance := range balances {
		if addressIdToAddressLotMap[balance.AddressId] == nil {
			addressIdToAddressLotMap[balance.AddressId] = &AddressLot{
				AddressId: balance.AddressId,
				Amount:  &uint128.Uint128{Lo: 0, Hi: 0},
			}
		}
		v128 := addressIdToAddressLotMap[balance.AddressId].Amount.Add(balance.Balance.Value)
		addressIdToAddressLotMap[balance.AddressId].Amount = &v128
	}

	total := uint64(len(addressIdToAddressLotMap))
	ret = make(map[uint64]*common.Decimal, total)

	for addressId, addressLot := range addressIdToAddressLotMap {
		decimal := common.NewDecimalFromUint128(*addressLot.Amount, int(runeInfo.Divisibility))
		ret[addressId] = decimal
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
	runeInfo := s.GetRuneInfo(runeId)
	if runeInfo == nil {
		common.Log.Errorf("%s not found", runeId)
		return nil, 0
	}
	rid, err := runestone.RuneIdFromString(runeInfo.Id)
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
				AddressId: balance.AddressId,
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
	runeInfo := s.GetRuneInfo(runeId)
	if runeInfo == nil {
		common.Log.Errorf("%s not found", runeId)
		return nil, 0
	}
	rid, err := runestone.RuneIdFromString(runeInfo.Id)
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
	if limit == 0 {
		return nil, uint64(len(balances))
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
			Divisibility: runeInfo.Divisibility,
		}
		ret.Balances[i] = addressLot
		i++

		// if runeId == "39241:1" {
		// 	common.Log.Infof("%x: %s\n", balance.OutPoint.UtxoId, balance.Balance.Value.String())
		// }
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
func (s *Indexer) GetAddressAssets(addressId uint64, utxos map[uint64]int64) map[string]*AddressAsset {
	assetMap := make(map[string]*AddressAsset)
	for utxoId := range utxos {
		assets := s.GetUtxoAssets(utxoId)
		for _, asset := range assets {
			old, ok := assetMap[asset.Rune]
			if ok {
				old.Balance = old.Balance.Add(asset.Balance)
			} else {
				assetMap[asset.Rune] = asset
			}
		}
	}

	return assetMap
}

func (s *Indexer) GetAddressAssetWithName(addressId uint64, name string) *common.Decimal {

	runeInfo := s.GetRuneInfo(name)
	if runeInfo == nil {
		return nil
	}

	utxos := s.baseIndexer.GetUTXOs(addressId)
	var balance uint128.Uint128
	for utxoId := range utxos {
		assets := s.GetUtxoAssets(utxoId)
		for _, asset := range assets {
			if asset.Rune == name {
				balance = balance.Add(asset.Balance)
			}
		}
	}

	return common.NewDecimalFromUint128(balance, int(runeInfo.Divisibility))
}

/*
*
desc: 根据utxo获取ticker名字和资产数量
*/
func (s *Indexer) GetUtxoAssets(utxoId uint64) []*UtxoAsset {
	outpoint := table.OutPointFromUtxoId(utxoId)
	outpointToBalancesValue := s.outpointToBalancesTbl.Get(outpoint)
	ret := make([]*UtxoAsset, len(outpointToBalancesValue.RuneIdLots))
	for i, runeIdLot := range outpointToBalancesValue.RuneIdLots {
		r := s.idToEntryTbl.Get(&runeIdLot.RuneId)
		if r == nil {
			continue
		}
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
	outpoint := table.OutPointFromUtxoId(utxoId)
	outpointToBalancesValue := s.outpointToBalancesTbl.Get(outpoint)
	total := len(outpointToBalancesValue.RuneIdLots)
	return total > 0
}
