package indexer

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/ns"
)

func (b *IndexerMgr) GetNSStatus() *common.NameServiceStatus {
	return b.ns.GetStatus()
}

func (b *IndexerMgr) getNameInfoWithRegInfo(reg *ns.NameRegister) *common.NameInfo {
	address := b.GetAddressById(reg.Nft.OwnerAddressId)
	utxo := b.GetUtxoById(reg.Nft.UtxoId)
	kvs := make(map[string]*common.KeyValueInDB)
	attr := b.ns.GetNameProperties(reg)
	if attr != nil {
		for k, v := range attr.KVs {
			kvs[k] = &common.KeyValueInDB{Value: v.Value, InscriptionId: v.InscriptionId}
		}
	}

	return &common.NameInfo{
		Base:         reg.Nft.Base,
		Id:           reg.Id,
		Name:         reg.Name,
		OwnerAddress: address,
		Utxo:         utxo,
		KVs:          kvs,
	}
}

func (b *IndexerMgr) GetNameInfo(name string) *common.NameInfo {
	reg := b.ns.GetNameRegisterInfo(name)
	if reg == nil {
		common.Log.Errorf("GetNameRegisterInfo %s failed", name)
		return nil
	}

	return b.getNameInfoWithRegInfo(reg)
}

func (b *IndexerMgr) IsNameExist(name string) bool {
	return b.ns.IsNameExist(name)
}

func (b *IndexerMgr) GetNameWithInscriptionId(id string) *common.NameInfo {
	reg := b.ns.GetNameRegisterInfoWithInscriptionId(id)
	if reg == nil {
		common.Log.Errorf("GetNameWithInscriptionId %s failed", id)
		return nil
	}

	address := b.GetAddressById(reg.Nft.OwnerAddressId)
	utxo := b.GetUtxoById(reg.Nft.UtxoId)
	kvs := make(map[string]*common.KeyValueInDB)
	attr := b.ns.GetNameProperties(reg)
	if attr != nil {
		for k, v := range attr.KVs {
			kvs[k] = &common.KeyValueInDB{Value: v.Value, InscriptionId: v.InscriptionId}
		}
	}

	return &common.NameInfo{
		Base:         reg.Nft.Base,
		Id:           reg.Id,
		Name:         reg.Name,
		OwnerAddress: address,
		Utxo:         utxo,
		KVs:          kvs,
	}
}

func (b *IndexerMgr) GetNamesWithUtxo(utxoId uint64) []string {
	return b.ns.GetNamesWithUtxo2(utxoId)
}

func (b *IndexerMgr) GetNames(start, limit int) []string {
	return b.ns.GetNames(start, limit)
}

func (b *IndexerMgr) getNamesWithAddressInBuffer(address string) []*common.Nft {
	if b.addressToNameMap == nil {
		return b.initAddressToNameMap(address)
	}

	b.mutex.RLock()
	ret, ok := b.addressToNameMap[address]
	if !ok {
		b.mutex.RUnlock()
		ret = b.initAddressToNameMap(address)
	} else {
		b.mutex.RUnlock()
	}

	return ret
}

func (b *IndexerMgr) initAddressToNameMap(address string) []*common.Nft {
	nfts := b.getNftWithAddressInBuffer(address)
	names := make([]*common.Nft, 0)
	for _, nft := range nfts {
		if nft.Base.TypeName == common.ASSET_TYPE_NS {
			names = append(names, nft)
		}
	}

	b.mutex.Lock()
	if b.addressToNameMap == nil {
		b.addressToNameMap = make(map[string][]*common.Nft)
	}
	b.addressToNameMap[address] = names
	b.mutex.Unlock()
	return names
}

func (b *IndexerMgr) GetNamesWithAddress(address string, start, limit int) ([]*common.NameInfo, int) {
	nfts := b.getNamesWithAddressInBuffer(address)
	total := len(nfts)
	if start >= total {
		return nil, total
	}
	end := total
	if limit > 0 && start+limit < total {
		end = start + limit
	}

	result := make([]*common.NameInfo, 0)
	rngs := nfts[start:end]
	for _, nft := range rngs {
		info := b.GetNameWithInscriptionId(nft.Base.InscriptionId)
		if info != nil {
			result = append(result, info)
		}
	}

	return result, total
}

func (b *IndexerMgr) GetNameAmountWithAddress(address string) int {
	inscrptions := b.getNamesWithAddressInBuffer(address)
	return len(inscrptions)
}

func getSubName(name string) (string, string) {
	parts := strings.Split(name, ".")
	l := len(parts)
	if l == 1 {
		return parts[0], ""
	} else if l == 2 {
		return parts[0], parts[1]
	} else {
		return "", ""
	}
}

func (b *IndexerMgr) GetSubNamesWithAddress(address, sub string, start, limit int) ([]*common.NameInfo, int) {
	nfts := b.getNamesWithAddressInBuffer(address)

	subSet := make([]*common.Nft, 0)
	for _, nft := range nfts {
		name := string(nft.Base.UserData)
		_, subName := getSubName(name)
		if subName == sub {
			subSet = append(subSet, nft)
		}
	}

	total := len(subSet)
	if start >= total {
		return nil, total
	}
	end := total
	if limit > 0 && start+limit < total {
		end = start + limit
	}

	result := make([]*common.NameInfo, 0)
	rngs := subSet[start:end]
	for _, nft := range rngs {
		info := b.GetNameWithInscriptionId(nft.Base.InscriptionId)
		if info != nil {
			result = append(result, info)
		}
	}

	return result, total
}

