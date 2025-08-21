package exotic

import (
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	indexer "github.com/sat20-labs/indexer/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

type ExoticTickInfo struct {
	Id             uint64
	Name           string
	MintInfo       *indexer.RangeRBTree            // mint history: 用于查找某个SatRange是否存在该ticker， Value是RBTreeValue_Mint
	InscriptionMap map[string]*common.MintAbbrInfo // key: inscriptionId
	MintAdded      []*common.Mint
	Ticker         *common.Ticker
}

type ExoticIndexer struct {
	db          common.KVDB
	baseIndexer *base.BaseIndexer

	mutex sync.RWMutex // 只保护这几个结构

	// exotic sat range
	exoticTickerMap  map[string]*ExoticTickInfo // 用于检索稀有聪. key 稀有聪种类
	exoticSyncHeight int

	firstSatInBlock *indexer.SatRBTree // sat->height
}

var _instance *ExoticIndexer = nil

func getExoticIndexer() *ExoticIndexer {
	return _instance
}

func newExoticTickerInfo(name string) *ExoticTickInfo {
	return &ExoticTickInfo{
		Name:           name,
		MintInfo:       indexer.NewRBTress(),
		InscriptionMap: make(map[string]*common.MintAbbrInfo, 0),
		MintAdded:      make([]*common.Mint, 0),
	}
}

func NewExoticIndexer(baseIndexer *base.BaseIndexer) *ExoticIndexer {
	_instance = &ExoticIndexer{
		db:              baseIndexer.GetBaseDB(),
		baseIndexer:     baseIndexer,
		firstSatInBlock: indexer.NewSatRBTress(),
	}
	return _instance
}

func (p *ExoticIndexer) Init() {
	height := p.baseIndexer.GetSyncHeight()
	initEpochSat(p.db, height)
	p.newExoticTickerMap(height)
}

func (p *ExoticIndexer) Clone() *ExoticIndexer {
	return p
}

func (p *ExoticIndexer) Subtract(another *ExoticIndexer) {
}

func (p *ExoticIndexer) newExoticTickerMap(height int) {

	if p.exoticTickerMap != nil {
		return
	}
	common.Log.Info("newExoticTickerMap ...")

	exoticTickerMap := make(map[string]*ExoticTickInfo, 0)

	startTime := time.Now()
	exoticmap := p.loadExoticRanges(height)
	common.Log.Infof("LoadExoticRanges %d ms\n", time.Since(startTime).Milliseconds())
	for name, ranges := range exoticmap {
		tickinfo := newExoticTickerInfo(string(name))
		tickinfo.Ticker = p.getExoticDefaultTicker(string(name))

		for _, rng := range ranges {
			tickinfo.MintInfo.AddMintInfo(rng, string(name))
		}

		exoticTickerMap[string(name)] = tickinfo
	}
	p.exoticSyncHeight = height
	p.exoticTickerMap = exoticTickerMap

	common.Log.Infof("newExoticTicker %d ms\n", time.Since(startTime).Milliseconds())
}

func (p *ExoticIndexer) getExoticDefaultTicker(name string) *common.Ticker {
	ticker := &common.Ticker{
		Base: &common.InscribeBaseContent{
			Id:       0,
			TypeName: common.ASSET_TYPE_EXOTIC,

			BlockHeight:        0,
			InscriptionAddress: 0,
			BlockTime:          time.Now().Unix(),
			Content:            nil,
			ContentType:        nil,
			InscriptionId:      "",
		},

		Id:         -1,
		Name:       name,
		Type:       common.ASSET_TYPE_EXOTIC,
		Limit:      100000000,
		N:          1,
		SelfMint:   0,
		Max:        0,
		BlockStart: 0,
		BlockEnd:   0,
		Attr:       common.SatAttr{},
		Desc:       "Ordinals Rare Sats",
	}

	return ticker
}

func (p *ExoticIndexer) updateExoticTicker(height int) {
	if p.exoticTickerMap == nil {
		return
	}

	exoticmap := p.GetMoreExoticRangesToHeight(p.exoticSyncHeight+1, height)
	for name, ranges := range exoticmap {
		tickinfo := p.exoticTickerMap[string(name)]
		if tickinfo == nil {
			tickinfo = newExoticTickerInfo(string(name))
			tickinfo.Ticker = p.getExoticDefaultTicker(string(name))
			p.exoticTickerMap[string(name)] = tickinfo
		}

		for _, rng := range ranges {
			tickinfo.MintInfo.AddMintInfo(rng, string(name))
		}
	}
	// 不需要加holder
	p.exoticSyncHeight = height
}

func (p *ExoticIndexer) UpdateTransfer(block *common.Block) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	blockValue := p.getBlockInBuffer(block.Height)
	p.firstSatInBlock.Put(blockValue.Ordinals.Start, block.Height)
	if block.Height%210000 == 0 {
		epoch := int64(block.Height / 210000)
		SetEpochStartingAndChangeLast(epoch, blockValue.Ordinals.Start)
	}

	startTime := time.Now()
	p.updateExoticTicker(block.Height)
	common.Log.Infof("updateExoticTicker in %v", time.Since(startTime))
}

