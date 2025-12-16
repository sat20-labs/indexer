package nft

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/base"
	"github.com/sat20-labs/indexer/indexer/db"

	ordCommon "github.com/sat20-labs/indexer/indexer/ord/common"
)

type SatInfo struct {
	AddressId uint64
	UtxoId    uint64
	Offset    int64
	CurseCount int
	Nfts      map[int64]bool // nftId
}

func (p *SatInfo) ToNftsInSat(sat int64) *common.NftsInSat {
	nfts := &common.NftsInSat{
		Sat: sat,
		OwnerAddressId: p.AddressId,
		UtxoId: p.UtxoId,
		Offset: p.Offset,
		CurseCount: int32(p.CurseCount),
	}
	for k := range p.Nfts {
		nfts.Nfts = append(nfts.Nfts, k)
	}
	sort.Slice(nfts.Nfts, func(i, j int) bool {
		return nfts.Nfts[i] < nfts.Nfts[j]
	})
	return nfts
}

// 所有nft的记录
// 以后ns和ordx模块，数据变大，导致加载、跑数据等太慢，需要按照这个模块的方式来修改优化。
type NftIndexer struct {
	db           common.KVDB
	status       *common.NftStatus
	enableHeight int
	disabledSats map[int64]bool // 所有disabled的satoshi

	baseIndexer *base.BaseIndexer
	mutex       sync.RWMutex

	// realtime buffer, utxoMap和satMap必须保持一致，utxo包含的聪，必须在satMap
	utxoMap           map[uint64][]*SatOffset // utxo->sats  确保utxo中包含的所有nft都列在这里
	satMap            map[int64]*SatInfo      // key: sat, 一个写入周期中新增加的铭文的转移结果，该sat绑定的nft都在这里
	contentMap        map[uint64]string       // contentId -> content
	contentToIdMap    map[string]uint64       //
	addedContentIdMap map[uint64]bool
	inscriptionToNftIdMap map[string]int64        // inscriptionId->nftId
	nftIdToinscriptionMap map[int64]string        // nftId->inscriptionId

	// 暂时不需要清理
	contentTypeMap     map[int]string // ctId -> content type
	contentTypeToIdMap map[string]int //
	lastContentTypeId  int

	// 状态变迁
	nftAdded        []*common.Nft // 保持顺序
	utxoDeled       []uint64

	// 不需要备份的数据
	nftBuffer       []*common.Nft // 一个区块内的缓存
	nftAddedUtxoMap map[uint64][]*common.Nft // 一个区块中，增量的nft在哪个输出的utxo中 utxoId->nftId->nft
}

func NewNftIndexer(db common.KVDB) *NftIndexer {
	enableHeight := 767430
	if !common.IsMainnet() {
		enableHeight = 27228
	}
	ns := &NftIndexer{
		db:        db,
		enableHeight: enableHeight,
		status:    nil,
		utxoMap:   nil,
		satMap:    nil,
		utxoDeled: nil,
	}
	ns.reset()
	return ns
}

// 只能被调用一次
func (p *NftIndexer) Init(baseIndexer *base.BaseIndexer) {
	p.baseIndexer = baseIndexer
	p.status = initStatusFromDB(p.db)
	p.disabledSats = loadAllDisalbedSatsFromDB(p.db)

	p.contentMap = make(map[uint64]string)
	p.contentToIdMap = make(map[string]uint64)
	p.addedContentIdMap = make(map[uint64]bool)
	p.inscriptionToNftIdMap = make(map[string]int64)
	p.nftIdToinscriptionMap = make(map[int64]string)

	p.contentTypeMap = getContentTypesFromDB(p.db)
	p.contentTypeToIdMap = make(map[string]int)
	for k, v := range p.contentTypeMap {
		p.contentTypeToIdMap[v] = k
	}
	p.lastContentTypeId = p.status.ContentTypeCount
}

func (p *NftIndexer) reset() {
	//p.disabledSats = make(map[int64]bool)
	p.utxoMap = make(map[uint64][]*SatOffset)
	p.satMap = make(map[int64]*SatInfo)
	p.nftAdded = make([]*common.Nft, 0)
	p.nftAddedUtxoMap = make(map[uint64][]*common.Nft)
	p.utxoDeled = make([]uint64, 0)
}

