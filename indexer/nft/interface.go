package nft

import (
	"sort"
	"strconv"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (p *NftIndexer) HasNftInUtxo(utxoId uint64) bool {
	sats := p.GetSatsWithUtxo(utxoId)
	return len(sats) != 0
}

func (p *NftIndexer) GetNftWithInscriptionId(inscriptionId string) *common.Nft {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.getNftWithInscriptionId(inscriptionId)
}

func (p *NftIndexer) getNftWithInscriptionId(inscriptionId string) *common.Nft {
	if inscriptionId == "" {
		return nil
	}

	nft := p.getNftInBufferWithInscriptionId(inscriptionId)
	if nft != nil {
		return nft
	}

	var value InscriptionInDB
	key := GetInscriptionIdKey(inscriptionId)
	err := db.GetValueFromDB([]byte(key), &value, p.db)
	if err != nil {
		//common.Log.Errorf("GetValueFromDB with inscription %s failed. %v", inscriptionId, err)
		return nil
	}

	nfts := p.getNftsWithSat(value.Sat)
	if nfts != nil {
		for _, nftId := range nfts.Nfts {
			if nftId == value.Id {
				nft, err := p.loadNftFromDB(nftId)
				if err != nil {
					return nil
				}
				nft.OwnerAddressId = nfts.OwnerAddressId
				nft.UtxoId = nfts.UtxoId
				nft.Offset = nfts.Offset

				return nft
			}
		}
	}

	return nil
}

func (p *NftIndexer) GetNftHolderWithInscriptionId(inscriptionId string) uint64 {
	nft := p.GetNftWithInscriptionId(inscriptionId)
	if nft != nil {
		return nft.OwnerAddressId
	}
	return common.INVALID_ID
}

// key: sat
func (p *NftIndexer) GetSatsWithUtxo(utxoId uint64) map[int64]int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	sats := p.getSatsWithUtxo(utxoId)
	if len(sats) == 0 {
		return nil
	}

	result := make(map[int64]int64)
	for sat, offset := range sats {
		if _, ok := p.disabledSats[sat]; ok {
			continue
		}
		result[sat] = offset
	}

	return result
}

// key: sat
func (p *NftIndexer) getSatsWithUtxo(utxoId uint64) map[int64]int64 {
	sats, ok := p.utxoMap[utxoId]
	if !ok {
		var value NftsInUtxo
		err := loadUtxoValueFromDB(utxoId, &value, p.db)
		if err != nil {
			return nil
		}
		sats = make(map[int64]int64)
		for _, item := range value.Sats {
			sats[item.Sat] = item.Offset
		}
		// p.utxoMap[utxoId] = sats, 如果设置，就要同步更新satMap，否则会在transfer中导致异常
	}

	return sats
}

func (p *NftIndexer) GetNftsWithAddress(utxoMap map[uint64]int64) []*common.Nft {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	// 因为数据库的原因，先统一读出sat，再读出nft
	utxos := make([]uint64, 0, len(utxoMap))
	for utxoId := range utxoMap {
		utxos = append(utxos, utxoId)
	}
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i] < utxos[j]
	})

	sats := make([]int64, 0, len(utxos))
	for _, utxoId := range utxos {
		satsmap := p.getSatsWithUtxo(utxoId)
		for sat := range satsmap {
			sats = append(sats, sat)
		}
	}
	sort.Slice(sats, func(i, j int) bool {
		return sats[i] < sats[j]
	})

	nftmap := make(map[int64]*common.Nft)
	nfts := make([]int64, 0, len(sats))
	for _, sat := range sats {
		info := p.getNftsWithSat(sat)
		for _, id := range info.Nfts {
			nfts = append(nfts, id)
			nftmap[id] = &common.Nft{Base: nil,
			OwnerAddressId: info.OwnerAddressId, UtxoId: info.UtxoId, Offset: info.Offset}
		}
	}
	sort.Slice(nfts, func(i, j int) bool {
		return nfts[i] < nfts[j]
	})

	result := make([]*common.Nft, 0, len(nftmap))
	for _, nftId := range nfts {
		base := p.getNftBaseWithId(nftId)
		if base == nil {
			continue
		}
		nft := nftmap[nftId]
		nft.Base = base
		result = append(result, nft)
	}

	return result
}

func (p *NftIndexer) GetNftsWithUtxo(utxoId uint64) []*common.Nft {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.getNftsWithUtxo(utxoId)
}

func (p *NftIndexer) getNftsWithUtxo(utxoId uint64) []*common.Nft {
	sats := p.getSatsWithUtxo(utxoId)

	result := make([]*common.Nft, 0)
	for sat := range sats {
		if _, ok := p.disabledSats[sat]; ok {
			continue
		}
		info := p.getNftsWithSat(sat)
		if info != nil {
			for _, nftId := range info.Nfts {
				base := p.getNftBaseWithId(nftId)
				if base == nil {
					continue
				}
				result = append(result, &common.Nft{Base: base,
					OwnerAddressId: info.OwnerAddressId, UtxoId: utxoId, Offset: info.Offset})
			}
		}
	}

	return result
}

func (p *NftIndexer) GetNftWithId(id int64) *common.Nft {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.getNftWithId(id)
}

