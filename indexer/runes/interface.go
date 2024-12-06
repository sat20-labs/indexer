package runes

import (
	"sort"

	"github.com/sat20-labs/indexer/common"
)

func (p *Indexer) GetRune(runeName string) *common.Ticker {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return nil
}

func (p *Indexer) GetMint(runeName string) *common.Mint {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return nil
}

// 获取该ticker的holder和持有的utxo
// return: key, address; value, utxos
func (p *Indexer) GetHoldersWithRune(runeName string) map[uint64][]uint64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	mp := make(map[uint64][]uint64, 0)
	return mp
}

// return: 按铸造时间排序的铸造历史
func (p *Indexer) GetMintHistory(runeName string, start, limit int) []any {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	result := make([]any, 0)
	sort.Slice(result, func(i, j int) bool {
		return true
	})

	end := len(result)
	if start >= end {
		return nil
	}
	if start+limit < end {
		end = start + limit
	}

	return result[start:end]
}

func (p *Indexer) GetMintHistoryWithAddress(addressId uint64, tick string, start, limit int) ([]any, int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]any, 0)
	sort.Slice(result, func(i, j int) bool {
		return true
	})

	total := len(result)
	end := total
	if start >= end {
		return nil, 0
	}
	if start+limit < end {
		end = start + limit
	}

	return result[start:end], total
}

func (p *Indexer) GetMintAmount(tick string) (int64, int64) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	amount := int64(0)

	return amount, 0
}