func (p *NftIndexer) Clone() *NftIndexer {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	newInst := NewNftIndexer(p.db)
	newInst.baseIndexer = p.baseIndexer

	newInst.disabledSats = p.disabledSats // 仅在rpc中使用
	newInst.utxoMap = make(map[uint64][]*SatOffset)
	for k, v := range p.utxoMap {
		nv := make([]*SatOffset, len(v))
		for i, s := range v {
			nv[i] = &SatOffset{
				Sat:    s.Sat,
				Offset: s.Offset,
			}
		}
		newInst.utxoMap[k] = nv
	}
	newInst.satMap = make(map[int64]*SatInfo)
	for k, v := range p.satMap {
		newV := &SatInfo{
			AddressId: v.AddressId,
			UtxoId:    v.UtxoId,
			Offset:    v.Offset,
			CurseCount: v.CurseCount,
			Nfts:      make(map[int64]bool),
		}
		for nftId := range v.Nfts {
			newV.Nfts[nftId] = true
		}
		newInst.satMap[k] = newV
	}

	newInst.contentMap = make(map[uint64]string)
	newInst.contentToIdMap = make(map[string]uint64)
	for k, v := range p.contentMap {
		newInst.contentMap[k] = v
		newInst.contentToIdMap[v] = k
	}

	newInst.addedContentIdMap = make(map[uint64]bool)
	for k, v := range p.addedContentIdMap {
		newInst.addedContentIdMap[k] = v
	}

	newInst.inscriptionToNftIdMap = make(map[string]int64)
	for k, v := range p.inscriptionToNftIdMap {
		newInst.inscriptionToNftIdMap[k] = v
	}

	newInst.nftIdToinscriptionMap = make(map[int64]string)
	for k, v := range p.nftIdToinscriptionMap {
		newInst.nftIdToinscriptionMap[k] = v
	}

	newInst.contentTypeMap = make(map[int]string)
	newInst.contentTypeToIdMap = make(map[string]int)
	for k, v := range p.contentTypeMap {
		newInst.contentTypeMap[k] = v
		newInst.contentTypeToIdMap[v] = k
	}

	newInst.nftAdded = make([]*common.Nft, len(p.nftAdded))
	for i, nft := range p.nftAdded {
		newInst.nftAdded[i] = nft.Clone()
	}

	newInst.utxoDeled = make([]uint64, len(p.utxoDeled))
	copy(newInst.utxoDeled, p.utxoDeled)

	newInst.status = p.status.Clone()

	return newInst
}

func (p *NftIndexer) Subtract(another *NftIndexer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for k := range another.utxoMap {
		delete(p.utxoMap, k)
	}
	for k, v := range another.satMap {
		nv, ok := p.satMap[k]
		if ok {
			if nv.UtxoId == v.UtxoId {
				// 没有变化就删除
				delete(p.satMap, k)
				// 同时必须删除utxoMap中对应的数据，保持一致
				delete(p.utxoMap, nv.UtxoId)
			}
		}
	}
	for k := range another.contentMap {
		delete(p.contentMap, k)
	}
	for k := range another.contentToIdMap {
		delete(p.contentToIdMap, k)
	}
	for k := range another.addedContentIdMap {
		delete(p.addedContentIdMap, k)
	}
	for k := range another.inscriptionToNftIdMap {
		delete(p.inscriptionToNftIdMap, k)
	}
	for k := range another.nftIdToinscriptionMap {
		delete(p.nftIdToinscriptionMap, k)
	}
	for k := range another.contentTypeMap {
		delete(p.contentTypeMap, k)
	}
	for k := range another.contentTypeToIdMap {
		delete(p.contentTypeToIdMap, k)
	}
	
	p.nftAdded = append([]*common.Nft(nil), p.nftAdded[len(another.nftAdded):]...)
	p.utxoDeled = append([]uint64(nil), p.utxoDeled[len(another.utxoDeled):]...)
}

// func (p *NftIndexer) IsEnabled() bool {
// 	return p.bEnabled
// }

func (p *NftIndexer) GetBaseIndexer() *base.BaseIndexer {
	return p.baseIndexer
}

func (p *NftIndexer) Repair() {

	fixingUtxoMap := make(map[uint64][]*SatOffset)
	p.db.BatchRead([]byte(DB_PREFIX_UTXO), false, func(k, v []byte) error {
		var value NftsInUtxo
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}

		utxoId, err := ParseUtxoKey(string(k))
		if err != nil {
			common.Log.Panicf("item.Key error: %v", err)
		}

		for _, sat := range value.Sats {
			if sat.Sat < 0 {
				nv := make([]*SatOffset, 0)
				for _, sat := range value.Sats {
					if sat.Sat >= 0 {
						nv = append(nv, sat)
					}
				}
				fixingUtxoMap[utxoId] = nv
				break
			}
		}
		return nil
	})

	common.Log.Infof("detect %d utxo has unbound sats", len(fixingUtxoMap))

	wb := p.db.NewWriteBatch()
	defer wb.Close()

	for utxoId, sats := range fixingUtxoMap {
		utxokey := GetUtxoKey(utxoId)
		var err error
		if len(sats) == 0 {
			err = wb.Delete([]byte(utxokey))
		} else {
			utxoValue := NftsInUtxo{Sats: sats}
			err = db.SetDBWithProto3([]byte(utxokey), &utxoValue, wb)
		}

		if err != nil {
			common.Log.Panicf("NftIndexer->Repair Error setting %s in db %v", utxokey, err)
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("NftIndexer->Repair Flush failed. %v", err)
	}
}

func (b *NftIndexer) getContentId(content string) (uint64, error) {
	id, ok := b.contentToIdMap[content]
	if ok {
		return id, nil
	}

	var err error
	id, err = GetContentIdFromDB(b.db, content)
	if err == nil {
		b.contentToIdMap[content] = id
		b.contentMap[id] = content
	}

	return id, err
}

