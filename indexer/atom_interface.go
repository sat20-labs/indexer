package indexer

import "github.com/sat20-labs/indexer/common"

func (b *IndexerMgr) GetAtomTickerMapV2(start, limit int) ([]string, int) {
	return b.atomIndexer.GetTickersWithRange(start, limit)
}

func (b *IndexerMgr) GetAtomTickerV2(tickerName string) *common.TickerInfo {
	return b.atomIndexer.GetTickerInfo(tickerName)
}

func (b *IndexerMgr) GetAtomMintHistoryWithAddress(addressId uint64, ticker string, start int, limit int) ([]*common.MintInfo, int) {
	result, total := b.atomIndexer.GetMintHistoryWithAddress(addressId, ticker, start, limit)
	for _, item := range result {
		if item.Address == "" {
			item.Address = b.GetAddressById(addressId)
		}
	}
	return result, total
}

func (b *IndexerMgr) GetAtomDBVer() string {
	return b.atomIndexer.GetDBVersion()
}
