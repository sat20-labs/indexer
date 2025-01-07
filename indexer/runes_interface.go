package indexer

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes"
)

func (b *IndexerMgr) GetRunesTickerMapV2() map[string]*common.TickerInfo {
	result := make(map[string]*common.TickerInfo)
	tickers := b.RunesIndexer.GetAllTickers()
	for _, tickerName := range tickers {
		t := b.GetRunesTickerV2(tickerName)
		if t != nil {
			assetName := common.TickerName{
				Protocol: common.PROTOCOL_NAME_RUNES,
				Type: common.ASSET_TYPE_FT,
				Ticker: tickerName,
			}
			result[assetName.String()] = t
		}
	}
	return result
}

func (p *IndexerMgr) GetRunesTickerV2(tickerName string) *common.TickerInfo {

	var ticker *runes.RuneInfo
	if strings.Contains(tickerName, "_") {
		ticker = p.RunesIndexer.GetRuneInfoWithId(tickerName)
	} else {
		ticker = p.RunesIndexer.GetRuneInfoWithName(tickerName)
	}
	if ticker == nil {
		return nil
	}
	result := &common.TickerInfo{}
	result.Protocol = common.PROTOCOL_NAME_RUNES
	result.Type = common.ASSET_TYPE_FT
	result.Ticker = ticker.Id
	result.DisplayName = ticker.Name
	result.Id = int64(ticker.Number)
	result.Divisibility = int(ticker.Divisibility)

	result.TotalMinted = ticker.Supply.String()
	result.MaxSupply = ticker.MaxSupply.String()
	if ticker.MintInfo != nil {
		result.Limit = ticker.MintInfo.Amount.String()
	}
	result.SelfMint = int(ticker.PreminePercentage)

	result.DeployHeight = ticker.BlockHeight()
	result.DeployBlocktime = int64(ticker.Timestamp)
	result.DeployTx = ticker.Etching

	_, holders := p.RunesIndexer.GetAllAddressBalances(ticker.Id, 0, 1)
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
	info := b.RunesIndexer.GetRuneInfoWithId(tickerName)
	if info == nil {
		return nil, 0
	}
	times := int64(0)
	if info.MintInfo != nil {
		times = info.MintInfo.Mints.Big().Int64()
	}
	return common.NewDecimalFromUint128(info.Supply, int(info.Divisibility)), times
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
