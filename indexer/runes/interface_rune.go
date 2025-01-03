package runes

import (
	"strconv"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/cli"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

/*
*
desc: 获取所有ticker
*/
func (s *Indexer) GetRuneInfos(start, limit uint64) (ret []*RuneInfo, total uint64) {
	runeEntrys := s.idToEntryTbl.GetList()
	for _, v := range runeEntrys {
		premine, err := v.Pile(v.Premine).Uint128()
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetRuneInfos-> v.Pile(v.Premine).Uint128() err:%s", err.Error())
		}
		supply := v.Supply()
		percentage := GetPercentage(&v.Premine, &supply)
		percentageNum, err := strconv.Atoi(percentage.String())
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetRuneInfos-> strconv.Atoi(%s) err:%s", percentage.String(), err.Error())
		}
		runeInfo := &RuneInfo{
			Name:               v.SpacedRune.String(),
			Number:             v.Number,
			Timestamp:          v.Timestamp,
			Id:                 v.RuneId.String(),
			EtchingBlock:       v.RuneId.Block,
			EtchingTransaction: v.RuneId.Tx,
			Supply:             v.Supply(),
			Premine:            *premine,
			PreminePercentage:  percentageNum,
			Burned:             v.Burned,
			Divisibility:       v.Divisibility,
			Symbol:             string(*v.Symbol),
			Turbo:              v.Turbo,
			Etching:            v.Etching,
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
				amount, err := v.Pile(*terms.Amount).Uint128()
				if err != nil {
					common.Log.Panicf("RuneIndexer.GetRuneInfos-> v.Pile(*terms.Amount).Uint128() err:%s", err.Error())
				}
				runeInfo.MintInfo.Amount = *amount
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
					progress, err := strconv.Atoi(mintProgress.String())
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
desc: 根据ticker名字获取ticker信息（参考brc20和ft的ticker信息，加上runes特有的信息）
*/
func (s *Indexer) GetRuneInfo(ticker string) (ret *RuneInfo) {
	spaceRune, err := runestone.SpacedRuneFromString(ticker)
	if err != nil {
		common.Log.Infof("RuneIndexer.GetRuneInfo-> runestone.SpacedRuneFromString(%s) err:%s", ticker, err.Error())
		return nil
	}
	runeId := s.runeToIdTbl.Get(&spaceRune.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetRuneInfo-> runeToIdTbl.Get(%s) rune not found, ticker: %s", spaceRune.String(), ticker)
		return nil
	}
	runeEntry := s.idToEntryTbl.Get(runeId)
	premine, err := runeEntry.Pile(runeEntry.Premine).Uint128()
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetRuneInfo-> runeEntry.Pile(v.Premine).Uint128() err:%s", err.Error())
	}
	ret = &RuneInfo{
		Name:               runeEntry.SpacedRune.String(),
		Number:             runeEntry.Number,
		Timestamp:          runeEntry.Timestamp,
		Id:                 runeEntry.RuneId.String(),
		EtchingBlock:       runeEntry.RuneId.Block,
		EtchingTransaction: runeEntry.RuneId.Tx,
		Supply:             runeEntry.Supply(),
		Premine:            *premine,
		Burned:             runeEntry.Burned,
		Divisibility:       runeEntry.Divisibility,
		Symbol:             string(*runeEntry.Symbol),
		Turbo:              runeEntry.Turbo,
		Etching:            runeEntry.Etching,
	}
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
			amount, err := runeEntry.Pile(*terms.Amount).Uint128()
			if err != nil {
				common.Log.Panicf("RuneIndexer.GetRuneInfo-> runeEntry.Pile(*terms.Amount).Uint128() err:%s", err.Error())
			}
			ret.MintInfo.Amount = *amount
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
				progress, err := strconv.Atoi(mintProgress.String())
				if err != nil {
					common.Log.Panicf("RuneIndexer.GetRuneInfos-> strconv.Atoi(%s) err:%s", mintProgress.String(), err.Error())
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

/*
*
desc: 判断一个ticker是否已经被部署
*/
func (s *Indexer) IsExistRune(ticker string) bool {
	ret := s.GetRuneInfo(ticker)
	return ret != nil
}

/*
*
desc: 根据edict列表构造edict数据
*/
func (s *Indexer) BuildEdictsData(list []*Edict) (ret []byte, err error) {
	r := runestone.Runestone{Edicts: []runestone.Edict{}}
	for _, edict := range list {
		spacedRune, err := runestone.SpacedRuneFromString(edict.RuneName)
		if err != nil {
			return nil, err
		}
		runeId := s.runeToIdTbl.GetFromDB(&spacedRune.Rune)
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
