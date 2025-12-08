package indexer

import (
	"github.com/sat20-labs/indexer/common"
)


func (b *IndexerMgr) GetRunesTickerMap() map[string]*common.TickerInfo {
	result := make(map[string]*common.TickerInfo)
	runeInfos := b.RunesIndexer.GetAllRuneInfos()
	for _, runeInfo := range runeInfos {
		assetName := common.TickerName{
			Protocol: common.PROTOCOL_NAME_RUNES,
			Type:     common.ASSET_TYPE_FT,
			Ticker:   runeInfo.Name,
		}

		tickerInfo := &common.TickerInfo{
			AssetName:       assetName,
			DisplayName:     runeInfo.Id,
			Id:              int64(runeInfo.Number),
			Divisibility:    int(runeInfo.Divisibility),
			StartBlock:      0,
			EndBlock:        0,
			SelfMint:        int(runeInfo.PreminePercentage),
			DeployHeight:    runeInfo.BlockHeight(),
			DeployBlocktime: int64(runeInfo.Timestamp),
			DeployTx:        runeInfo.Etching,
			Limit:           "",
			N:               0,
			TotalMinted:     common.NewDecimalFromUint128(runeInfo.Supply, int(runeInfo.Divisibility)).String(),
			MintTimes:       0,
			MaxSupply:       common.NewDecimalFromUint128(runeInfo.MaxSupply, int(runeInfo.Divisibility)).String(),
			HoldersCount:    int(runeInfo.HolderCount),
			InscriptionId:   "",
			InscriptionNum:  0,
			Description:     "",
			Rarity:          "",
			DeployAddress:   "",
			Content:         []byte{},
			ContentType:     "",
			Delegate:        "",
		}

		if runeInfo.MintInfo != nil {
			tickerInfo.MintTimes = runeInfo.MintInfo.Mints.Big().Int64()
			tickerInfo.Limit = common.NewDecimalFromUint128(runeInfo.MintInfo.Amount, tickerInfo.Divisibility).String()
		}
		result[assetName.String()] = tickerInfo
	}
	return result
}

func (b *IndexerMgr) GetRunesTickerMapV2() []string {
	return b.RunesIndexer.GetAllRuneIds()
}

func (p *IndexerMgr) GetRunesTickerV2(tickerName string) *common.TickerInfo {
	ticker := p.RunesIndexer.GetRuneInfo(tickerName)
	if ticker == nil {
		return nil
	}
	result := &common.TickerInfo{}
	result.Protocol = common.PROTOCOL_NAME_RUNES
	result.Type = common.ASSET_TYPE_FT
	result.Ticker = ticker.Name
	result.DisplayName = ticker.Id
	result.Id = int64(ticker.Number)
	result.Divisibility = int(ticker.Divisibility)

	decimal := common.NewDecimalFromUint128(ticker.Supply, result.Divisibility)
	result.TotalMinted = decimal.String()
	decimal = common.NewDecimalFromUint128(ticker.MaxSupply, result.Divisibility)
	result.MaxSupply = decimal.String()
	if ticker.MintInfo != nil {
		result.MintTimes = ticker.MintInfo.Mints.Big().Int64()
		decimal = common.NewDecimalFromUint128(ticker.MintInfo.Amount, result.Divisibility)
		result.Limit = decimal.String()
	}
	result.SelfMint = int(ticker.PreminePercentage)

	result.DeployHeight = ticker.BlockHeight()
	result.DeployBlocktime = int64(ticker.Timestamp)
	result.DeployTx = ticker.Etching

	holders := ticker.HolderCount
	result.HoldersCount = int(holders)
	result.InscriptionId = ""
	result.Description = ""
	result.Rarity = ""
	result.DeployAddress = ""
	result.ContentType = ""
	result.Delegate = ""

	return result
}

func (b *IndexerMgr) GetRunesMintAmount(tickerName string) (*common.Decimal, int64) {
	info := b.RunesIndexer.GetRuneInfo(tickerName)
	if info == nil {
		return nil, 0
	}
	times := int64(0)
	if info.MintInfo != nil {
		times = info.MintInfo.Mints.Big().Int64()
	}
	return common.NewDecimalFromUint128(info.Supply, 0), times
}

func (b *IndexerMgr) GetRunesDBVer() string {
	return "1.0.0"
}

func (p *IndexerMgr) GetRunesMintHistoryWithAddress(addressId uint64,
	ticker string, start int, limit int) ([]*common.MintInfo, int) {
	result := make([]*common.MintInfo, 0)
	infos, total := p.RunesIndexer.GetAddressMintHistory(ticker, addressId, uint64(start), uint64(limit))
	for _, info := range infos {
		result = append(result, &common.MintInfo{
			Id:            int64(info.Number),
			Address:       p.GetAddressById(info.AddressId),
			Amount:        info.Amount.String(),
			Height:        int(info.Height),
			InscriptionId: info.Utxo,
		})
	}
	return result, int(total)
}

func (p *IndexerMgr) GetRunesMintHistory(
	ticker string, start int, limit int) ([]*common.MintInfo, int) {
	result := make([]*common.MintInfo, 0)
	infos, total := p.RunesIndexer.GetMintHistory(ticker, uint64(start), uint64(limit))
	for _, info := range infos {
		result = append(result, &common.MintInfo{
			Id:            int64(info.Number),
			Address:       p.GetAddressById(info.AddressId),
			Amount:        info.Amount.String(),
			Height:        int(info.Height),
			InscriptionId: info.Utxo,
		})
	}
	return result, int(total)
}
