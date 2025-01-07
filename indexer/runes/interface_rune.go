package runes

import (
	"strconv"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/cli"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

func (s *Indexer) getRuneInfoWithId(runeId *runestone.RuneId) (ret *RuneInfo) {
	runeEntry := s.idToEntryTbl.Get(runeId)
	// premine, err := runeEntry.Pile(runeEntry.Premine).Uint128()
	// if err != nil {
	// 	common.Log.Panicf("RuneIndexer.getRuneInfoWithId-> runeEntry.Pile(v.Premine).Uint128() err:%s", err.Error())
	// }
	ret = &RuneInfo{
		Name:      runeEntry.SpacedRune.String(),
		Number:    runeEntry.Number,
		Timestamp: runeEntry.Timestamp,
		Id:        runeEntry.RuneId.String(),
		Supply:    runeEntry.Supply(),
		MaxSupply: runeEntry.MaxSupply(),
		// Premine:      *premine,
		Premine:      runeEntry.Premine,
		Burned:       runeEntry.Burned,
		Divisibility: runeEntry.Divisibility,
		Turbo:        runeEntry.Turbo,
		Etching:      runeEntry.Etching,
	}
	symbol := defaultRuneSymbol
	if runeEntry.Symbol != nil {
		symbol = *runeEntry.Symbol
	}
	ret.Symbol = string(symbol)
	terms := runeEntry.Terms
	if terms != nil {
		ret.MintInfo = &MintInfo{}
		if len(terms.Height) > 0 {
			if terms.Height[0] != nil {
				ret.MintInfo.Start = strconv.FormatUint(*terms.Height[0], 10)
			}
			if terms.Height[1] != nil {
				ret.MintInfo.End = strconv.FormatUint(*terms.Height[1], 10)
			}
		}
		if terms.Amount != nil {
			// amount, err := runeEntry.Pile(*terms.Amount).Uint128()
			// if err != nil {
			// 	common.Log.Panicf("RuneIndexer.getRuneInfoWithId-> runeEntry.Pile(*terms.Amount).Uint128() err:%s", err.Error())
			// }
			// ret.MintInfo.Amount = *amount
			ret.MintInfo.Amount = *terms.Amount
		}
		ret.MintInfo.Mints = runeEntry.Mints
		if terms.Cap != nil {
			ret.MintInfo.Cap = *terms.Cap
		}
		ret.MintInfo.Remaining = ret.MintInfo.Cap.Sub(ret.MintInfo.Mints)

		_, err := runeEntry.Mintable(s.Status.Height + 1)
		ret.MintInfo.Mintable = err == nil

		if ret.MintInfo.Mintable {
			if runeEntry.Terms.Cap.Cmp(uint128.Zero) > 0 {
				mintProgress := GetPercentage(&ret.MintInfo.Mints, &ret.MintInfo.Cap)

				progress, err := strconv.ParseFloat(mintProgress.String(), 64)
				if err != nil {
					common.Log.Panicf("RuneIndexer.getRuneInfoWithId-> strconv.Atoi(%s) err:%s", mintProgress.String(), err.Error())
				}
				ret.MintInfo.Progress = progress
			}
		}
	}
	if runeEntry.Parent != nil {
		ret.Parent = string(*runeEntry.Parent)
	}
	return
}

func (s *Indexer) GetAllTickers() []string {
	runesIds := make([]string, 0)
	runeEntrys := s.idToEntryTbl.GetList()
	for _, v := range runeEntrys {
		runesIds = append(runesIds, v.RuneId.String())
	}
	return runesIds
}

/*
*
desc: 获取所有runeInfo
*/
func (s *Indexer) GetRuneInfos(start, limit uint64) (ret []*RuneInfo, total uint64) {
	runeEntrys := s.idToEntryTbl.GetList()
	var i = 0
	for _, v := range runeEntrys {
		common.Log.Infof("RuneIndexer.GetRuneInfos-> runeEntrys index: %d", i)
		// premine, err := v.Pile(v.Premine).Uint128()
		// if err != nil {
		// 	common.Log.Panicf("RuneIndexer.GetRuneInfos-> v.Pile(v.Premine).Uint128() err:%s", err.Error())
		// }
		supply := v.Supply()
		percentage := NewDecimal(&uint128.Zero, 2)
		if supply.Cmp(uint128.Zero) != 0 {
			percentage = GetPercentage(&v.Premine, &supply)
		}
		percentageNum, err := strconv.ParseFloat(percentage.String(), 64)
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetRuneInfos-> strconv.Atoi(%s) err:%s", percentage.String(), err.Error())
		}
		runeInfo := &RuneInfo{
			Name:      v.SpacedRune.String(),
			Number:    v.Number,
			Timestamp: v.Timestamp,
			Id:        v.RuneId.String(),
			Supply:    supply,
			MaxSupply: v.MaxSupply(),
			// Premine:           *premine,
			Premine:           v.Premine,
			PreminePercentage: percentageNum,
			Burned:            v.Burned,
			Divisibility:      v.Divisibility,
			Turbo:             v.Turbo,
			Etching:           v.Etching,
		}
		if v.Symbol != nil {
			runeInfo.Symbol = string(*v.Symbol)
		} else {
			runeInfo.Symbol = string(defaultRuneSymbol)
		}
		terms := v.Terms
		if terms != nil {
			runeInfo.MintInfo = &MintInfo{}
			if len(terms.Height) > 0 {
				if terms.Height[0] != nil {
					runeInfo.MintInfo.Start = strconv.FormatUint(*terms.Height[0], 10)
				}
				if terms.Height[1] != nil {
					runeInfo.MintInfo.End = strconv.FormatUint(*terms.Height[1], 10)
				}
			}
			if terms.Amount != nil {
				// amount, err := v.Pile(*terms.Amount).Uint128()
				// if err != nil {
				// 	common.Log.Panicf("RuneIndexer.GetRuneInfos-> v.Pile(*terms.Amount).Uint128() err:%s", err.Error())
				// }
				// runeInfo.MintInfo.Amount = *amount
				runeInfo.MintInfo.Amount = *terms.Amount
			}
			runeInfo.MintInfo.Mints = v.Mints
			if terms.Cap != nil {
				runeInfo.MintInfo.Cap = *terms.Cap
			}
			runeInfo.MintInfo.Remaining = runeInfo.MintInfo.Cap.Sub(runeInfo.MintInfo.Mints)

			_, err := v.Mintable(s.Status.Height + 1)
			runeInfo.MintInfo.Mintable = err == nil

			if runeInfo.MintInfo.Mintable {
				if v.Terms.Cap.Cmp(uint128.Zero) > 0 {
					mintProgress := GetPercentage(&runeInfo.MintInfo.Mints, &runeInfo.MintInfo.Cap)
					progress, err := strconv.ParseFloat(mintProgress.String(), 64)
					if err != nil {
						common.Log.Panicf("RuneIndexer.GetRuneInfos-> strconv.Atoi(%s) err:%s", mintProgress.String(), err.Error())
					}
					runeInfo.MintInfo.Progress = progress
				}
			}
		}
		if v.Parent != nil {
			runeInfo.Parent = string(*v.Parent)
		}
		ret = append(ret, runeInfo)
		i++
	}

	total = uint64(len(ret))
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
desc: 根据runeId获取rune信息
*/
func (s *Indexer) GetRuneInfoWithId(runeid string) *RuneInfo {
	runeId, err := runestone.RuneIdFromString(runeid)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetRuneInfoWithId-> runestone.RuneIdFromDec(%s) err:%s", runeid, err.Error())
		return nil
	}
	return s.getRuneInfoWithId(runeId)
}

