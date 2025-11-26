package base

import (
	"sort"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/rpcserver/utils"
	"github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type Model struct {
	indexer base_indexer.Indexer
}

func NewModel(i base_indexer.Indexer) *Model {
	return &Model{
		indexer: i,
	}
}


func (s *Model) getPlainUtxos(address string, value int64, start, limit int) ([]*wire.PlainUtxo, int, error) {
	utxomap, err := s.indexer.GetUTXOsWithAddress(address)
	if err != nil {
		return nil, 0, err
	}
	avaibableUtxoList := make([]*wire.PlainUtxo, 0)
	utxos := make([]*common.UtxoIdInDB, 0)
	for key, value := range utxomap {
		utxos = append(utxos, &common.UtxoIdInDB{UtxoId: key, Value: value})
	}

	// sort.Slice(utxos, func(i, j int) bool {
	// 	return utxos[i].Value > utxos[j].Value
	// })

	// // 分页显示
	totalRecords := len(utxos)
	// if totalRecords < start {
	// 	return nil, totalRecords, fmt.Errorf("start exceeds the count of UTXO")
	// }
	// if totalRecords < start+limit {
	// 	limit = totalRecords - start
	// }
	// end := start + limit
	// utxos = utxos[start:end]

	for _, utxoId := range utxos {
		//Indicates that this utxo has been spent and cannot be used for indexing
		utxo := s.indexer.GetUtxoById(utxoId.UtxoId)
		if utxo == "" {
			continue
		}

		if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxoId.UtxoId, false) {
			continue
		}

		if s.indexer.IsUtxoSpent(utxo) {
			continue
		}

		txid, vout, err := common.ParseUtxo(utxo)
		if err != nil {
			continue
		}

		height, index, _ := common.FromUtxoId(utxoId.UtxoId)
		//Find utxo with value
		if utxoId.Value >= value {
			avaibableUtxoList = append(avaibableUtxoList, &wire.PlainUtxo{
				Height: height,
				Index: index,
				Txid:  txid,
				Vout:  vout,
				Value: utxoId.Value,
			})
		}
	}

	sort.Slice(avaibableUtxoList, func(i, j int) bool {
		return avaibableUtxoList[i].Value > avaibableUtxoList[j].Value
	})

	return avaibableUtxoList, totalRecords, nil
}

func (s *Model) getAllUtxos(address string, start, limit int) ([]*wire.PlainUtxo, []*wire.PlainUtxo, int, error) {
	utxomap, err := s.indexer.GetUTXOsWithAddress(address)
	if err != nil {
		return nil, nil, 0, err
	}

	utxos := make([]*common.UtxoIdInDB, 0)
	for key, value := range utxomap {
		utxos = append(utxos, &common.UtxoIdInDB{UtxoId: key, Value: value})
	}

	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Value > utxos[j].Value
	})

	// // 分页显示
	totalRecords := len(utxos)
	// if totalRecords < start {
	// 	return nil, nil, totalRecords, fmt.Errorf("start exceeds the count of UTXO")
	// }
	// if totalRecords < start+limit {
	// 	limit = totalRecords - start
	// }
	// end := start + limit
	// utxos = utxos[start:end]

	plainUtxos := make([]*wire.PlainUtxo, 0)
	otherUtxos := make([]*wire.PlainUtxo, 0)

	for _, utxoId := range utxos {
		//Indicates that this utxo has been spent and cannot be used for indexing
		utxo := s.indexer.GetUtxoById(utxoId.UtxoId)
		if utxo == "" {
			continue
		}

		if utils.IsUtxoSpent(utxo) {
			continue
		}

		txid, vout, err := common.ParseUtxo(utxo)
		if err != nil {
			continue
		}

		height, index, _ := common.FromUtxoId(utxoId.UtxoId)
		//Find common utxo (that is, utxo with non-ordinal attributes)
		if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxoId.UtxoId, false) {
			otherUtxos = append(otherUtxos, &wire.PlainUtxo{
				Height: height,
				Index: index,
				Txid:  txid,
				Vout:  vout,
				Value: utxoId.Value,
			})
		} else {
			plainUtxos = append(plainUtxos, &wire.PlainUtxo{
				Height: height,
				Index: index,
				Txid:  txid,
				Vout:  vout,
				Value: utxoId.Value,
			})
		}

	}

	return plainUtxos, otherUtxos, totalRecords, nil
}


func (s *Model) GetSatInfo(sat int64) *wire.SatInfo {
	sm := exotic.Sat(sat)

	return &wire.SatInfo{
		Sat:        int64(sm),
		Height:     sm.Height(),
		Epoch:      int64(sm.Epoch()),
		Cycle:      int64(sm.Cycle()),
		Period:     int64(sm.Period()),
		Satributes: sm.Satributes(),
	}
}
