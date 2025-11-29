package common

import (
	"github.com/sat20-labs/indexer/common/pb"
)

type InscribeBaseContent = pb.InscribeBaseContent
type Nft struct {
	Base           *InscribeBaseContent
	OwnerAddressId uint64
	UtxoId         uint64
	Offset         int64
}

func (p *Nft) Clone() *Nft {
	return &Nft{
		Base:           p.Base, // 固定数据
		OwnerAddressId: p.OwnerAddressId,
		UtxoId:         p.UtxoId,
		Offset:         p.Offset,
	}
}

type NftStatus struct {
	Version          string
	Count            uint64
	Unbound          uint64 // 负数铭文数量，指没有绑定到聪上的铭文。能检索到，但无法转移。
	ContentCount     uint64 // content count
	ContentTypeCount int    // content type count
}

type NftsInSat = pb.NftsInSat

func (p *NftStatus) Clone() *NftStatus {
	c := *p
	return &c
}
