package ordx

import (
	"fmt"
	"sort"

	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type Model struct {
	indexer base_indexer.Indexer
}

func NewModel(indexer base_indexer.Indexer) *Model {
	return &Model{
		indexer: indexer,
	}
}

func (s *Model) newTickerStatusResp(ticker *common.Ticker) *rpcwire.TickerStatus {
	txid, _, err := common.ParseOrdInscriptionID(ticker.Base.InscriptionId)
	if err != nil {
		common.Log.Warnf("ordx.Model.GetTickStatusList-> parse ticker utxo error: %s, ticker: %v", err.Error(), ticker)
		return nil
	}

	tickerStatusResp := &rpcwire.TickerStatus{
		Ticker:          ticker.Name,
		ID:              (ticker.Id),
		InscriptionId:   ticker.Base.InscriptionId,
		Limit:           ticker.Limit,
		N:               ticker.N,
		SelfMint:        ticker.SelfMint,
		Max:             ticker.Max, // 无效：< 0
		StartBlock:      ticker.BlockStart,
		EndBlock:        ticker.BlockEnd,
		Rarity:          ticker.Attr.Rarity,
		Description:     ticker.Desc,
		DeployBlocktime: ticker.Base.BlockTime,
		DeployHeight:    int(ticker.Base.BlockHeight),
		DeployAddress:   s.indexer.GetAddressById(ticker.Base.InscriptionAddress),
		InscriptionNum:  ticker.Base.Id,
		Content:         ticker.Base.Content,
		ContentType:     string(ticker.Base.ContentType),
		Delegate:        ticker.Base.Delegate,
		TxId:            txid,
		HoldersCount:    s.indexer.GetHolderAmountWithTick(ticker.Name),
	}

	tickerStatusResp.TotalMinted, tickerStatusResp.MintTimes = s.indexer.GetMintAmount(ticker.Name)

	return tickerStatusResp
}

func (s *Model) getTickStatusMap() (map[string]*rpcwire.TickerStatus, error) {
	tickerMap, err := s.indexer.GetTickerMap()
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*rpcwire.TickerStatus)
	for tickerName, ticker := range tickerMap {
		tickerStatusResp := s.newTickerStatusResp(ticker)
		ret[tickerName] = tickerStatusResp
	}
	return ret, nil
}

func (s *Model) getMintableTickStatusMap(protocol string) (map[string]*rpcwire.TickerStatus, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *Model) getTicker(tickerName string) (*rpcwire.TickerStatus, error) {
	ticker := s.indexer.GetTicker(tickerName)
	if ticker == nil {
		return nil, fmt.Errorf("can't find ticker %s", tickerName)
	}

	tickerStatusResp := s.newTickerStatusResp(ticker)

	return tickerStatusResp, nil
}

func (s *Model) getTickerInfo(tickerName string) (*common.TickerInfo, error) {
	ticker := s.indexer.GetTickerInfo(common.NewAssetNameFromString(tickerName))
	if ticker == nil {
		return nil, fmt.Errorf("can't find ticker %s", tickerName)
	}

	return ticker, nil
}

func (s *Model) GetAssetSummary(address string, start int, limit int) (*rpcwire.AssetSummary, error) {
	tickerMap := s.indexer.GetAssetSummaryInAddressV2(address)

	result := rpcwire.AssetSummary{}
	for tickName, balance := range tickerMap {
		resp := &common.AssetInfo{}
		resp.Name = tickName
		resp.Amount = balance
		resp.BindingSat = uint32(s.indexer.GetBindingSat(&tickName))
		result.Data = append(result.Data, resp)
	}
	result.Start = 0
	result.Total = uint64(len(result.Data))

	sort.Slice(result.Data, func(i, j int) bool {
		return result.Data[i].Amount > result.Data[j].Amount
	})

	return &result, nil
}

