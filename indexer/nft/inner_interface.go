package nft

import "github.com/sat20-labs/indexer/common"

/*
 提供一些内部接口，在跑数据时供内部模块快速访问。
 只能在跑数据的线程中调用。
*/

func (p *NftIndexer) GetNftsWithSatNoDisable(sat int64) *common.NftsInSat {
	return p.getNftsWithSat(sat)
}


func (p *NftIndexer) GetNftWithIdWithNoLock(id int64) *common.Nft {
	return p.getNftWithId(id)
}


func (p *NftIndexer) GetNftWithInscriptionIdWithNoLock(inscriptionId string) *common.Nft {
	return p.getNftWithInscriptionId(inscriptionId)
}