package ordx

import (
	"fmt"
	"sort"

	"github.com/sat20-labs/indexer/common"
	ordx "github.com/sat20-labs/indexer/common"
	serverOrdx "github.com/sat20-labs/indexer/server/define"
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

func (s *Model) newTickerStatusResp(ticker *ordx.Ticker) *serverOrdx.TickerStatus {
	txid, _, err := ordx.ParseOrdInscriptionID(ticker.Base.InscriptionId)
	if err != nil {
		ordx.Log.Warnf("ordx.Model.GetTickStatusList-> parse ticker utxo error: %s, ticker: %v", err.Error(), ticker)
		return nil
	}

	tickerStatusResp := &serverOrdx.TickerStatus{
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

func (s *Model) getTickStatusMap() (map[string]*serverOrdx.TickerStatus, error) {
	tickerMap, err := s.indexer.GetTickerMap()
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*serverOrdx.TickerStatus)
	for tickerName, ticker := range tickerMap {
		tickerStatusResp := s.newTickerStatusResp(ticker)
		ret[tickerName] = tickerStatusResp
	}
	return ret, nil
}

func (s *Model) getTicker(tickerName string) (*serverOrdx.TickerStatus, error) {
	ticker := s.indexer.GetTicker(tickerName)
	if ticker == nil {
		return nil, fmt.Errorf("can't find ticker %s", tickerName)
	}

	tickerStatusResp := s.newTickerStatusResp(ticker)

	return tickerStatusResp, nil
}


func IsAssetBindingSat(asset *swire.AssetName) uint16 {
	if asset.Protocol == common.PROTOCOL_NAME_ORD ||
		asset.Protocol == common.PROTOCOL_NAME_ORDX {
		return 1
	}
	return 0
}

func (s *Model) GetAssetSummary(address string, start int, limit int) (*AssetSummary, error) {
	tickerMap := s.indexer.GetAssetSummaryInAddress(address)

	result := AssetSummary{}
	for tickName, balance := range tickerMap {
		resp := &swire.AssetInfo{}
		resp.Name = tickName
		resp.Amount = balance
		resp.BindingSat = IsAssetBindingSat(&tickName)
		result.Data = append(result.Data, resp)
	}
	result.Start = 0
	result.Total = uint64(len(result.Data))

	sort.Slice(result.Data, func(i, j int) bool {
		return result.Data[i].Amount > result.Data[j].Amount
	})

	return &result, nil
}

func (s *Model) GetAssetWithUtxo(utxo string) (*TxOutput, error) {

	txOut := s.indexer.GetTxOutputWithUtxo(utxo)
	if txOut == nil {
		return nil, fmt.Errorf("can't get txout from %s", utxo)
	}

	output := TxOutput{
		OutPoint: utxo,
		OutValue: txOut.OutValue,
		Sats: txOut.Sats,
		Assets: txOut.Assets,
	}

	return &output, nil
}


func (s *Model) GetAssetsWithUtxos(req *UtxosReq) ([]*TxOutput, error) {
	result := make([]*TxOutput, 0)
	for _, utxo := range req.Utxos {
		
		txOutput, err := s.GetAssetWithUtxo(utxo)
		if err != nil {
			continue
		}

		result = append(result, txOutput)
	}

	return result, nil
}

func (s *Model) GetUtxosWithAssetName(address, name string, start, limit int) ([]*TxOutput, int, error) {
	

	return nil, 0, nil
}

