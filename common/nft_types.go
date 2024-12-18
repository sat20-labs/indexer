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
	Unbound uint64
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