func (b *IndexerMgr) GetSubNamesWithFilters(address, sub, filters string, start, limit int) ([]*common.NameInfo, int) {
	nfts := b.getNamesWithAddressInBuffer(address)

	filterv := strings.Split(filters, "+")

	subSet := make([]*common.Nft, 0)
	for _, nft := range nfts {
		name := string(nft.Base.UserData)
		prefix, subName := getSubName(name)
		if subName == sub && filterName(prefix, filterv) {
			subSet = append(subSet, nft)
		}
	}

	total := len(subSet)
	if start >= total {
		return nil, total
	}
	end := total
	if limit > 0 && start+limit < total {
		end = start + limit
	}

	result := make([]*common.NameInfo, 0)
	rngs := subSet[start:end]
	for _, nft := range rngs {
		info := b.GetNameWithInscriptionId(nft.Base.InscriptionId)
		if info != nil {
			result = append(result, info)
		}
	}

	return result, total
}


func (b *IndexerMgr) GetNamesWithKey(address, key string, start, limit int) ([]*common.NameInfo, int) {
	nfts := b.getNamesWithAddressInBuffer(address)

	subSet := make([]*common.NameInfo, 0)
	for _, nft := range nfts {
		// name := string(nft.Base.UserData)
		// _, subName := getSubName(name)
		// if sub != "" && subName != sub {
		// 	continue
		// }
		info := b.GetNameWithInscriptionId(nft.Base.InscriptionId)
		if info == nil {
			continue
		}
		if _, ok := info.KVs[key]; ok {
			subSet = append(subSet, info)
		}
	}

	total := len(subSet)
	if start >= total {
		return nil, total
	}
	end := total
	if limit > 0 && start+limit < total {
		end = start + limit
	}

	return subSet[start:end], total
}

func (b *IndexerMgr) GetSubNameAmountWithAddress(address, sub string) int {
	nfts := b.getNamesWithAddressInBuffer(address)

	subSet := make([]*common.Nft, 0)
	for _, nft := range nfts {
		name := string(nft.Base.UserData)
		_, subName := getSubName(name)
		if subName == sub {
			subSet = append(subSet, nft)
		}
	}
	return len(subSet)
}

func (b *IndexerMgr) GetSubNameSummaryWithAddress(address string) map[string]int64 {
	nfts := b.getNamesWithAddressInBuffer(address)

	result := make(map[string]int64)
	for _, nft := range nfts {
		name := string(nft.Base.UserData)
		_, subName := getSubName(name)
		result[subName] += 1
	}
	return result
}

func (b *IndexerMgr) GetNamesWithSat(sat int64) []*common.NameInfo {
	result := make([]*common.NameInfo, 0)

	names := b.ns.GetNameRegisterInfoWithSat(sat)
	for _, name := range names {
		info := b.getNameInfoWithRegInfo(name)
		if info != nil {
			result = append(result, info)
		}
	}

	return result
}

func (b *IndexerMgr) getNamesWithUtxo(utxoId uint64) map[string]map[string][]*common.Range {
	result := make(map[string]map[string][]*common.Range)
	names := b.ns.GetNamesWithUtxo(utxoId)
	for _, name := range names {
		mintInfo := make(map[string][]*common.Range)
		mintInfo[name.Nft.Base.InscriptionId] = []*common.Range{{Start: name.Nft.Base.Sat, Size: 1}}
		result[name.Name] = mintInfo
	}
	return result
}

func (b *IndexerMgr) getNamesWithRanges(ranges []*common.Range) map[string][]*common.Range {
	result := make(map[string][]*common.Range)
	sats := b.ns.GetNamesWithRanges(ranges)
	for _, sat := range sats {
		infos := b.GetNamesWithSat(sat)
		for _, info := range infos {
			result[info.Base.InscriptionId] = []*common.Range{{Start: info.Base.Sat, Size: 1}}
		}
	}
	return result
}

func (p *IndexerMgr) GetNameHistory(start int, limit int) []*common.MintAbbrInfo {
	result := make([]*common.MintAbbrInfo, 0)
	names := p.ns.GetNames(start, limit)
	for _, name := range names {
		reg := p.ns.GetNameRegisterInfo(name)
		if reg != nil {
			info := common.NewMintAbbrInfo2(reg.Nft.Base)
			result = append(result, info)
		}
	}
	return result
}

func (p *IndexerMgr) GetNameHistoryWithAddress(addressId uint64, start int, limit int) ([]*common.MintAbbrInfo, int) {
	result := make([]*common.MintAbbrInfo, 0)
	nfts, total := p.ns.GetNamesWithInscriptionAddress(addressId, start, limit)
	for _, nft := range nfts {
		info := common.NewMintAbbrInfo2(nft.Base)
		result = append(result, info)
	}
	return result, total
}