func (s *Model) GetUtxoInfo(utxo string) (*rpcwire.TxOutputInfo, error) {

	txOut := s.indexer.GetTxOutputWithUtxo(utxo)
	if txOut == nil {
		return nil, fmt.Errorf("can't get txout from %s", utxo)
	}

	assets := make([]*rpcwire.UtxoAssetInfo, 0)
	for _, asset := range txOut.Assets {
		offsets := txOut.Offsets[asset.Name]

		info := rpcwire.UtxoAssetInfo{
			Asset:   asset,
			Offsets: offsets,
		}
		assets = append(assets, &info)
	}

	output := rpcwire.TxOutputInfo{
		UtxoId:    txOut.UtxoId,
		OutPoint:  utxo,
		OutValue:  txOut.OutValue,
		AssetInfo: assets,
	}

	return &output, nil
}

func (s *Model) GetUtxoInfoList(req *rpcwire.UtxosReq) ([]*rpcwire.TxOutputInfo, error) {
	result := make([]*rpcwire.TxOutputInfo, 0)
	for _, utxo := range req.Utxos {
		if rpcwire.IsExistUtxoInMemPool(utxo) {
			continue
		}
		txOutput, err := s.GetUtxoInfo(utxo)
		if err != nil {
			continue
		}

		result = append(result, txOutput)
	}

	return result, nil
}

func (s *Model) GetUtxosWithAssetName(address, name string, start, limit int) ([]*rpcwire.TxOutputInfo, int, error) {
	result := make([]*rpcwire.TxOutputInfo, 0)
	assetName := common.NewAssetNameFromString(name)
	outputMap, err := s.indexer.GetAssetUTXOsInAddressWithTickV2(address, assetName)
	if err != nil {
		return nil, 0, err
	}
	for _, txOut := range outputMap {
		if rpcwire.IsExistUtxoInMemPool(txOut.OutPointStr) {
			continue
		}
		assets := make([]*rpcwire.UtxoAssetInfo, 0)
		for _, asset := range txOut.Assets {
			offsets := txOut.Offsets[asset.Name]

			info := rpcwire.UtxoAssetInfo{
				Asset:   asset,
				Offsets: offsets,
			}
			assets = append(assets, &info)
		}

		output := rpcwire.TxOutputInfo{
			UtxoId:    txOut.UtxoId,
			OutPoint:  txOut.OutPointStr,
			OutValue:  txOut.OutValue,
			AssetInfo: assets,
		}

		result = append(result, &output)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].OutValue.Value > result[j].OutValue.Value
	})

	return result, len(result), nil
}

func (s *Model) GetAssetSummaryV3(address string, start int, limit int) ([]*common.DisplayAsset, error) {
	tickerMap := s.indexer.GetAssetSummaryInAddressV3(address)

	result := make([]*common.DisplayAsset, 0)
	for tickName, balance := range tickerMap {
		resp := &common.DisplayAsset{}
		resp.AssetName = tickName
		resp.Amount = balance.String()
		resp.BindingSat = s.indexer.GetBindingSat(&tickName)
		result = append(result, resp)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Amount > result[j].Amount
	})

	return result, nil
}

func (s *Model) GetUtxoInfoV3(utxo string) (*common.AssetsInUtxo, error) {
	return s.indexer.GetTxOutputWithUtxoV3(utxo), nil
}

func (s *Model) GetUtxoInfoListV3(req *rpcwire.UtxosReq) ([]*common.AssetsInUtxo, error) {
	result := make([]*common.AssetsInUtxo, 0)
	for _, utxo := range req.Utxos {
		if rpcwire.IsExistUtxoInMemPool(utxo) {
			continue
		}
		txOutput, err := s.GetUtxoInfoV3(utxo)
		if err != nil {
			continue
		}

		result = append(result, txOutput)
	}

	return result, nil
}

func (s *Model) GetUtxosWithAssetNameV3(address, name string, start, limit int) ([]*common.AssetsInUtxo, int, error) {
	result := make([]*common.AssetsInUtxo, 0)
	assetName := common.NewAssetNameFromString(name)
	outputMap, err := s.indexer.GetAssetUTXOsInAddressWithTickV3(address, assetName)
	if err != nil {
		return nil, 0, err
	}
	for _, txOut := range outputMap {
		result = append(result, txOut)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Value > result[j].Value
	})

	return result, len(result), nil
}