func (b *NftIndexer) getContentById(id uint64) (string, error) {
	content, ok := b.contentMap[id]
	if ok {
		return content, nil
	}

	var err error
	content, err = GetContentByIdFromDB(b.db, id)
	if err == nil {
		b.contentToIdMap[content] = id
		b.contentMap[id] = content
	}

	return content, err
}


func (b *NftIndexer) getInscriptionIdByNftId(id int64) (string, error) {
	inscriptionId, ok := b.nftIdToinscriptionMap[id]
	if ok {
		return inscriptionId, nil
	}

	var err error
	nft := b.getNftWithId(id)
	if nft != nil {
		inscriptionId = nft.Base.InscriptionId
		b.inscriptionToNftIdMap[inscriptionId] = id
		b.nftIdToinscriptionMap[id] = inscriptionId
	}

	return inscriptionId, err
}

// 每个NFT Mint都调用
func (p *NftIndexer) NftMint(input *common.TxInput, nft *common.Nft) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	//p.nftBuffer = append(p.nftBuffer, nft)

	p.nftMint(input, nft)
}

func (p *NftIndexer) nftMint(input *common.TxInput, nft *common.Nft) {

	// sat 还没有调整过，这里无法判断
	// if nft.Base.Sat >= 0 && nft.Base.CurseType == 0 {
	// 	// 检查是否同一个聪上有多个铸造
	// 	nftsInSat := p.getNftsWithSat(nft.Base.Sat)
	// 	if nftsInSat != nil {
	// 		if int(nftsInSat.CurseCount) < len(nftsInSat.Nfts) {
	// 			// 已经存在非cursed的铭文，后面的铭文都是reinscription
	// 			nft.Base.CurseType = int32(ordCommon.Reinscription)
	// 			common.Log.Infof("%s is reinscription in sat %d", nft.Base.InscriptionId, nft.Base.Sat)
	// 		}
	// 	}
	// }

	if nft.Base.CurseType != 0 && nft.Base.BlockHeight >= int32(common.Jubilee_Height) {
		nft.Base.CurseType = 0
	}

	if nft.Base.CurseType != 0 {
		p.status.CurseCount++
		nft.Base.Id = -int64(p.status.CurseCount) // 从-1开始
	} else {
		nft.Base.Id = int64(p.status.Count) // 从0开始
		p.status.Count++
	}

	p.nftAdded = append(p.nftAdded, nft)

	if nft.Base.Sat < 0 {
		// mainnet: c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151
		// unbound nft，负数铭文，没有绑定任何聪，也不在哪个utxo中，也没有地址，仅保存数据
		// 在Jubilee之前属于cursed铭文，Jubilee之后，正常编号
		p.status.Unbound++
		nft.Base.Sat = -int64(p.status.Unbound) // 从-1开始

		// 直接添加，为了保存sat信息
		info := &SatInfo{
			AddressId: nft.OwnerAddressId,
			UtxoId:    nft.UtxoId,
			Offset:    nft.Offset,
			Nfts:      make(map[int64]bool),
		}
		info.Nfts[nft.Base.Id] = true
		p.satMap[nft.Base.Sat] = info

		return
	}

	// 批量铸造时，多个nft输出到同一个utxo
	p.nftAddedUtxoMap[nft.UtxoId] = append(p.nftAddedUtxoMap[nft.UtxoId], nft)
	p.inscriptionToNftIdMap[nft.Base.InscriptionId] = nft.Base.Id
	p.nftIdToinscriptionMap[nft.Base.Id] = nft.Base.InscriptionId

	// 为节省空间作准备
	ct := string(nft.Base.ContentType)
	_, ok := p.contentTypeToIdMap[ct]
	if !ok {
		p.status.ContentTypeCount++ // 从1开始
		ctId := p.status.ContentTypeCount

		p.contentTypeMap[ctId] = ct
		p.contentTypeToIdMap[ct] = ctId
	}

	clen := len(nft.Base.Content)
	if clen > 16 && clen < 512 {
		// 转换为id
		content := string(nft.Base.Content)
		id, err := p.getContentId(content)
		if err != nil {
			p.status.ContentCount++
			id = p.status.ContentCount // 0 无效，从1开始

			p.contentMap[id] = content
			p.contentToIdMap[content] = id
			p.addedContentIdMap[id] = true
		}
		nft.Base.ContentId = id
	}

	if nft.Base.Delegate != "" {
		delegate := p.getNftWithInscriptionId(nft.Base.Delegate)
		if delegate != nil {
			p.inscriptionToNftIdMap[nft.Base.Delegate] = delegate.Base.Id
			p.nftIdToinscriptionMap[delegate.Base.Id] = nft.Base.Delegate
		}
	}

	if nft.Base.Parent != "" {
		parent := p.getNftWithInscriptionId(nft.Base.Parent)
		if parent != nil {
			p.inscriptionToNftIdMap[nft.Base.Parent] = parent.Base.Id
			p.nftIdToinscriptionMap[parent.Base.Id] = nft.Base.Parent
		}
	}
	
}

