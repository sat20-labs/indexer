package indexer

import (
	"fmt"
	"sort"

	"github.com/sat20-labs/indexer/common"
)

func (b *IndexerMgr) GetNftStatus() *common.NftStatus {
	return b.nft.GetStatus()
}

func (b *IndexerMgr) GetNftInfo(id int64) *common.Nft {
	return b.nft.GetNftWithId(id)
}

func (b *IndexerMgr) GetNftInfoWithInscriptionId(id string) *common.Nft {
	return b.nft.GetNftWithInscriptionId(id)
}

// result: nft ids
func (b *IndexerMgr) GetNftsWithUtxo(utxoId uint64) []string {
	result := make([]string, 0)
	sats := b.nft.GetSatsWithUtxo(utxoId)
	for sat := range sats {
		info := b.GetNftsWithSat(sat)
		if info != nil {
			for _, nftId := range info.Nfts {
				result = append(result, fmt.Sprintf("%d", nftId))
			}
		}
	}
	return result
}

func (b *IndexerMgr) GetNftsWithSat(sat int64) *common.NftsInSat {
	return b.nft.GetNftsWithSat(sat)
}

func (b *IndexerMgr) GetNfts(start, limit int) ([]int64, int) {
	return b.nft.GetNfts(start, limit)
}

func (b *IndexerMgr) getNftWithAddressInBuffer(address string) []*common.Nft {
	if b.addressToNftMap == nil {
		return b.initAddressToNftMap(address)
	}

	b.mutex.RLock()
	ret, ok := b.addressToNftMap[address]
	if !ok {
		b.mutex.RUnlock()
		ret = b.initAddressToNftMap(address)
	} else {
		b.mutex.RUnlock()
	}

	return ret
}

func (b *IndexerMgr) initAddressToNftMap(address string) []*common.Nft {
	utxoMap, err := b.GetUTXOsWithAddress(address)
	if err != nil {
		common.Log.Warnf("GetNftsWithAddress %s failed. %v", address, err)
		return nil
	}

	nftIds := make([]*common.Nft, 0)
	for utxoId := range utxoMap {
		ids := b.nft.GetNftsWithUtxo(utxoId)
		nftIds = append(nftIds, ids...)
	}

	sort.Slice(nftIds, func(i, j int) bool {
		return nftIds[i].Base.Id > nftIds[j].Base.Id
	})

	b.mutex.Lock()
	if b.addressToNftMap == nil {
		b.addressToNftMap = make(map[string][]*common.Nft)
	}
	b.addressToNftMap[address] = nftIds
	b.mutex.Unlock()
	return nftIds
}

// holder: address
func (b *IndexerMgr) GetNftsWithAddress(address string, start, limit int) ([]*common.Nft, int) {
	nfts := b.getNftWithAddressInBuffer(address)
	total := len(nfts)
	if start >= total {
		return nil, total
	}
	end := total
	if limit > 0 && start+limit < total {
		end = start + limit
	}

	return nfts[start:end], total
}

func (b *IndexerMgr) GetNftAmountWithAddress(address string) map[string]int64 {
	return b.getNftAmountWithAddress(address, nil)
}

func (b *IndexerMgr) getNftAmountWithAddress(address string, unconfirmedSpents map[uint64]*common.TxOutput) map[string]int64 {
	nfts := b.getNftWithAddressInBuffer(address)

	result := make(map[string]int64)
	b.mutex.RLock()
	for _, nft := range nfts {
		if _, ok := unconfirmedSpents[nft.UtxoId]; ok {
			continue
		}
		for k, v := range b.clmap {
			if k.Type == common.ASSET_TYPE_NFT {
				_, ok := v[nft.Base.InscriptionId]
				if ok {
					result[k.Ticker] += 1
				}
			}
		}
	}
	b.mutex.RUnlock()

	return result
}

func (b *IndexerMgr) getNftsWithUtxo(utxoId uint64) map[string]common.AssetOffsets {
	result := make(map[string]common.AssetOffsets)
	sats := b.nft.GetSatsWithUtxo(utxoId)
	for sat := range sats {
		nfts := b.nft.GetNftsWithSat(sat)
		if nfts != nil {
			offsets := common.AssetOffsets{
				{
					Start: nfts.Offset,
					End:   nfts.Offset + 1,
				},
			}
			for _, nftId := range nfts.Nfts {
				result[fmt.Sprintf("%d", nftId)] = offsets
			}
		}
	}
	return result
}

func (p *IndexerMgr) GetNftHistory(start int, limit int) ([]*common.MintAbbrInfo, int) {
	result := make([]*common.MintAbbrInfo, 0)
	ids, total := p.nft.GetNfts(start, limit)
	for _, id := range ids {
		nft := p.nft.GetNftWithId(id)
		if nft != nil {
			info := common.NewMintAbbrInfo2(nft.Base)
			result = append(result, info)
		}
	}
	return result, total
}

// gen address
func (p *IndexerMgr) GetNftHistoryWithAddress(addressId uint64, start int, limit int) ([]*common.MintAbbrInfo, int) {
	result := make([]*common.MintAbbrInfo, 0)
	ids, total := p.nft.GetNftsWithInscriptionAddress(addressId, start, limit)
	for _, id := range ids {
		nft := p.nft.GetNftWithId(id)
		if nft != nil {
			info := common.NewMintAbbrInfo2(nft.Base)
			result = append(result, info)
		}
	}
	return result, total
}
