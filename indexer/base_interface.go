package indexer

import (
	"github.com/sat20-labs/indexer/common"
)

///// rpc interface, run in mul-thread

// address, utxo, message
func (p *IndexerMgr) FindSat(sat int64) (string, string, error) {
	return p.rpcService.FindSat(sat)
}

func (p *IndexerMgr) GetOrdinalsWithUtxo(utxo string) (uint64, []*common.Range, error) {
	return p.rpcService.GetOrdinalsWithUtxo(utxo)
}

func (p *IndexerMgr) GetOrdinalsWithUtxoId(id uint64) (string, []*common.Range, error) {
	return p.rpcService.GetOrdinalsWithUtxoId(id)
}

func (p *IndexerMgr) GetUTXOsWithAddress(address string) (map[uint64]int64, error) {
	return p.rpcService.GetUTXOs(address)
}

func (p *IndexerMgr) GetSyncHeight() int {
	return p.rpcService.GetHeight()
}

func (p *IndexerMgr) GetChainTip() int {
	return p.compiling.GetChainTip()
}

func (p *IndexerMgr) GetBlockInfo(height int) (*common.BlockInfo, error) {
	return p.rpcService.GetBlockInfo(height)
}

func (p *IndexerMgr) GetHolderAddress(inscriptionId string) string {
	nft := p.nft.GetNftWithInscriptionId(inscriptionId)
	if nft != nil {
		address, err := p.rpcService.GetAddressByID(nft.OwnerAddressId)
		if err == nil {
			return address
		}
	}
	return ""
}

func (p *IndexerMgr) GetAddressById(id uint64) string {
	address, _ := p.rpcService.GetAddressByID(id)
	return address
}

func (p *IndexerMgr) GetAddressId(address string) uint64 {
	return p.rpcService.GetAddressId(address)
}

func (p *IndexerMgr) GetUtxoById(id uint64) string {
	str, _ := p.rpcService.GetUtxoByID(id)
	return str
}

func (p *IndexerMgr) GetUtxoId(utxo string) uint64 {
	id, _, _ := p.rpcService.GetOrdinalsWithUtxo(utxo)
	return id
}