// Mint和Transfer需要仔细协调，确保新增加的nft可以正确被转移
func (p *NftIndexer) UpdateTransfer(block *common.Block, coinbase []*common.Range) {
	if block.Height < p.enableHeight {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// if block.Height == 30689 {
	// 	common.Log.Infof("")
	// }

	// 预加载
	startTime := time.Now()
	p.db.View(func(txn common.ReadBatch) error {
		type pair struct {
			key    string
			utxoId uint64 // utxoId
		}
		inputsToLoad := make([]*pair, 0)
		for _, tx := range block.Transactions[1:] {
			for _, input := range tx.Inputs {
				_, ok := p.utxoMap[input.UtxoId]
				if ok {
					continue
				}
				inputsToLoad = append(inputsToLoad, &pair{
					key:    GetUtxoKey(input.UtxoId),
					utxoId: input.UtxoId,
				})
			}
		}
		// pebble数据库的优化手段: 尽可能将随机读变成按照key的顺序读
		sort.Slice(inputsToLoad, func(i, j int) bool {
			return inputsToLoad[i].key < inputsToLoad[j].key
		})
		satsToLoad := make(map[int64]bool)
		for _, v := range inputsToLoad {
			value := NftsInUtxo{}
			err := db.GetValueFromTxnWithProto3([]byte(v.key), txn, &value)
			if err != nil {
				continue
			}
			p.utxoMap[v.utxoId] = value.Sats
			for _, sat := range value.Sats {
				satsToLoad[sat.Sat] = true
			}
		}

		satLoadingVector := make([]int64, 0, len(satsToLoad))
		for k := range satsToLoad {
			satLoadingVector = append(satLoadingVector, k)
		}
		sort.Slice(satLoadingVector, func(i, j int) bool {
			return satLoadingVector[i] < satLoadingVector[j]
		})
		for _, v := range satLoadingVector {
			_, ok := p.satMap[v]
			if ok {
				continue
			}
			value := common.NftsInSat{}
			err := loadNftsInSatFromTxn(v, &value, txn)
			if err != nil {
				common.Log.Panicf("block %d loadNftsInSatFromTxn sat %d failed, %v", block.Height, v, err)
			}

			info := &SatInfo{
				AddressId: value.OwnerAddressId,
				UtxoId:    value.UtxoId,
				Offset:    value.Offset,
				CurseCount: int(value.CurseCount),
				Nfts:      make(map[int64]bool),
			}
			for _, nftId := range value.Nfts {
				info.Nfts[nftId] = true
			}
			p.satMap[v] = info
		}

		return nil
	})

	// 计算新位置
	coinbaseInput := common.NewTxOutput(coinbase[0].Size)
	for _, tx := range block.Transactions[1:] {
		var allInput *common.TxOutput
		for _, in := range tx.Inputs {
			// if tx.TxId == "408d74bb4c068c4a43282af3d3b403c285ea0863f63c7bddbd6a064006e3ea74" {
			// 	common.Log.Infof("")
			// }
			input := in.Clone() // 不要影响原来tx的数据
			sats := p.utxoMap[input.UtxoId] // 已经铭刻的聪
			if len(sats) > 0 {
				for _, sat := range sats {
					info := p.satMap[sat.Sat]
					if info == nil {
						common.Log.Panicf("%s should load sat %d before", input.OutPointStr, sat.Sat)
					}
					asset := common.AssetInfo{
						Name: common.AssetName{
							Protocol: common.PROTOCOL_NAME_ORD,
							Type:     common.ASSET_TYPE_NFT,
							Ticker:   fmt.Sprintf("%d", sat.Sat), // 绑定了资产的聪
						},
						Amount:     *common.NewDecimal(1, 0),
						BindingSat: uint32(len(info.Nfts)),
					}
					input.Assets.Add(&asset)
					input.Offsets[asset.Name] = common.AssetOffsets{&common.OffsetRange{Start: sat.Offset, End: sat.Offset + 1}}
				}

				delete(p.utxoMap, input.UtxoId)
				p.utxoDeled = append(p.utxoDeled, input.UtxoId)
			}

			if allInput == nil {
				allInput = input.Clone()
			} else {
				allInput.Append(input)
			}
		}

		change := p.innerUpdateTransfer3(tx, allInput)
		coinbaseInput.Append(change)
	}

	// 处理哪些直接输出到奖励聪的铸造结果
	tx := block.Transactions[0]
	change := p.innerUpdateTransfer3(tx, coinbaseInput)
	if !change.Zero() {
		common.Log.Panicf("UpdateTransfer should consume all input assets")
	}
	

	p.nftAddedUtxoMap = make(map[uint64][]*common.Nft)

	common.Log.Infof("NftIndexer.UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))
}

func (p *NftIndexer) innerUpdateTransfer3(tx *common.Transaction,
	input *common.TxOutput) *common.TxOutput {
	// 只考虑放在第一个地址上 (output的地址处理过，肯定有值)

	change := input
	for _, txOut := range tx.Outputs {
		// if txOut.UtxoId == 1016876457263104 || txOut.UtxoId == 1022786333310976 || txOut.UtxoId == 1022958131478528 {
		// 	common.Log.Infof("")
		// }
		if txOut.OutValue.Value == 0 {
			continue
		}

		newOut, newChange, err := change.Cut(txOut.OutValue.Value)
		if err != nil {
			common.Log.Panicf("innerUpdateTransfer3 Cut failed, %v", err)
		}

		sats := make([]*SatOffset, 0)
		change = newChange
		if len(newOut.Assets) != 0 {
			for _, asset := range newOut.Assets {
				if asset.Name.Protocol == common.PROTOCOL_NAME_ORD && 
				asset.Name.Type == common.ASSET_TYPE_NFT {
					sat, err := strconv.ParseInt(asset.Name.Ticker, 10, 64)
					if err != nil {
						common.Log.Panicf("innerUpdateTransfer3 ParseInt %s failed, %v", asset.Name.Ticker, err)
					}
					offsets := newOut.Offsets[asset.Name]

					// 更新聪的位置
					satInfo := p.satMap[sat]
					satInfo.AddressId = txOut.AddressId
					satInfo.UtxoId = txOut.UtxoId
					satInfo.Offset = offsets[0].Start

					sats = append(sats, &SatOffset{
						Sat:    sat,
						Offset: satInfo.Offset,
					})
				}
			}
		}

		// 合并本次铸造的资产
		addedNft := p.nftAddedUtxoMap[txOut.UtxoId] // 本次区块中铭刻的聪
		for _, nft := range addedNft {
			newSat := true
			// 检查是否是重复铭刻
			for _, s := range sats {
				if s.Offset == nft.Offset { // 偏移相同，是同一个聪
					if s.Sat != nft.Base.Sat {
						nft.Base.Sat = s.Sat // 同一个聪，需要命名一致
						// 根据ordinals规则，判断是否是reinscription
						if nft.Base.CurseType == 0 {
							satInfo := p.satMap[nft.Base.Sat] // 预加载，肯定有值
							if int(satInfo.CurseCount) < len(satInfo.Nfts) {
								// 已经存在非cursed的铭文，后面的铭文都是reinscription
								nft.Base.CurseType = int32(ordCommon.Reinscription)
								p.status.CurseCount++
								common.Log.Infof("%s is reinscription in sat %d", nft.Base.InscriptionId, nft.Base.Sat)
							}
						}
					}
					newSat = false
					break
				}
			}
			if newSat {
				sats = append(sats, &SatOffset{
					Sat:    nft.Base.Sat,
					Offset: nft.Offset,
				})
			}

			// 添加到satMap
			info, ok := p.satMap[nft.Base.Sat]
			if !ok {
				info = &SatInfo{
					AddressId: nft.OwnerAddressId,
					UtxoId:    nft.UtxoId,
					Offset:    nft.Offset,
					Nfts:      make(map[int64]bool),
				}
				p.satMap[nft.Base.Sat] = info
			}
			info.Nfts[nft.Base.Id] = true
			if nft.Base.CurseType != 0 {
				info.CurseCount++
			}
		}
		
		if len(sats) > 0 {
			p.utxoMap[txOut.UtxoId] = sats
		}
	}
	return change
}

// fast
func (p *NftIndexer) getBindingSatsWithUtxo(utxoId uint64) []*SatOffset {
	sats, ok := p.utxoMap[utxoId]
	if ok {
		return sats
	}

	value := NftsInUtxo{}
	err := p.db.View(func(txn common.ReadBatch) error {
		return loadUtxoValueFromTxn(utxoId, &value, txn)
	})
	if err != nil {
		return nil
	}

	p.utxoMap[utxoId] = value.Sats
	return value.Sats
}

func (p *NftIndexer) refreshNft(nft *common.Nft) {
	satinfo, ok := p.satMap[nft.Base.Sat]
	if ok {
		nft.OwnerAddressId = satinfo.AddressId
		nft.UtxoId = satinfo.UtxoId
		nft.Offset = satinfo.Offset
	}
}

// 注意
func (p *NftIndexer) getNftInBuffer(id int64) *common.Nft {
	for _, nft := range p.nftAdded {
		if nft.Base.Id == id {
			p.refreshNft(nft)
			return nft
		}
	}
	return nil
}

func (p *NftIndexer) getNftInBuffer2(inscriptionId string) *common.Nft {
	for _, nft := range p.nftAdded {
		if nft.Base.InscriptionId == inscriptionId {
			p.refreshNft(nft)
			return nft
		}
	}
	return nil
}

func (p *NftIndexer) getNftInBuffer4(sat int64) []*common.Nft {
	result := make([]*common.Nft, 0)
	for _, nft := range p.nftAdded {
		if nft.Base.Sat == sat {
			p.refreshNft(nft)
			result = append(result, nft)
		}
	}
	return result
}

// 跟base数据库同步
func (p *NftIndexer) UpdateDB() {
	//common.Log.Infof("NftIndexer->UpdateDB start...")
	if !p.IsEnabled() {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	startTime := time.Now()

	//nftmap := p.prefetchNftsFromDB()
	buckDB := NewBuckStore(p.db)
	buckNfts := make(map[int64]*BuckValue)

	wb := p.db.NewWriteBatch()
	defer wb.Close()

	for _, nft := range p.nftAdded {
		key := GetInscriptionIdKey(nft.Base.InscriptionId)
		value := InscriptionInDB{Sat: nft.Base.Sat, Id: nft.Base.Id}
		err := db.SetDB([]byte(key), &value, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}

		key = GetInscriptionAddressKey(nft.Base.InscriptionAddress, nft.Base.Id)
		err = db.SetDB([]byte(key), nft.Base.Sat, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}

		// 节省空间
		ctId := p.contentTypeToIdMap[string(nft.Base.ContentType)]
		nft.Base.ContentType = []byte(fmt.Sprintf("%d", ctId))
		if nft.Base.ContentId != 0 {
			nft.Base.Content = nil
		}
		if nft.Base.Delegate != "" {
			id, ok := p.inscriptionToNftIdMap[nft.Base.Delegate]
			if ok {
				nft.Base.Delegate = fmt.Sprintf("%x", id)
			}
		}
		if nft.Base.Parent != "" {
			id, ok := p.inscriptionToNftIdMap[nft.Base.Parent]
			if ok {
				nft.Base.Parent = fmt.Sprintf("%x", id)
			}
		}

		key = GetNftKey(nft.Base.Id)
		err = db.SetDBWithProto3([]byte(key), nft.Base, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}

		buckNfts[nft.Base.Id] = &BuckValue{Sat: nft.Base.Sat}
	}

	// 处理nft的转移
	for sat, nft := range p.satMap {
		key := GetSatKey(sat)

		info := &common.NftsInSat{
			Sat:            sat,
			OwnerAddressId: nft.AddressId,
			UtxoId:         nft.UtxoId,
			Offset:         nft.Offset,
			CurseCount:     int32(nft.CurseCount),
			Nfts:           make([]int64, 0, len(nft.Nfts)),
		}
		for k := range nft.Nfts {
			info.Nfts = append(info.Nfts, k)
		}
		sort.Slice(info.Nfts, func(i, j int) bool {
			return info.Nfts[i] < info.Nfts[j]
		})

		err := db.SetDBWithProto3([]byte(key), info, wb)
		//err := db.SetDB([]byte(key), nft, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}
	}

	for _, utxoId := range p.utxoDeled {
		utxokey := GetUtxoKey(utxoId)
		err := wb.Delete([]byte(utxokey))
		if err != nil {
			common.Log.Errorf("NftIndexer->UpdateDB Error delete %s in db %v", utxokey, err)
		}
	}

	for utxoId, sats := range p.utxoMap {
		utxokey := GetUtxoKey(utxoId)
		utxoValue := NftsInUtxo{Sats: sats}
		// err := db.SetDB([]byte(utxokey), &utxoValue, wb)
		err := db.SetDBWithProto3([]byte(utxokey), &utxoValue, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", utxokey, err)
		}
	}

	for contentId := range p.addedContentIdMap {
		key := GetContentIdKey(contentId)
		value := p.contentMap[contentId]
		err := db.SetDB([]byte(key), value, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}

		err = BindContentDBKeyToId(value, contentId, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}
	}

	for ctId := p.lastContentTypeId; ctId < p.status.ContentTypeCount; ctId++ {
		key := GetCTKey(ctId)
		value := p.contentTypeMap[ctId]
		err := db.SetDB([]byte(key), value, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}
	}

	err := db.SetDB([]byte(NFT_STATUS_KEY), p.status, wb)
	if err != nil {
		common.Log.Panicf("NftIndexer->UpdateDB Error setting in db %v", err)
	}

	err = wb.Flush()
	if err != nil {
		common.Log.Panicf("NftIndexer->UpdateDB Error wb flushing writes to db %v", err)
	}

	err = buckDB.BatchPut(buckNfts)
	if err != nil {
		common.Log.Panicf("NftIndexer->UpdateDB BatchPut %v", err)
	}

	// reset memory buffer
	//p.satTree = indexer.NewSatRBTress()
	p.nftAdded = make([]*common.Nft, 0)
	p.utxoMap = make(map[uint64][]*SatOffset)
	p.utxoDeled = make([]uint64, 0)
	p.satMap = make(map[int64]*SatInfo)
	p.contentMap = make(map[uint64]string)
	p.contentToIdMap = make(map[string]uint64)
	p.inscriptionToNftIdMap = make(map[string]int64)
	p.nftIdToinscriptionMap = make(map[int64]string)
	p.addedContentIdMap = make(map[uint64]bool)
	p.lastContentTypeId = p.status.ContentTypeCount

	common.Log.Infof("NftIndexer->UpdateDB takes %v", time.Since(startTime))
}

// 耗时很长。仅用于在数据编译完成时验证数据，或者测试时验证数据。
func (p *NftIndexer) CheckSelf(baseDB common.KVDB) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	common.Log.Info("NftIndexer->checkSelf ... ")

	startTime := time.Now()
	common.Log.Infof("stats: %v", p.status)

	// var wg sync.WaitGroup
	// wg.Add(3)


	// p.db.BatchRead([]byte(DB_PREFIX_NFT), false, func(k, v []byte) error {
	// 	//defer wg.Done()

	// 	var value common.InscribeBaseContent
	// 	err := db.DecodeBytesWithProto3(v, &value)
	// 	if err != nil {
	// 		common.Log.Panicf("item.Value error: %v", err)
	// 	}
	// 	if value.CurseType != 0 {
	// 		common.Log.Infof("%d %s is cursed %d", value.Id, value.InscriptionId, value.CurseType)
	// 	}

	// 	return nil
	// })


	addressesInT1 := make(map[uint64]bool, 0)
	utxosInT1 := make(map[uint64]bool, 0)
	satsInT1 := make(map[uint64]uint64, 0)
	nftsInT1 := make(map[int64]bool, 0)
	startTime2 := time.Now()
	common.Log.Infof("calculating in %s table ...", DB_PREFIX_SAT)
	p.db.BatchRead([]byte(DB_PREFIX_SAT), false, func(k, v []byte) error {
		//defer wg.Done()

		var value common.NftsInSat
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}
		if value.Sat < 0 {
			// 负数铭文，没有绑定到任何聪的铭文，只统计nft数量
			for _, nftId := range value.Nfts {
				nftsInT1[nftId] = true
			}
			return nil
		}

		addressesInT1[value.OwnerAddressId] = true
		utxosInT1[value.UtxoId] = true
		satsInT1[uint64(value.Sat)] = value.UtxoId
		for _, nftId := range value.Nfts {
			nftsInT1[nftId] = true
		}

		return nil
	})
	common.Log.Infof("%s table takes %v", DB_PREFIX_SAT, time.Since(startTime2))
	common.Log.Infof("1: address %d, utxo %d, sats %d, nfts %d", len(addressesInT1), len(utxosInT1), len(satsInT1), len(nftsInT1))

	// utxo的数据涉及到delete操作，但是badger的delete操作有隐藏的bug，需要检查下该utxo是否存在
	utxosInT2 := make(map[uint64]bool)
	satsInT2 := make(map[uint64]uint64)
	startTime2 = time.Now()
	common.Log.Infof("calculating in %s table ...", DB_PREFIX_UTXO)
	p.db.BatchRead([]byte(DB_PREFIX_UTXO), false, func(k, v []byte) error {
		//defer wg.Done()

		var value NftsInUtxo
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}

		utxoId, err := ParseUtxoKey(string(k))
		if err != nil {
			common.Log.Panicf("item.Key error: %v", err)
		}

		utxosInT2[utxoId] = true
		for _, sat := range value.Sats {
			if sat.Sat < 0 { // 不统计负数铭文
				continue
			}
			satsInT2[uint64(sat.Sat)] = utxoId
		}
		return nil
	})
	common.Log.Infof("%s table takes %v", DB_PREFIX_UTXO, time.Since(startTime2))
	common.Log.Infof("2: utxo %d, sats %d", len(utxosInT2), len(satsInT2))

	bs := NewBuckStore(p.db)
	lastkey := bs.GetLastKey()
	var buckmap map[int64]*BuckValue
	getbuck := func() {
		//defer wg.Done()
		startTime2 := time.Now()
		buckmap = bs.GetAll()
		common.Log.Infof("%s table takes %v", DB_PREFIX_BUCK, time.Since(startTime2))
		common.Log.Infof("3: nfts %d", len(buckmap))
	}
	getbuck()

	//wg.Wait()
	common.Log.Infof("nft count: %d %d %d", p.status.Count-uint64(len(p.nftAdded)), len(nftsInT1), lastkey+1)

	wrongAddress := make([]uint64, 0)
	wrongUtxo1 := make([]uint64, 0)
	wrongUtxo2 := make([]uint64, 0)

	//wg.Add(2)
	baseDB.View(func(txn common.ReadBatch) error {
		//defer wg.Done()
		startTime2 = time.Now()
		for address := range addressesInT1 {
			key := db.GetAddressIdKey(address)
			_, err := txn.Get(key)
			if err != nil {
				wrongAddress = append(wrongAddress, address)
			}
		}
		common.Log.Infof("check addressesInT1 in baseDB takes %v", time.Since(startTime2))
		return nil
	})

	// 耗时很长，90w的高度，基本要10-20分钟
	baseDB.View(func(txn common.ReadBatch) error {
		//defer wg.Done()
		startTime2 = time.Now()
		for utxo := range utxosInT2 {
			key := db.GetUtxoIdKey(utxo)
			_, err := txn.Get(key)
			if err != nil {
				wrongUtxo2 = append(wrongUtxo2, utxo)
			}
		}
		common.Log.Infof("check utxosInT2 in baseDB takes %v", time.Since(startTime2))
		return nil
	})

	//wg.Wait()
	common.Log.Infof("check in baseDB completed")

	wrongIds := make([]int64, 0)
	wrongSats := make([]int64, 0)
	for id, v := range buckmap {
		_, ok := nftsInT1[id]
		if !ok {
			wrongIds = append(wrongIds, id)
		}
		if v.Sat < 0 {
			continue
		}
		_, ok = satsInT1[uint64(v.Sat)]
		if !ok {
			wrongSats = append(wrongSats, v.Sat)
		}
	}

	common.Log.Infof("wrong address %d", len(wrongAddress))
	common.Log.Infof("wrong id %d", len(wrongIds))
	common.Log.Infof("wrong sat %d", len(wrongSats))
	common.Log.Infof("wrong utxo1 %d, utxo2 %d", len(wrongUtxo1), len(wrongUtxo2))
	for i, value := range wrongAddress {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong address %d: %d", i, value)
	}
	for i, value := range wrongIds {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong id %d: %d", i, value)
	}
	for i, value := range wrongSats {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong sat %d: %d", i, value)
	}
	for i, value := range wrongUtxo1 {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong utxo1 %d: %d", i, value)
	}
	for i, value := range wrongUtxo2 {
		if i > 10 {
			break
		}
		common.Log.Infof("wrong utxo2 %d: %d", i, value)
	}

	result := true
	if len(wrongAddress) != 0 || len(wrongIds) != 0 || len(wrongSats) != 0 || len(wrongUtxo1) != 0 {
		common.Log.Error("data wrong")
		result = false
	}

	count := p.status.Count - uint64(len(p.nftAdded))
	if count != uint64(len(nftsInT1)) || count != uint64(lastkey+1) {
		common.Log.Errorf("nft count different %d %d %d", count, len(nftsInT1), uint64(lastkey+1))
		result = false
	}

	common.Log.Infof("utxos not in table %s", DB_PREFIX_UTXO)
	utxos1 := findDifferentItems(utxosInT1, utxosInT2)
	if len(utxos1) > 0 {
		p.printfUtxos(utxos1, baseDB)
		common.Log.Errorf("utxo1 wrong %d %v", len(utxos1), utxos1)
		result = false
	}

	common.Log.Infof("utxos not in table %s", DB_PREFIX_SAT)
	utxos2 := findDifferentItems(utxosInT2, utxosInT1)
	if len(utxos2) > 0 {
		p.printfUtxos(utxos2, baseDB)
		common.Log.Errorf("utxo2 wrong %d", len(utxos2))
		result = false
	}

	// needReCheck := false
	common.Log.Infof("sats not in table %s", DB_PREFIX_UTXO)
	sats1 := findDifferentItemsV2(satsInT1, satsInT2)
	if len(sats1) > 0 {
		common.Log.Errorf("sat1 wrong %d %v", len(sats1), sats1)
		result = false
	}

	common.Log.Infof("sats not in table %s", DB_PREFIX_SAT)
	sats2 := findDifferentItemsV2(satsInT2, satsInT1)
	if len(sats2) > 0 {
		common.Log.Errorf("sats2 wrong %d", len(sats2))
		result = false
	}

	// 1. 每个utxoId都存在baseDB中
	// 2. 两个表格中的数据相互对应: name，sat
	// 3. name的总数跟stats中一致
	if result {
		common.Log.Infof("nft DB checked successfully, %v", time.Since(startTime))
	}

	return result
}

