package common

import (
	"github.com/sat20-labs/indexer/common/pb"
)

type InscribeBaseContent = pb.InscribeBaseContent
type Nft struct {
	Base           *InscribeBaseContent
	OwnerAddressId uint64
	UtxoId         uint64
}

type NftStatus struct {
	Version string
	Count   uint64
	Unbound uint64 // 负数铭文数量，指没有绑定到聪上的铭文。能检索到，但无法转移。
}

type NftsInSat = pb.NftsInSat


func (p *NftStatus) Clone() *NftStatus {
	c := &NftStatus{
		Version: p.Version,
		Count:   p.Count,
		Unbound: p.Unbound,
	}
	return c
}