/*
*
desc: 根据runeName名字获取rune信息
*/
func (s *Indexer) GetRuneInfoWithName(runeName string) *RuneInfo {
	spaceRune, err := runestone.SpacedRuneFromString(runeName)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetRuneInfoWithName-> runestone.SpacedRuneFromString(%s) err:%s", runeName, err.Error())
		return nil
	}
	runeId := s.runeToIdTbl.Get(&spaceRune.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetRuneInfoWithName-> runeToIdTbl.Get(%s) rune not found, runeName: %s", spaceRune.String(), runeName)
		return nil
	}
	return s.getRuneInfoWithId(runeId)
}

/*
*
desc: 根据runeName名字获取runeId
*/
func (s *Indexer) GetRuneIdWithName(runeName string) (*runestone.RuneId, error) {
	spaceRune, err := runestone.SpacedRuneFromString(runeName)
	if err != nil {
		return nil, err
	}
	runeId := s.runeToIdTbl.Get(&spaceRune.Rune)
	return runeId, nil
}

/*
*
desc: 判断一个rune是否已经被部署
*/
func (s *Indexer) IsExistRuneWithName(runeName string) bool {
	ret := s.GetRuneInfoWithName(runeName)
	return ret != nil
}

/*
*
desc: 判断一个rune是否已经被部署
*/
func (s *Indexer) IsExistRuneWithId(runeId string) bool {
	ret := s.GetRuneInfoWithId(runeId)
	return ret != nil
}

/*
*
desc: 根据edict列表构造edict数据
*/
func (s *Indexer) BuildEdictsData(list []*Edict) (ret []byte, err error) {
	r := runestone.Runestone{Edicts: []runestone.Edict{}}
	for _, edict := range list {
		runeId, err := runestone.RuneIdFromHex(edict.RuneId)
		if err != nil {
			return nil, err
		}
		r.Edicts = append(r.Edicts, runestone.Edict{
			ID:     *runeId,
			Amount: edict.Amount,
			Output: edict.Output,
		})
	}
	data, err := r.Encipher()
	if err != nil {
		return nil, err
	}
	return data, nil
}

/*
*
desc: 根据edict列表构造edict交易数据
*/
func (s *Indexer) BuildEdictsTxData(
	prvKey *btcec.PrivateKey,
	address string,
	utxos []*cli.Utxo,
	toAmount int64,
	feePerByte int64,
	list []*Edict) (ret []byte, err error) {
	ret, err = s.BuildEdictsData(list)
	if err != nil {
		return nil, err
	}
	ret, err = cli.BuildTransferBTCTx(prvKey, utxos, address, toAmount, feePerByte, s.chaincfgParam, ret)
	if err != nil {
		return nil, err
	}
	return
}
