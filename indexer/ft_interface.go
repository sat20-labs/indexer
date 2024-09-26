package indexer

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
)

// 检查一个tick中的哪些nft已经被分拆
func (b *IndexerMgr) GetSplittedInscriptionsWithTick(tickerName string) []string {
	return b.ftIndexer.GetSplittedInscriptionsWithTick(tickerName)
}

func (b *IndexerMgr) GetMintPermissionInfo(ticker, address string) int64 {
	ticker = strings.ToLower(ticker)
	return b.getMintAmount(ticker, b.GetAddressId(address))
}

func (b *IndexerMgr) GetTickerMap() (map[string]*common.Ticker, error) {
	return b.ftIndexer.GetTickerMap()
}

func (b *IndexerMgr) GetTicker(ticker string) *common.Ticker {
	return b.ftIndexer.GetTicker(ticker)
}

func (b *IndexerMgr) GetMintAmount(tickerName string) (int64, int64) {
	return b.ftIndexer.GetMintAmount(tickerName)
}

func (b *IndexerMgr) GetOrdxDBVer() string {
	return b.ftIndexer.GetDBVersion()
}

func (p *IndexerMgr) GetFTMintHistoryWithAddress(addressId uint64, ticker string, start int, limit int) ([]*common.InscribeBaseContent, int) {
	result := make([]*common.InscribeBaseContent, 0)
	infos, total := p.ftIndexer.GetMintHistoryWithAddress(addressId, ticker, start, limit)
	for _, info := range infos {
		mint := p.ftIndexer.GetMint(info.InscriptionId)
		if mint != nil {
			result = append(result, mint.Base)
		}
	}
	return result, total
}
