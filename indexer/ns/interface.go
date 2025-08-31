package ns

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func (p *NameService) HasNameInSat(sat int64) bool {

	nftInSat := p.nftIndexer.GetNftsWithSat(sat)
	for _, nft := range nftInSat.Nfts {
		if nft.TypeName == common.ASSET_TYPE_NS {
			return true
		}
	}
	return false
}


// fast
func (p *NameService) HasNamesInUtxo(utxoId uint64) bool {
	nfts := p.nftIndexer.GetNftsWithUtxo(utxoId)
	for _, nft := range nfts {
		switch nft.Base.TypeName {
		case common.ASSET_TYPE_NS:
			name := string(nft.Base.UserData)
			reg := p.GetNameRegisterInfo(name)
			if reg != nil {
				return true
			}
		}
	}
	return false
}

// fast
func (p *NameService) GetNamesWithUtxo(utxoId uint64) []*NameRegister {
	result := make([]*NameRegister, 0)
	nfts := p.nftIndexer.GetNftsWithUtxo(utxoId)
	for _, nft := range nfts {
		switch nft.Base.TypeName {
		case common.ASSET_TYPE_NS:
			name := string(nft.Base.UserData)
			reg := p.GetNameRegisterInfo(name)
			if reg != nil {
				result = append(result, reg)
			}
		}
	}
	return result
}

func (p *NameService) GetNamesWithUtxo2(utxoId uint64) []string {
	result := make([]string, 0)
	nfts := p.GetNamesWithUtxo(utxoId)
	for _, nft := range nfts {
		result = append(result, nft.Name)
	}
	return result
}

func (p *NameService) GetNamesWithRanges(rngs []*common.Range) []int64 {

	sats := p.nftIndexer.GetNftsWithRanges(rngs)
	result := make([]int64, 0)
	for _, sat := range sats {
		if p.HasNameInSat(sat) {
			result = append(result, sat)
		}
	}

	return result
}

func (p *NameService) GetNameRegisterInfoWithInscriptionId(inscId string) *NameRegister {
	nft := p.nftIndexer.GetNftWithInscriptionId(inscId)
	if nft == nil || nft.Base.TypeName != common.ASSET_TYPE_NS {
		return nil
	}
	name := string(nft.Base.UserData)

	return p.GetNameRegisterInfo(name)
}

func (p *NameService) GetNameRegisterInfoWithSat(sat int64) []*NameRegister {
	name := ""
	nftInSat := p.nftIndexer.GetNftsWithSat(sat)
	if nftInSat == nil {
		return nil
	}
	result := make([]*NameRegister, 0)
	for _, nft := range nftInSat.Nfts {
		if nft.TypeName == common.ASSET_TYPE_NS {
			name = string(nft.UserData)
			reg := p.GetNameRegisterInfo(name)
			if reg != nil {
				result = append(result, reg)
			}
		}
	}

	return result
}

func (p *NameService) IsNameExist(name string) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	name = strings.ToLower(name)
	reg := p.getNameInBuffer(name)
	if reg != nil {
		return true
	}

	value := NameValueInDB{}
	err := loadNameFromDB(name, &value, p.db)
	return err == nil
}

func (p *NameService) GetNameRegisterInfo(name string) *NameRegister {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	name = strings.ToLower(name)
	reg := p.getNameInBuffer(name)
	if reg != nil {
		// nft 可能已经被转移了，更新属性
		nft := p.nftIndexer.GetNftWithId(reg.Nft.Base.Id)
		reg.Nft = nft
		return reg
	}

	value := NameValueInDB{}
	err := loadNameFromDB(name, &value, p.db)
	if err != nil {
		return nil
	}

	nft := p.nftIndexer.GetNftWithId(value.NftId)
	if nft == nil {
		common.Log.Errorf("GetNftWithId %d failed.", value.NftId)
		return nil
	}

	reg = &NameRegister{Nft: nft, Id: value.Id, Name: value.Name}

	return reg
}

func (p *NameService) GetStatus() *common.NameServiceStatus {
	r := *p.status
	return &r
}

// 按照铸造时间
func (p *NameService) GetNames(start, limit int) []string {
	result := make([]string, 0)
	buckDB := NewBuckStore(p.db)
	end := start + limit
	namemap := buckDB.BatchGet(start, end)
	for _, reg := range p.nameAdded {
		namemap[int(reg.Id)] = &BuckValue{Name: reg.Name, Sat: reg.Nft.Base.Sat}
	}

	for i := start; i < end; i++ {
		value, ok := namemap[i]
		if ok {
			result = append(result, value.Name)
		}
	}

	return result
}

// 按照铸造时间
func (p *NameService) GetNamesWithInscriptionAddress(addressId uint64, start, limit int) ([]*common.Nft, int) {

	nftIds := p.nftIndexer.GetAllNftsWithInscriptionAddress(addressId)

	result := make([]*common.Nft, 0)
	// 过滤不是name的nft
	for _, id := range nftIds {
		nft := p.nftIndexer.GetNftWithId(id)
		if nft != nil && nft.Base.TypeName == common.ASSET_TYPE_NS {
			result = append(result, nft)
		}
	}

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

func (p *NameService) GetNameProperties(reg *NameRegister) *NameProperties {
	if reg == nil {
		return nil
	}

	kvs := loadNameProperties(reg.Name, p.db)

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, update := range p.updateAdded {
		if update.Name == reg.Name {
			for _, kv := range update.KVs {
				kvs[kv.Key] = &common.KeyValueInDB{Value: kv.Value, InscriptionId: update.InscriptionId}
			}
		}
	}

	return &NameProperties{
		NameRegister: *reg,
		KVs:          kvs,
	}
}

func (p *NameService) GetValueWithKey(name, key string) *common.KeyValueInDB {
	name = strings.ToLower(name)
	kv := loadValueWithKey(name, key, p.db)

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	for _, update := range p.updateAdded {
		if update.Name == name {
			for _, v := range update.KVs {
				if v.Key == key {
					kv = &common.KeyValueInDB{Value: v.Value, InscriptionId: update.InscriptionId}
				}
			}
		}
	}

	return kv
}
