package runes

import (
	"strconv"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

// TODO
//  1. 资产数量相关数据采用decimal表达, common.decimal.go NewDecimal
//  2. 接口：
//     获取所有tickers的名字, (在接口中的方法里面需要得到所有RUNEINFO) ok
//     判断一个ticker是否已经被部署(在接口中的方法里面需要一个判断一个TICKER是否存在)
//     根据ticker名字获取ticker信息（参考brc20和ft的ticker信息，加上runes特有的信息） ok
//     根据ticker名字获取所有持有者地址和持有数量
//     根据ticker名字获取所有的带有该ticker的utxo和该utxo中的资产数量
//     根据ticker名字获取铸造历史
//     根据地址获取该地址所有ticker和持有的数量
//     根据地址获取指定ticker的铸造历史
//     根据utxo获取ticker名字和资产数量（多个）
//     判断utxo中是否有runes资产

// 获取所有tickers的名字, (在接口中的方法里面需要得到所有RUNEINFO)
func (s *Indexer) GetRuneInfoList(start, limit uint64) (ret []*RuneInfo, total uint64) {
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

			_, err := v.Mintable(s.status.Height + 1)
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

// 根据ticker名字获取ticker信息
func (s *Indexer) GetRuneInfo(ticker string) (ret *RuneInfo) {
	r, err := runestone.RuneFromString(ticker)
	if err != nil {
		common.Log.Debugf("RuneIndexer.GetRuneInfo-> runestone.RuneFromString(%s) err:%s", ticker, err.Error())
		return nil
	}
	runeId := s.runeToIdTbl.GetFromDB(r)
	if runeId == nil {
		common.Log.Infof("RuneIndexer.GetRuneInfo-> runeToIdTbl.GetFromDB(%s) rune not found, ticker: %s", r.String(), ticker)
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

func (s *Indexer) GetHolders(ticker string, start, limit int) ([]*runestone.RuneHolder, int) {
	r, err := runestone.RuneFromString(ticker)
	if err != nil {
		common.Log.Debugf("RuneIndexer.GetHolders-> runestone.RuneFromString(%s) err:%v", ticker, err.Error())
		return nil, 0
	}
	holders := s.runeHolderTbl.GetFromDB(r)
	return holders, 0
}

func (s *Indexer) GetMintHistory(ticker string, start, limit int) (runestone.RuneMintHistorys, int) {
	r, err := runestone.RuneFromString(ticker)
	if err != nil {
		common.Log.Debugf("RuneIndexer.GetMintHistory-> runestone.RuneFromString(%s) err:%v", ticker, err.Error())
		return nil, 0
	}
	mintHistorys := s.runeMintHistorysTbl.GetFromDB(r)
	if mintHistorys == nil {
		return nil, 0
	}
	end := len(mintHistorys)
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}
	return mintHistorys[start:end], end
}

func (s *Indexer) GetAddressMintHistory(address runestone.Address, ticker string, start, limit int) (runestone.RuneMintHistorys, int) {
	r, err := runestone.RuneFromString(ticker)
	if err != nil {
		common.Log.Debugf("RuneIndexer.GetAddressMintHistory-> runestone.RuneFromString(%s) err:%v", ticker, err.Error())
		return nil, 0
	}
	ledger := s.runeLedgerTbl.GetFromDB(address)
	if ledger == nil {
		common.Log.Infof("RuneIndexer.GetAddressMintHistory-> runeLedgerTbl.GetFromDB(%s) rune not found, ticker: %s", address, ticker)
		return nil, 0
	}

	mintHistorys := make(runestone.RuneMintHistorys, len(ledger.Assets[*r].Mints))
	mints := ledger.Assets[*r].Mints
	for i, mint := range mints {
		mintHistory := &runestone.RuneMintHistory{
			Address: address,
			Rune:    *r,
			Utxo:    mint.String(),
		}
		mintHistorys[i] = mintHistory
	}

	total := len(mintHistorys)
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}

	return mintHistorys[start:end], total
}

func (s *Indexer) GetMintAmount(ticker string) (mint uint64, supply uint64) {
	runeEntry := s.GetRuneInfo(ticker)
	if runeEntry == nil {
		return 0, 0
	}

	return mint, supply
}
