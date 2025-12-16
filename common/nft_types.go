package common

import (
	"github.com/sat20-labs/indexer/common/pb"
)

const Jubilee_Height int = 824544

type InscribeBaseContent = pb.InscribeBaseContent
type Nft struct {
	Base           *InscribeBaseContent
	OwnerAddressId uint64
	UtxoId         uint64
	Offset         int64 // 在对外接口时重新获取，内部不要用也不要重新
}

type NftStatus struct {
	Version          string
	// 所有铭文数量 = Count + CurseCount
	Count            uint64 // blessed，vindicated
	CurseCount       uint64 // cursed
	Unbound          uint64 // 负数铭文数量，指没有绑定到聪上的铭文。能检索到，但无法转移。
	ContentCount     uint64 // content count
	ContentTypeCount int    // content type count
}

type NftsInSat = pb.NftsInSat

func (p *NftStatus) Clone() *NftStatus {
	c := *p
	return &c
}
