package indexer

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func (b *IndexerMgr) GetBRC20TickerMap() (map[string]*common.BRC20Ticker, error) {
	return b.brc20Indexer.GetTickerMap()
}


func (b *IndexerMgr) GetBRC20TickerMapV2() (map[string]*common.TickerInfo) {
	result := make(map[string]*common.TickerInfo)
	tickers := b.brc20Indexer.GetAllTickers()
	for _, tickerName := range tickers {
		t := b.GetBRC20TickerV2(tickerName)
		if t != nil {
			assetName := common.TickerName{
				Protocol: common.PROTOCOL_NAME_BRC20,
				Type: common.ASSET_TYPE_FT,
				Ticker: tickerName,
			}
			result[assetName.String()] = t
		}
	}
	return result
}


func (p *IndexerMgr) GetBRC20TickerV2(tickerName string) *common.TickerInfo {
	ticker := p.brc20Indexer.GetTicker(tickerName)
	if ticker == nil {
		return nil
	}
	result := &common.TickerInfo{}
	result.Protocol = common.PROTOCOL_NAME_BRC20
	result.Type = common.ASSET_TYPE_FT
	result.Ticker = strings.ToLower(ticker.Name)
	result.DisplayName = ticker.Name
	result.Id = ticker.Id
	result.Divisibility = int(ticker.Decimal)
	
	result.MaxSupply = ticker.Max.String()
	minted, ms := p.brc20Indexer.GetMintAmount(tickerName)
	result.TotalMinted = minted.String()
	result.MintTimes = ms

	result.Limit = ticker.Limit.String()
	if ticker.SelfMint {
		result.SelfMint = 100
	} else {
		result.SelfMint = 0
	}
	
	result.DeployHeight = int(ticker.Nft.Base.BlockHeight)
	result.DeployBlocktime = ticker.Nft.Base.BlockTime
	result.DeployTx = common.TxIdFromInscriptionId(ticker.Nft.Base.InscriptionId)
	
	holders := p.brc20Indexer.GetHoldersWithTick(ticker.Name)
	result.HoldersCount = len(holders)
	result.InscriptionId = ticker.Nft.Base.InscriptionId
	result.InscriptionNum = ticker.Nft.Base.Id
	result.Description = ""
	result.Rarity = ""
	result.DeployAddress = p.GetAddressById(ticker.Nft.Base.InscriptionAddress)
	result.Content = ticker.Nft.Base.Content
	result.ContentType = string(ticker.Nft.Base.ContentType)
	result.Delegate = ""
	return result
}

func (b *IndexerMgr) GetBRC20MintAmount(tickerName string) (*common.Decimal, int64) {
	return b.brc20Indexer.GetMintAmount(tickerName)
}

func (b *IndexerMgr) GetBRC20DBVer() string {
	return b.brc20Indexer.GetDBVersion()
}

func (p *IndexerMgr) GetBRC20MintHistoryWithAddress(addressId uint64, ticker string, start int, limit int) ([]*common.InscribeBaseContent, int) {
	result := make([]*common.InscribeBaseContent, 0)
	infos, total := p.brc20Indexer.GetMintHistoryWithAddress(addressId, ticker, start, limit)
	for _, info := range infos {
		mint := p.brc20Indexer.GetMint(ticker, info.InscriptionId)
		if mint != nil {
			result = append(result, mint.Nft.Base)
		}
	}
	return result, total
}
