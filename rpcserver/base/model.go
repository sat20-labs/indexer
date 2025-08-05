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

func (s *Model) GetSatRangeInUtxo(utxo string) (*wire.ExoticSatRangeUtxo, error) {
	_, utxoRanges, err := s.indexer.GetOrdinalsWithUtxo(utxo)
	if err != nil {
		common.Log.Errorf("GetOrdinalsForUTXO failed, %s", utxo)
		return nil, err
	}

	// Caluclate the offset for each range
	var satList []wire.SatDetailInfo
	sr := s.indexer.GetExoticsWithRanges(utxoRanges)
	for _, r := range sr {
		exoticSat := exotic.Sat(r.Range.Start)
		sat := wire.SatDetailInfo{
			SatributeRange: wire.SatributeRange{
				SatRange: wire.SatRange{
					Start:  r.Range.Start,
					Size:   r.Range.Size,
					Offset: r.Offset,
				},
				Satributes: r.Satributes,
			},
			Block: int(exoticSat.Height()),
			// Time:  0, //暂时不显示，需要获取Block的时间。
		}
		satList = append(satList, sat)
	}

	offset := int64(0)
	for _, r := range utxoRanges {
		exoticSat := exotic.Sat(r.Start)
		sat := wire.SatDetailInfo{
			SatributeRange: wire.SatributeRange{
				SatRange: wire.SatRange{
					Start:  r.Start,
					Size:   r.Size,
					Offset: offset,
				},
				Satributes: nil,
			},
			Block: int(exoticSat.Height()),
			// Time:  0, //暂时不显示，需要获取Block的时间。
		}
		offset += r.Size
		satList = append(satList, sat)
	}

	return &wire.ExoticSatRangeUtxo{
		Utxo:  utxo,
		Value: common.GetOrdinalsSize(utxoRanges),
		Sats:  satList,
	}, nil
}