func (p *ExoticIndexer) GetMoreExoticRangesToHeight(startHeight, endHeight int) map[string][]*common.Range {
	if p.baseIndexer.GetHeight() < 0 {
		return nil
	}

	var result map[string][]*common.Range
	p.db.View(func(txn common.ReadBatch) error {
		result = p.getMoreRodarmorRarityRangesToHeight(startHeight, endHeight, txn)
		// TODO
		//result[Alpha] = p.GetRangesForAlpha(startHeight, endHeight)
		//result[Omega] = p.GetRangesForOmega(startHeight, endHeight)
		if endHeight >= 9 {
			result[Block9] = p.getRangeForBlock(9, txn)
		}
		if endHeight >= 78 {
			result[Block78] = p.getRangeForBlock(78, txn)
		}
		validBlock := make([]int, 0)
		for h := range NakamotoBlocks {
			if h <= endHeight {
				validBlock = append(validBlock, h)
			}
		}
		result[Nakamoto] = p.getRangesForBlocks(validBlock, txn)

		result[FirstTransaction] = FirstTransactionRanges
		if endHeight >= 1000 {
			result[Vintage] = p.getRangeToBlock(1000, txn)
		}
		return nil
	})

	return result
}

func initEpochSat(ldb common.KVDB, height int) {

	ldb.View(func(txn common.ReadBatch) error {
		currentEpoch := height / HalvingInterval
		underpays := int64(0)

		for epoch := (height / HalvingInterval); epoch > 0; epoch-- {

			value := &common.BlockValueInDB{}
			key := db.GetBlockDBKey(210000 * epoch)
			err := db.GetValueFromTxn(key, value, txn)
			if err != nil {
				common.Log.Panicf("GetValueFromDB %s failed. %v", key, err)
			}

			if epoch == currentEpoch {
				underpays = int64(Epoch(int64(epoch)).GetStartingSat()) - value.Ordinals.Start
			}
			SetEpochStartingSat(int64(epoch), value.Ordinals.Start)
		}

		for epoch := currentEpoch + 1; epoch < MAX_EPOCH; epoch++ {
			SetEpochStartingSat(int64(epoch), int64(Epoch(int64(epoch)).GetStartingSat())-underpays)
		}
		return nil
	})

}

// 跟base数据库同步
func (p *ExoticIndexer) UpdateDB() {
	//common.Log.Infof("NftIndexer->UpdateDB start...")
	startTime := time.Now()

	p.InitRarityDB(p.baseIndexer.GetSyncHeight())

	// reset memory buffer

	common.Log.Infof("ExoticIndexer->UpdateDB takes %v", time.Since(startTime))
}

func (p *ExoticIndexer) CheckSelf() bool {
	return true
}
