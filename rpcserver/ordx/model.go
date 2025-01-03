package ordx

import (
	"fmt"
	"sort"

	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/base_indexer"
	swire "github.com/sat20-labs/satsnet_btcd/wire"
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

func (s *Model) getTickerInfo(tickerName string) (*rpcwire.TickerInfo, error) {
	ticker := s.indexer.GetTickerInfo(swire.NewAssetNameFromString(tickerName))
	if ticker == nil {
		return nil, fmt.Errorf("can't find ticker %s", tickerName)
	}

	return &rpcwire.TickerInfo{
		Protocol: ticker.Protocol,
		Ticker: ticker.Name,
		Divisibility: ticker.Divisibility,
		TotalMinted: ticker.MintedAmt,
		MaxSupply: ticker.MaxSupply,
	}, nil
}

func (s *Model) GetAssetSummary(address string, start int, limit int) (*rpcwire.AssetSummary, error) {
	tickerMap := s.indexer.GetAssetSummaryInAddress(address)

	result := rpcwire.AssetSummary{}
	for tickName, balance := range tickerMap {
		resp := &swire.AssetInfo{}
		resp.Name = tickName
		resp.Amount = balance
		resp.BindingSat = common.IsBindingSat(&tickName)
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

	assets := make([]*rpcwire.AssetInfo, 0)
	for _, asset := range txOut.Assets {
		offsets := txOut.Offsets[asset.Name]

		info := rpcwire.AssetInfo{
			Asset:   asset,
			Offsets: offsets,
		}
		assets = append(assets, &info)
	}

	output := rpcwire.TxOutputInfo{
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
	assetName := swire.NewAssetNameFromString(name)
	outputMap, err := s.indexer.GetAssetUTXOsInAddressWithTickV2(address, assetName)
	if err != nil {
		return nil, 0, err
	}
	for _, txOut := range outputMap {
		if rpcwire.IsExistUtxoInMemPool(txOut.OutPointStr) {
			continue
		}
		assets := make([]*rpcwire.AssetInfo, 0)
		for _, asset := range txOut.Assets {
			offsets := txOut.Offsets[asset.Name]

			info := rpcwire.AssetInfo{
				Asset:   asset,
				Offsets: offsets,
			}
			assets = append(assets, &info)
		}

		output := rpcwire.TxOutputInfo{
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