func (s *Model) GetExoticUtxos(address string) ([]*wire.ExoticSatRangeUtxo, error) {
	utxoList, err := s.indexer.GetUTXOsWithAddress(address)
	if err != nil {
		common.Log.Errorf("GetUTXOs failed. %s", err)
		return nil, err
	}
	satributeSatList := make([]*wire.ExoticSatRangeUtxo, 0)
	for utxoId, value := range utxoList {
		utxo, res, err := s.indexer.GetOrdinalsWithUtxoId(utxoId)
		if err != nil {
			common.Log.Errorf("GetOrdinalsWithUtxoId failed, %d", utxoId)
			return nil, err
		}

		if s.indexer.HasAssetInUtxo(utxo, true) {
			//common.Log.Infof("HasAssetInUtxo return true %s", utxo)
			continue
		}

		// Caluclate the offset for each range
		var satList []wire.SatDetailInfo
		sr := s.indexer.GetExoticsWithRanges(res)
		for _, r := range sr {
			exoticSat := exotic.Sat(r.Range.Start)
			sat := wire.SatDetailInfo{
				SatributeRange: wire.SatributeRange{
					SatRange: wire.SatRange{
						Start:  r.Range.Start,
						Size:   r.Range.Size,
						Offset: r.Offset,
					},
					Satributes: r.Satributes,
				},
				Block: int(exoticSat.Height()),
				// Time:  0, //暂时不显示，需要获取Block的时间。
			}
			satList = append(satList, sat)
		}

		satributeSatList = append(satributeSatList, &wire.ExoticSatRangeUtxo{
			Utxo:  utxo,
			Value: value,
			Sats:  satList,
		})

	}

	sort.Slice(satributeSatList, func(i, j int) bool {
		return satributeSatList[i].Value > satributeSatList[j].Value
	})

	return satributeSatList, nil
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
		utxo, _, err := s.indexer.GetOrdinalsWithUtxoId(utxoId.UtxoId)
		if err != nil {
			continue
		}

		if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxo, false) {
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
		utxo, _, err := s.indexer.GetOrdinalsWithUtxoId(utxoId.UtxoId)
		if err != nil {
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
		if base_indexer.ShareBaseIndexer.HasAssetInUtxo(utxo, false) {
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

func (s *Model) GetExoticUtxosWithType(address string, typ string, amount int64) ([]*wire.SpecificExoticUtxo, error) {
	utxos, err := s.indexer.GetUTXOsWithAddress(address)
	if err != nil {
		return nil, err
	}
	utxoList := make([]*wire.SpecificExoticUtxo, 0)

	for utxoId, value := range utxos {

		if value < amount {
			continue
		}

		//Indicates that this utxo has been spent and cannot be used for indexing
		utxo, ranges, err := s.indexer.GetOrdinalsWithUtxoId(utxoId)
		if err != nil {
			common.Log.Errorf("GetOrdinalsForUTXO failed, %d", utxoId)
			continue
		}

		// //Find common utxo (that is, utxo with non-ordinal attributes)
		if s.indexer.HasAssetInUtxo(utxo, true) {
			//common.Log.Infof("HasAssetInUtxo return true %s", utxo)
			continue
		}

		exoticRanges := s.indexer.GetExoticsWithType(ranges, typ)

		total := int64(0)
		sats := make([]wire.SatRange, 0)
		for _, rng := range exoticRanges {
			total += rng.Range.Size
			sats = append(sats, wire.SatRange{Start: rng.Range.Start, Size: rng.Range.Size, Offset: rng.Offset})
		}

		if total < amount {
			continue
		}

		if utils.IsUtxoSpent(utxo) {
			common.Log.Infof("IsUtxoSpent return true %s", utxo)
			continue
		}

		utxoList = append(utxoList, &wire.SpecificExoticUtxo{
			Utxo:   utxo,
			Value:  value,
			Type:   typ,
			Amount: total,
			Sats:   sats,
		})
	}

	sort.Slice(utxoList, func(i, j int) bool {
		return utxoList[i].Amount > utxoList[j].Amount
	})

	return utxoList, nil
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

func (s *Model) findSatsInAddress(req *wire.SpecificSatReq) ([]*wire.SpecificSat, error) {

	utxos, err := s.indexer.GetUTXOsWithAddress(req.Address)
	if err != nil {
		return nil, err
	}
	utxoList := make([]*wire.SpecificSat, 0)

	for utxoId, value := range utxos {
		utxo, ranges, err := s.indexer.GetOrdinalsWithUtxoId(utxoId)
		if err != nil {
			common.Log.Errorf("GetOrdinalsForUTXO failed, %d", utxoId)
			continue
		}

		for _, sat := range req.Sats {
			if common.IsSatInRanges(sat, ranges) {
				offset := int64(0)
				sats := make([]wire.SatRange, 0)
				for _, rng := range ranges {
					sats = append(sats, wire.SatRange{Start: rng.Start, Size: rng.Size, Offset: offset})
					offset += rng.Size
				}

				utxoList = append(utxoList, &wire.SpecificSat{
					Address:     req.Address,
					Utxo:        utxo,
					Value:       value,
					SpecificSat: sat,
					Sats:        sats,
				})
			}
		}
	}

	return utxoList, nil

}

func (s *Model) findSat(sat int64) (*wire.SpecificSat, error) {
	address, utxo, err := s.indexer.FindSat(sat)
	if err != nil {
		return nil, err
	}

	_, ranges, err := s.indexer.GetOrdinalsWithUtxo(utxo)
	if err != nil {
		common.Log.Errorf("GetOrdinalsForUTXO failed, %s", utxo)
		return nil, err
	}

	var result *wire.SpecificSat
	if common.IsSatInRanges(sat, ranges) {
		offset := int64(0)
		sats := make([]wire.SatRange, 0)
		for _, rng := range ranges {
			sats = append(sats, wire.SatRange{Start: rng.Start, Size: rng.Size, Offset: offset})
			offset += rng.Size
		}

		result = &wire.SpecificSat{
			Address:     address,
			Utxo:        utxo,
			Value:       common.GetOrdinalsSize(ranges),
			SpecificSat: sat,
			Sats:        sats,
		}
	}

	return result, nil
}
