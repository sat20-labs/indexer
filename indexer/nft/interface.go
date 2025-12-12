package nft

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

func (p *NftIndexer) HasNftInUtxo(utxoId uint64) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	sats, ok := p.utxoMap[utxoId]
	if !ok {
		var value NftsInUtxo
		err := loadUtxoValueFromDB(utxoId, &value, p.db)
		if err != nil {
			return false
		}
		sats = value.Sats
	}
	if len(sats) == 0 {
		return false
	}

	// 过滤disabled的sat
	disableCount := 0
	for _, sat := range sats {
		_, ok := p.disabledSats[sat]
		if ok {
			disableCount++
		}
	}

	return disableCount != len(sats)
}

func (p *NftIndexer) GetNftWithInscriptionId(inscriptionId string) *common.Nft {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if inscriptionId == "" {
		return nil
	}

	nft := p.getNftInBuffer2(inscriptionId)
	if nft != nil {
		return nft
	}

	var value InscriptionInDB
	key := GetInscriptionIdKey(inscriptionId)
	err := db.GetValueFromDB([]byte(key), &value, p.db)
	if err != nil {
		//common.Log.Errorf("GetValueFromDB with inscription %s failed. %v", inscriptionId, err)
		//return nil
	} else {
		nfts := p.getNftsWithSat(value.Sat)
		if nfts != nil {
			for _, nft := range nfts.Nfts {
				if nft.Id == value.Id {
					return &common.Nft{
						Base:           nft,
						OwnerAddressId: nfts.OwnerAddressId, 
						UtxoId: nfts.UtxoId,
						Offset: nfts.Offset,
					}
				}
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
func (p *NftIndexer) GetBoundSatsWithUtxo(utxoId uint64) []int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	value := &NftsInUtxo{}

	loadUtxoValueFromDB(utxoId, value, p.db)

	//if err != nil {
	// 还没有保存到数据库
	// return nil
	//}

	satmap := make(map[int64]bool)
	for _, sat := range value.Sats {
		if _, ok := p.disabledSats[sat]; ok {
			continue
		}
		satmap[sat] = true
	}

	sats, ok := p.utxoMap[utxoId]
	if ok {
		for _, sat := range sats {
			if _, ok := p.disabledSats[sat]; ok {
				continue
			}
			satmap[sat] = true
		}
	}

	result := make([]int64, 0)
	for sat := range satmap {
		result = append(result, sat)
	}

	return result
}

func (p *NftIndexer) GetNftsWithUtxo(utxoId uint64) []*common.Nft {
	sats := p.GetBoundSatsWithUtxo(utxoId)

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]*common.Nft, 0)
	for _, sat := range sats {
		if _, ok := p.disabledSats[sat]; ok {
			continue
		}
		info := p.getNftsWithSat(sat)
		if info != nil {
			for _, nft := range info.Nfts {
				result = append(result, &common.Nft{
					Base: nft,
					OwnerAddressId: info.OwnerAddressId, 
					UtxoId: utxoId,
					Offset: info.Offset,
				})
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
	err = loadNftFromDB(bv.Sat, nfts, p.db)
	if err != nil {
		return nil
	}

	for _, nft := range nfts.Nfts {
		if nft.Id == id {
			return &common.Nft{
				Base:           nft,
				OwnerAddressId: nfts.OwnerAddressId, 
				UtxoId: nfts.UtxoId,
				Offset: nfts.Offset,
			}
		}
	}

	return nil
}

// return sats
func (p *NftIndexer) GetNftsWithRanges(rngs []*common.Range) []int64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]int64, 0)

	for _, rng := range rngs {
		startKey := []byte(GetSatKey(rng.Start))
		endKey := []byte(GetSatKey(rng.Start + rng.Size - 1))
		err := db.IterateRangeInDB(p.db, nil, startKey, endKey, func(key, value []byte) error {
			sat, err := ParseSatKey(string(key))
			if err == nil {
				if _, ok := p.disabledSats[sat]; ok {
					return nil
				}
				result = append(result, sat)
			}
			return err
		})
		if err != nil {
			common.Log.Errorf("IterateRangeInDB failed. %v", err)
		}
	}

	return result
}

func (p *NftIndexer) GetNftsWithSat(sat int64) *common.NftsInSat {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.getNftsWithSat(sat)
}

func (p *NftIndexer) getNftsWithSat(sat int64) *common.NftsInSat {
	if _, ok := p.disabledSats[sat]; ok {
		return nil
	}
	nfts := &common.NftsInSat{}
	err := loadNftFromDB(sat, nfts, p.db)
	addedNfts := p.getNftInBuffer4(sat)
	if len(addedNfts) != 0 {
		if err != nil {
			nfts.OwnerAddressId = addedNfts[0].OwnerAddressId
			nfts.Sat = addedNfts[0].Base.Sat
			nfts.UtxoId = addedNfts[0].UtxoId
			nfts.Offset = addedNfts[0].Offset
		}
		for _, nft := range addedNfts {
			nfts.Nfts = append(nfts.Nfts, nft.Base)
		}
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
	sats := p.GetBoundSatsWithUtxo(utxoId)
	// 实际上是将上面这所有的聪disable了

	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, sat := range sats {
		p.disabledSats[sat] = true
		saveDisabledSatToDB(sat, proof, p.db)
	}
	return nil
}