func (p *NftIndexer) getNftWithId(id int64) *common.Nft {

	nft := p.getNftInBuffer(id)
	if nft != nil {
		return nft
	}

	buckDB := NewBuckStore(p.db)
	bv, err := buckDB.Get(id)
	if err != nil {
		return nil
	}

	nfts := &common.NftsInSat{}
	err = loadNftsInSatFromDB(bv.Sat, nfts, p.db)
	if err != nil {
		return nil
	}

	for _, nftId := range nfts.Nfts {
		if nftId == id {
			nft, err := p.loadNftFromDB(nftId)
			if err != nil {
				return nil
			}
			nft.OwnerAddressId = nfts.OwnerAddressId
			nft.UtxoId = nfts.UtxoId
			nft.Offset = nfts.Offset

			return nft
		}
	}

	return nil
}

func (p *NftIndexer) getNftBaseWithId(id int64) *common.InscribeBaseContent {
	nft := p.getNftInBuffer(id)
	if nft != nil {
		return nft.Base
	}

	nft, err := p.loadNftFromDB(id)
	if err != nil {
		return nil
	}

	return nft.Base
}

func (p *NftIndexer) GetNftsWithSat(sat int64) *common.NftsInSat {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if _, ok := p.disabledSats[sat]; ok {
		return nil
	}

	return p.getNftsWithSat(sat)
}

func (p *NftIndexer) getNftsWithSat(sat int64) *common.NftsInSat {
	info, ok := p.satMap[sat]
	if ok {
		return info.ToNftsInSat(sat)
	}

	nfts := &common.NftsInSat{}
	err := loadNftsInSatFromDB(sat, nfts, p.db)
	if err != nil {
		return nil
	}

	return nfts
}

func (p *NftIndexer) GetStatus() *common.NftStatus {
	return p.status
}

// 按照铸造时间
func (p *NftIndexer) GetNfts(start, limit int) ([]int64, int) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	end := start + limit

	result := make([]int64, 0)
	buckDB := NewBuckStore(p.db)
	idmap := buckDB.BatchGet(int64(start), int64(end))
	for _, nft := range p.nftAdded {
		idmap[nft.Base.Id] = &BuckValue{nft.Base.Sat}
	}
	for i := start; i < end; i++ {
		_, ok := idmap[int64(i)]
		if ok {
			result = append(result, int64(i))
		}
	}

	return result, len(idmap)
}

// 按照铸造时间
func (p *NftIndexer) GetNftsWithInscriptionAddress(addressId uint64, start, limit int) ([]int64, int) {
	result := p.GetAllNftsWithInscriptionAddress(addressId)

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

// 按照铸造时间
func (p *NftIndexer) GetAllNftsWithInscriptionAddress(addressId uint64) []int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := getNftsWithAddressFromDB(addressId, p.db)
	for _, nft := range p.nftAdded {
		if nft.Base.InscriptionAddress == addressId {
			result = append(result, nft.Base.Id)
		}
	}

	return result
}

func (p *NftIndexer) DisableNftsInUtxo(utxoId uint64, proof []byte) error {
	sats := p.GetSatsWithUtxo(utxoId)
	// 实际上是将上面这所有的聪disable了

	p.mutex.Lock()
	defer p.mutex.Unlock()

	for sat := range sats {
		p.disabledSats[sat] = true
		saveDisabledSatToDB(sat, proof, p.db)
	}
	return nil
}

// 当前状态还没有设置
func (p *NftIndexer) loadNftFromDB(nftId int64) (*common.Nft, error) {
	var baseContent common.InscribeBaseContent
	err := p.loadNftBaseConentFromDB(nftId, &baseContent)
	if err != nil {
		return nil, err
	}

	nft := &common.Nft{
		Base: &baseContent,
	}

	// 需要外部加写锁
	//p.inscriptionToNftIdMap[baseContent.InscriptionId] = nft
	//p.nftIdToinscriptionMap[baseContent.Id] = nft
	return nft, nil
}

func (p *NftIndexer) loadNftBaseConentFromDB(nftId int64, value *common.InscribeBaseContent) error {
	err := loadNftFromDB(nftId, value, p.db)
	if err != nil {
		return err
	}

	// 恢复相关数据
	id, err := strconv.Atoi(string(value.ContentType))
	if err == nil {
		value.ContentType = []byte(p.contentTypeMap[id])
	}

	if _enable_compress_nft {
		if value.ContentId != 0 {
			content, err := p.getContentById(value.ContentId)
			if err == nil {
				value.Content = []byte(content)
			}
		}
		if len(value.Delegate) != 0 && len(value.Delegate) < 16 {
			id, err := strconv.ParseInt(value.Delegate, 16, 64)
			if err == nil {
				inscriptionId, err := p.getInscriptionIdByNftId(id)
				if err == nil {
					value.Delegate = inscriptionId
				}
			}
		}
		if len(value.Parent) != 0 && len(value.Parent) < 16 {
			id, err := strconv.ParseInt(value.Parent, 16, 64)
			if err == nil {
				inscriptionId, err := p.getInscriptionIdByNftId(id)
				if err == nil {
					value.Parent = inscriptionId
				}
			}
		}
	}

	return nil
}

func (p *NftIndexer) IsEnabled() bool {
	return p.baseIndexer.GetHeight() >= p.enableHeight
}

func (p *NftIndexer) GetContentTye(t int) string {
	return p.contentTypeMap[t]
}
