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
	list := s.idToEntryTbl.GetListFromDB()
	for _, v := range list {
		supply := v.Supply()
		terms := v.Terms
		percentage := GetPercentage(&v.Premine, &supply)
		runeInfo := &RuneInfo{
			Name:               v.SpacedRune.String(),
			Number:             v.Number,
			Timestamp:          v.Timestamp,
			Id:                 v.RuneId.String(),
			EtchingBlock:       v.RuneId.Block,
			EtchingTransaction: v.RuneId.Tx,
			Supply:             v.Supply(),
			Premine:            v.Pile(v.Premine).String(),
			PreminePercentage:  percentage.String(),
			Burned:             v.Burned,
			Divisibility:       v.Divisibility,
			Symbol:             string(*v.Symbol),
			Turbo:              v.Turbo,
			Etching:            v.Etching,
		}
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
				runeInfo.MintInfo.Amount = v.Pile(*terms.Amount).String()
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
					runeInfo.MintInfo.Progress = mintProgress.String()
				}
			}
		}
		if v.Parent != nil {
			runeInfo.Parent = string(*v.Parent)
		}
		ret = append(ret, runeInfo)
	}

	return nil, 0
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
	name := spaceRune.Rune.String()
	common.Log.Infof("RuneIndexer.GetRuneInfo-> name:%s", name)
	runeId := s.runeToIdTbl.GetFromDB(&spaceRune.Rune)
	if runeId == nil {
		common.Log.Errorf("RuneIndexer.GetRuneInfo-> runeToIdTbl.GetFromDB(%s) rune not found, ticker: %s", spaceRune.String(), ticker)
		return nil
	}
	runeEntry := s.idToEntryTbl.GetFromDB(runeId)
	ret = &RuneInfo{
		Name:               runeEntry.SpacedRune.String(),
		Number:             runeEntry.Number,
		Timestamp:          runeEntry.Timestamp,
		Id:                 runeEntry.RuneId.String(),
		EtchingBlock:       runeEntry.RuneId.Block,
		EtchingTransaction: runeEntry.RuneId.Tx,
		Supply:             runeEntry.Supply(),
		Premine:            runeEntry.Pile(runeEntry.Premine).String(),
		Burned:             runeEntry.Burned,
		Divisibility:       runeEntry.Divisibility,
		Symbol:             string(*runeEntry.Symbol),
		Turbo:              runeEntry.Turbo,
		Etching:            runeEntry.Etching,
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
