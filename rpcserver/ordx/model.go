package ordx

import (
	"fmt"
	"sort"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	rpcwire "github.com/sat20-labs/indexer/rpcserver/wire"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type Model struct {
	indexer  base_indexer.Indexer
	nonceMap map[string]int64
	mutex    sync.RWMutex
}

func NewModel(indexer base_indexer.Indexer) *Model {
	return &Model{
		indexer:  indexer,
		nonceMap: make(map[string]int64),
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

func (s *Model) getBindingSatFromOrdxTicker(tickerName *common.AssetName) int {
	ticker := s.indexer.GetTickerInfo(tickerName)
	if ticker == nil {
		return 1
	}

	return ticker.N
}

func (s *Model) GetAssetSummaryV3(address string, start int, limit int) ([]*common.DisplayAsset, error) {
	tickerMap := s.indexer.GetAssetSummaryInAddressV3(address)

	result := make([]*common.DisplayAsset, 0)
	for tickName, balance := range tickerMap {
		resp := &common.DisplayAsset{}
		resp.AssetName = tickName
		resp.Amount = balance.String()
		resp.Precision = balance.Precision
		resp.BindingSat = s.indexer.GetBindingSat(&tickName)
		result = append(result, resp)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Amount > result[j].Amount
	})

	return result, nil
}

func (s *Model) GetUtxoInfoV3(utxo string) (*common.AssetsInUtxo, error) {
	if s.indexer.IsUtxoSpent(utxo) {
		return nil, fmt.Errorf("utxo %s is spent", utxo)
	}
	ret := s.indexer.GetTxOutputWithUtxoV3(utxo)
	if ret == nil {
		return nil, fmt.Errorf("can't find utxo %s", utxo)
	}
	return ret, nil
}

func (s *Model) GetUtxoInfoListV3(req *rpcwire.UtxosReq) ([]*common.AssetsInUtxo, error) {
	result := make([]*common.AssetsInUtxo, 0)
	for _, utxo := range req.Utxos {
		txOutput, err := s.GetUtxoInfoV3(utxo)
		if err != nil {
			continue
		}

		result = append(result, txOutput)
	}

	return result, nil
}

func (s *Model) UnlockOrdinals(req *rpcwire.UnlockOrdinalsReq) (map[string]error, error)  {
	return s.indexer.UnlockOrdinals(req.Utxos, req.PubKey, req.Sig)
}

func (s *Model) GetLockedUtxoInAddress(address string) ([]*common.AssetsInUtxo, error) {
	return s.indexer.GetLockedUTXOsInAddress(address)
}


func (s *Model) GetUtxosWithAssetNameV3(address, name string, start, limit int) ([]*common.AssetsInUtxo, int, error) {
	result := make([]*common.AssetsInUtxo, 0)
	assetName := common.NewAssetNameFromString(name)
	//t1 := time.Now()
	outputMap, err := s.indexer.GetAssetUTXOsInAddressWithTickV3(address, assetName)
	if err != nil {
		return nil, 0, err
	}
	//common.Log.Infof("GetAssetUTXOsInAddressWithTickV3 takes %v", time.Since(t1))
	//t1 = time.Now()
	for _, txOut := range outputMap {
		if s.indexer.IsUtxoSpent(txOut.OutPoint) {
			continue
		}
		result = append(result, txOut)
	}
	//common.Log.Infof("filtering takes %v", time.Since(t1))

	sort.Slice(result, func(i, j int) bool {
		return result[i].Value > result[j].Value
	})

	return result, len(result), nil
}

type TickHolders struct {
	LastTimestamp        int64
	Total                uint64
	HoldersAddressAmount []*rpcwire.HolderV3
}

const tickHoldersCacheDuration = 10 * time.Minute

var (
	runeHoldersCache cmap.ConcurrentMap[string, *TickHolders]
)

func init() {
	runeHoldersCache = cmap.New[*TickHolders]()
}

func (s *Model) GetHolderListV3(tickName string, start, limit uint64) ([]*rpcwire.HolderV3, uint64, error) {
	result := make([]*rpcwire.HolderV3, 0)
	needUpdate := false

	if runeHolders, exist := runeHoldersCache.Get(tickName); exist {
		if time.Since(time.Unix(runeHolders.LastTimestamp, 0)) < tickHoldersCacheDuration {
			result = runeHolders.HoldersAddressAmount
		} else {
			needUpdate = true
		}
	} else {
		needUpdate = true
	}

	if needUpdate {
		assetName := common.NewAssetNameFromString(tickName)
		holders := s.indexer.GetHoldersWithTickV2(assetName)

		result = make([]*rpcwire.HolderV3, 0, len(holders))
		for address, amt := range holders {
			ordxMintInfo := &rpcwire.HolderV3{
				Wallet:       s.indexer.GetAddressById(address),
				TotalBalance: amt.String(),
			}
			result = append(result, ordxMintInfo)
		}
		sort.Slice(result, func(i, j int) bool {
			a, _ := common.NewDecimalFromString(result[i].TotalBalance, 20)
			b, _ := common.NewDecimalFromString(result[j].TotalBalance, 20)
			return a.Cmp(b) > 0
		})

		total := uint64(len(result))
		runeHolders := &TickHolders{
			LastTimestamp:        time.Now().Unix(),
			Total:                total,
			HoldersAddressAmount: result,
		}
		runeHoldersCache.Set(tickName, runeHolders)
	}

	total := uint64(len(result))
	end := total
	if start >= end {
		return nil, 0, nil
	}
	if start+limit < end {
		end = start + limit
	}
	result = result[start:end]
	return result, total, nil
}

func (s *Model) GetMintHistoryV3(tickName string, start, limit int) (*rpcwire.MintHistoryV3, error) {
	assetName := common.NewAssetNameFromString(tickName)
	result := rpcwire.MintHistoryV3{Ticker: tickName}
	mintInfos := s.indexer.GetMintHistoryV2(assetName, start, limit)
	for _, mintInfo := range mintInfos {
		ordxMintInfo := &rpcwire.MintHistoryItemV3{
			MintAddress:    mintInfo.Address,
			HolderAddress:  s.indexer.GetHolderAddress(mintInfo.InscriptionId),
			Balance:        mintInfo.Amount,
			InscriptionID:  mintInfo.InscriptionId,
			InscriptionNum: mintInfo.InscriptionNum,
		}
		result.Items = append(result.Items, ordxMintInfo)
	}
	_, times := s.indexer.GetMintAmount(tickName)
	result.Total = int(times)

	return &result, nil
}
