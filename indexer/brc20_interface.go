package indexer

import (
	"github.com/sat20-labs/indexer/common"
)

func (b *IndexerMgr) GetBRC20TickerMap() (map[string]*common.BRC20Ticker, error) {
	return b.brc20Indexer.GetTickerMap()
}

func (b *IndexerMgr) GetBRC20Ticker(ticker string) *common.BRC20Ticker {
	return b.brc20Indexer.GetTicker(ticker)
}

func (b *IndexerMgr) GetBRC20MintAmount(tickerName string) (common.Decimal, int64) {
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