func findDifferentItemsV2(map1, map2 map[uint64]uint64) map[uint64]uint64 {
	differentItems := make(map[uint64]uint64)
	for key, v := range map1 {
		if _, exists := map2[key]; !exists {
			differentItems[key] = v
		}
	}

	return differentItems
}

func findDifferentItems(map1, map2 map[uint64]bool) map[uint64]bool {
	differentItems := make(map[uint64]bool)
	for key := range map1 {
		if _, exists := map2[key]; !exists {
			differentItems[key] = true
		}
	}

	return differentItems
}

// only for test
func (b *NftIndexer) printfUtxos(utxos map[uint64]bool, ldb common.KVDB) map[uint64]string {
	result := make(map[uint64]string)
	ldb.BatchRead([]byte(common.DB_KEY_UTXO), false, func(k, v []byte) error {

		var value common.UtxoValueInDB
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Errorf("item.Value error: %v", err)
			return nil
		}

		// 用于打印不存在table中的utxo
		if _, ok := utxos[value.UtxoId]; ok {

			str, err := db.GetUtxoByDBKey(k)
			if err == nil {
				common.Log.Infof("%x %s %d", value.UtxoId, str, value.Value)
				result[value.UtxoId] = str
			}

			delete(utxos, value.UtxoId)
			if len(utxos) == 0 {
				return nil
			}
		}

		return nil
	})

	return result
}

// only for test
func (b *NftIndexer) deleteSats(sats map[uint64]uint64) {
	wb := b.db.NewWriteBatch()
	defer wb.Close()

	for sat := range sats {
		key := GetSatKey(int64(sat))
		err := wb.Delete([]byte(key))
		if err != nil {
			common.Log.Errorf("NftIndexer.deleteSats-> Error deleting db: %v\n", err)
		} else {
			common.Log.Infof("sat deled: %d", sat)
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("NftIndexer.deleteSats-> Error satwb flushing writes to db %v", err)
	}
}
