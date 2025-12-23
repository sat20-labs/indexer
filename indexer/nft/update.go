package nft

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"

	ordCommon "github.com/sat20-labs/indexer/indexer/ord/common"
)

// 每个NFT Mint都调用
func (p *NftIndexer) NftMint(input *common.TxInput, inOffset int64, nft *common.Nft) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if nft.Base.Sat < 0 {
		// 788200: c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151
		// 788312: 99e70421ab229d1ccf356e594512da6486e2dd1abdf6c2cb5014875451ee8073
		// unbound nft，负数铭文，没有绑定任何聪，也不在哪个utxo中，也没有地址，仅保存数据
		// 在Jubilee之前属于cursed铭文，Jubilee之后，正常编号
		p.status.Unbound++
		nft.Base.Sat = -int64(p.status.Unbound) // 从-1开始
	}

	info := &InscribeInfo{
		Input:    input,
		InOffset: inOffset,
		UtxoId:   nft.UtxoId,
		Nft:      nft,
	}

	txMap, ok := p.actionBufferMap[input.InTxIndex]
	if !ok {
		txMap = make(map[int][]*InscribeInfo)
		p.actionBufferMap[input.InTxIndex] = txMap
	}
	txMap[input.TxInIndex] = append(txMap[input.TxInIndex], info)
	p.nftAddedUtxoMap[input.UtxoId] = append(p.nftAddedUtxoMap[input.UtxoId], info)

	p.nftAdded = append(p.nftAdded, nft)
}

// 必须先加载satmap和utxomap
func (p *NftIndexer) sortInscriptionInBlock(block *common.Block) {
	// 对所有铸造信息排序
	type pair struct {
		idx1   int
		idx2   int
		idx3   int
		offset int64
		info   *InscribeInfo
	}

	totalTxs := len(block.Transactions)
	mid := make([]*pair, 0)
	for txIndex, txIndexMap := range p.actionBufferMap {
		for txInIndex, infoVector := range txIndexMap {
			for idx3, info := range infoVector {
				// if info.Input.TxId == "4341ee078033631b1c17b5312ac279e899d456880b3628c7fb332044a7a42f47" {
				// 	common.Log.Infof("")
				// }

				idx := txIndex
				if info.Nft.Base.Sat < 0 {
					// unbound
					idx += totalTxs + 1
				}
				_, outTxIndex, _ := common.FromUtxoId(info.UtxoId)
				if outTxIndex == 0 {
					// fee spent
					idx += totalTxs
				}
				item := &pair{
					idx1:   idx,
					idx2:   txInIndex,
					idx3:   idx3, // txIn内部铭文顺序
					offset: info.InOffset,
					info:   info,
				}
				mid = append(mid, item)
			}
		}
	}
	sort.Slice(mid, func(i, j int) bool {
		if mid[i].idx1 == mid[j].idx1 {
			if mid[i].idx2 == mid[j].idx2 {
				if mid[i].idx3 == mid[j].idx3 {
					return mid[i].offset < mid[j].offset
				}
				return mid[i].idx3 < mid[j].idx3
			}
			return mid[i].idx2 < mid[j].idx2
		}
		return mid[i].idx1 < mid[j].idx1
	})

	traceTaget := int64(0)
	for _, item := range mid {
		p.nftMint(item.info)
		//common.Log.Infof("%d %s", item.info.Nft.Base.Id, item.info.Nft.Base.InscriptionId)

		// nft := item.info.Nft
		// if nft.Base.InscriptionId == "1654500098640d6cedeff295eceeb4b61e63d554ddc5ebade6f2fbd879f640bbi0" {
		// 	traceTaget = nft.Base.Id
		// }
	}

	if traceTaget != 0 {
		for i := traceTaget - 100; i <= traceTaget+10; i++ {
			nft := p.getNftWithId(i)
			common.Log.Infof("%d %s", i, nft.Base.InscriptionId)
		}
	}

}

func (p *NftIndexer) nftMint(info *InscribeInfo) {
	nft := info.Nft

	if nft.Base.BlockHeight >= int32(common.Jubilee_Height) {
		nft.Base.CurseType = 0
	}

	if nft.Base.CurseType != 0 {
		p.status.CurseCount++
		nft.Base.Id = -int64(p.status.CurseCount) // 从-1开始
	} else {
		nft.Base.Id = int64(p.status.Count) // 从0开始
		p.status.Count++
	}

	common.Log.Debugf("nftMint add nft #%d %s", nft.Base.Id, nft.Base.InscriptionId)

	// 为节省空间作准备
	p.inscriptionToNftIdMap[nft.Base.InscriptionId] = nft
	p.nftIdToinscriptionMap[nft.Base.Id] = nft

	ct := string(nft.Base.ContentType)
	_, ok := p.contentTypeToIdMap[ct]
	if !ok {
		p.status.ContentTypeCount++ // 从1开始
		ctId := p.status.ContentTypeCount

		p.contentTypeMap[ctId] = ct
		p.contentTypeToIdMap[ct] = ctId
	}

	clen := len(nft.Base.Content)
	if clen > 32 && clen < 512 {
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
			p.inscriptionToNftIdMap[delegate.Base.InscriptionId] = delegate
			p.nftIdToinscriptionMap[delegate.Base.Id] = delegate
		}
	}

	if nft.Base.Parent != "" {
		parent := p.getNftWithInscriptionId(nft.Base.Parent)
		if parent != nil {
			p.inscriptionToNftIdMap[parent.Base.InscriptionId] = parent
			p.nftIdToinscriptionMap[parent.Base.Id] = parent
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

	// prepare 1: 预加载
	startTime := time.Now()
	p.db.View(func(txn common.ReadBatch) error {
		type pair struct {
			key    string
			utxoId uint64 // utxoId
		}
		// 预加载utxomap
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
			satoffsetMap := make(map[int64]int64)
			for _, item := range value.Sats {
				if item.Sat == 0 { // TODO 检查下为何老数据中有很多sat=0的值在里面
					continue
				}
				satoffsetMap[item.Sat] = item.Offset
				satsToLoad[item.Sat] = true
			}
			p.utxoMap[v.utxoId] = satoffsetMap
		}

		// 预加载satmap
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
				AddressId:  value.OwnerAddressId,
				UtxoId:     value.UtxoId,
				Offset:     value.Offset,
				CurseCount: int(value.CurseCount),
				Nfts:       make(map[*common.Nft]bool),
			}
			for _, nftId := range value.Nfts {
				// 不需要加载数据，只是生成一个临时对象
				nft := &common.Nft{
					Base: &common.InscribeBaseContent{
						Id:  nftId,
						Sat: v,
					},
					Offset:         value.Offset,
					OwnerAddressId: value.OwnerAddressId,
					UtxoId:         value.UtxoId,
				}
				info.Nfts[nft] = true
			}
			p.satMap[v] = info
		}

		return nil
	})
	//common.Log.Infof("NftIndexer.UpdateTransfer preload takes %v", time.Since(startTime))

	// prepare 2: calc inscription number
	for _, tx := range block.Transactions[1:] {
		for _, input := range tx.Inputs {
			// if tx.TxId == "b2648c6021e5ca1d71a93728762cf42b5919f6919f57539f676d0051c8f838ae" {
			// 	common.Log.Infof("%d %d %d", block.Height, input.InTxIndex, input.TxInIndex)
			// }

			// 合并资产
			sats := p.utxoMap[input.UtxoId]                 // 已经铭刻的聪
			if sats == nil {
				sats = make(map[int64]int64)
			}

			addedNft := p.nftAddedUtxoMap[input.UtxoId] // 本次区块中铭刻的聪
			// 将铸造资产加入utxomap，并计算可能的reinscription
			for _, info := range addedNft { // 新增加的nft，有可能已经是重复铭刻
				nft := info.Nft
				newSat := true
				for sat, offset := range sats {
					if offset == info.InOffset {
						if sat != nft.Base.Sat {
							nft.Base.Sat = sat // 同一个聪，需要命名一致
							// 根据ordinals规则，判断是否是reinscription
							if nft.Base.CurseType == 0 {
								nftsInSat := p.satMap[nft.Base.Sat] // 预加载，肯定有值
								if int(nftsInSat.CurseCount) < len(nftsInSat.Nfts) {
									// 已经存在非cursed的铭文，后面的铭文都是reinscription
									// Jubilee后，也是需要记录reinscription
									nft.Base.CurseType = int32(ordCommon.Reinscription)
									nft.Base.Reinscription = 1
									common.Log.Debugf("%s is reinscription in sat %d", nft.Base.InscriptionId, nft.Base.Sat)
								}
							}
						}
						newSat = false
						break
					}
				}
				if newSat {
					// 加入utxomap
					sats[nft.Base.Sat] = info.InOffset
				}
				p.addNftToSatMap(nft)
			}

			if len(sats) > 0 {
				p.utxoMap[input.UtxoId] = sats
			}
			
		}
	}

	// 更新铭文的编号，更新satmap
	p.sortInscriptionInBlock(block)
	// hook: prepare for transfer
	p.processCallback.PrepareUpdateTransfer(block, coinbase)

	// 执行真正的transfer
	// 计算新位置，资产直接加入block的交易数据中，方便后面模块直接处理资产数据
	coinbaseInput := common.NewTxOutput(coinbase[0].Size)
	for i, tx := range block.Transactions[1:] {
		var allInput *common.TxOutput
		for _, input := range tx.Inputs {
			// if tx.TxId == "5f3beb0d677f8fbd167d933f63cd9992a747c2bc5dc009d6f8cba55750819163" {
			// 	common.Log.Infof("%d", i)
			// }
			sats, ok := p.utxoMap[input.UtxoId]
			if ok {
				for sat, offset := range sats {
					addSatInfoToOutput(&input.TxOutput, sat, offset)
				}

				delete(p.utxoMap, input.UtxoId)
				p.utxoDeled = append(p.utxoDeled, input.UtxoId)
			}

			if allInput == nil {
				allInput = input.Clone()
			} else {
				allInput.Append(&input.TxOutput)
			}
		}

		change := p.innerUpdateTransfer(tx, allInput)
		// hook: process tx
		change2 := p.processCallback.TxInputProcess(i+1, tx, block, coinbase)
		change.Merge(change2)

		coinbaseInput.Append(change)
	}

	// 处理哪些直接输出到奖励聪的铸造结果
	tx := block.Transactions[0]
	change := p.innerUpdateTransfer(tx, coinbaseInput)
	if !change.Zero() {
		common.Log.Panicf("UpdateTransfer should consume all input assets")
	}
	// hook: process coin base
	p.processCallback.TxInputProcess(0, tx, block, coinbase)

	p.nftAddedUtxoMap = make(map[uint64][]*InscribeInfo)
	p.actionBufferMap = make(map[int]map[int][]*InscribeInfo)
	p.CheckPointWithBlockHeight(block.Height)
	// hook: process checkpoint
	p.processCallback.UpdateTransferFinished(block)

	common.Log.Infof("NftIndexer.UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))
}

// 需要预加载satmap
func (p *NftIndexer) addNftToSatMap(nft *common.Nft) {
	info, ok := p.satMap[nft.Base.Sat]
	if !ok {
		info = &SatInfo{
			AddressId:  nft.OwnerAddressId,
			UtxoId:     nft.UtxoId,
			Offset:     nft.Offset,
			CurseCount: 0,
			Nfts:       make(map[*common.Nft]bool),
		}
		p.satMap[nft.Base.Sat] = info
	}
	info.Nfts[nft] = true
	if nft.Base.CurseType != 0 {
		info.CurseCount++
	}
}

func (p *NftIndexer) addSatToUtxoMap(sat, offset int64, utxoId uint64) {
	if utxoId == common.INVALID_ID {
		return
	}
	satoffsetMap, ok := p.utxoMap[utxoId]
	if !ok {
		satoffsetMap = make(map[int64]int64)
		p.utxoMap[utxoId] = satoffsetMap
	}
	satoffsetMap[sat] = offset
}

func addSatInfoToOutput(output *common.TxOutput, sat, offset int64) {
	asset := common.AssetInfo{
		Name: common.AssetName{
			Protocol: common.PROTOCOL_NAME_ORD,
			Type:     common.ASSET_TYPE_NFT,
			Ticker:   fmt.Sprintf("%d", sat), // 绑定了资产的聪
		},
		Amount:     *common.NewDecimal(1, 0),
		BindingSat: 1,
	}
	output.Assets.Add(&asset)
	output.Offsets[asset.Name] = common.AssetOffsets{&common.OffsetRange{Start: offset, End: offset + 1}}
}

func (p *NftIndexer) innerUpdateTransfer(tx *common.Transaction,
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
			common.Log.Panicf("innerUpdateTransfer Cut failed, %v", err)
		}

		change = newChange
		if len(newOut.Assets) != 0 {
			for _, asset := range newOut.Assets {
				if asset.Name.Protocol == common.PROTOCOL_NAME_ORD &&
					asset.Name.Type == common.ASSET_TYPE_NFT {
					sat, err := strconv.ParseInt(asset.Name.Ticker, 10, 64)
					if err != nil {
						common.Log.Panicf("innerUpdateTransfer ParseInt %s failed, %v", asset.Name.Ticker, err)
					}
					if sat == 0 {
						common.Log.Panicf("innerUpdateTransfer invalid sat %s", asset.Name.Ticker)
					}
					offsets := newOut.Offsets[asset.Name]
					offset := offsets[0].Start

					// 更新聪的位置
					satInfo := p.satMap[sat]
					satInfo.AddressId = txOut.AddressId
					satInfo.UtxoId = txOut.UtxoId
					satInfo.Offset = offset

					p.addSatToUtxoMap(sat, offset, txOut.UtxoId)
					addSatInfoToOutput(&txOut.TxOutput, sat, offset) // 合并资产信息，供下一个模块处理
				}
			}
		}
	}
	return change
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

	//db.PrintLog = true

	//common.Log.Debugf("add %d nft...", len(p.nftAdded))
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
			d, ok := p.inscriptionToNftIdMap[nft.Base.Delegate]
			if ok {
				nft.Base.Delegate = fmt.Sprintf("%x", d.Base.Id)
			}
		}
		if nft.Base.Parent != "" {
			d, ok := p.inscriptionToNftIdMap[nft.Base.Parent]
			if ok {
				nft.Base.Parent = fmt.Sprintf("%x", d.Base.Id)
			}
		}

		key = GetNftKey(nft.Base.Id)
		err = db.SetDBWithProto3([]byte(key), nft.Base, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}

		buckNfts[nft.Base.Id] = &BuckValue{Sat: nft.Base.Sat}
	}
	//common.Log.Debugf("NftIndexer->UpdateDB add %d nft takes %v", len(p.nftAdded), time.Since(startTime))
	//startTime = time.Now()
	//db.PrintLog = false

	// 处理nft的转移
	//common.Log.Debugf("add %d sat...", len(p.satMap))
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
			info.Nfts = append(info.Nfts, k.Base.Id)
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
	//common.Log.Debugf("NftIndexer->UpdateDB add %d sat takes %v", len(p.satMap), time.Since(startTime))
	//startTime = time.Now()

	//common.Log.Debugf("delete %d utxo...", len(p.utxoDeled))
	for _, utxoId := range p.utxoDeled {
		utxokey := GetUtxoKey(utxoId)
		err := wb.Delete([]byte(utxokey))
		if err != nil {
			common.Log.Errorf("NftIndexer->UpdateDB Error delete %s in db %v", utxokey, err)
		}
	}
	//common.Log.Infof("NftIndexer->UpdateDB delete %d utxo takes %v", len(p.utxoDeled), time.Since(startTime))
	//startTime = time.Now()

	//common.Log.Debugf("write %d utxo...", len(p.utxoMap))
	for utxoId, sats := range p.utxoMap {
		utxokey := GetUtxoKey(utxoId)
		satv := make([]*SatOffset, len(sats))
		i := 0
		for sat, offset := range sats {
			if sat == 0 {
				common.Log.Infof("zero sat find in utxo %d", utxoId)
				continue
			}
			satv[i] = &SatOffset{
				Sat:    sat,
				Offset: offset,
			}
			i++
		}
		utxoValue := NftsInUtxo{Sats: satv}
		// err := db.SetDB([]byte(utxokey), &utxoValue, wb)
		err := db.SetDBWithProto3([]byte(utxokey), &utxoValue, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", utxokey, err)
		}
	}
	//common.Log.Debugf("NftIndexer->UpdateDB write %d utxo takes %v", len(p.utxoMap), time.Since(startTime))
	//startTime = time.Now()

	//common.Log.Debugf("add %d content id...", len(p.addedContentIdMap))
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
	//common.Log.Debugf("NftIndexer->UpdateDB add %d content id takes %v", len(p.addedContentIdMap), time.Since(startTime))
	//startTime = time.Now()

	//common.Log.Debugf("add %d content type...", p.status.ContentTypeCount-p.lastContentTypeId)
	for ctId := p.lastContentTypeId; ctId < p.status.ContentTypeCount; ctId++ {
		key := GetCTKey(ctId)
		value := p.contentTypeMap[ctId]
		err := db.SetDB([]byte(key), value, wb)
		if err != nil {
			common.Log.Panicf("NftIndexer->UpdateDB Error setting %s in db %v", key, err)
		}
	}
	//common.Log.Debugf("NftIndexer->UpdateDB add %d content type takes %v", p.status.ContentTypeCount-p.lastContentTypeId, time.Since(startTime))
	//startTime = time.Now()

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
	p.utxoMap = make(map[uint64]map[int64]int64)
	p.utxoDeled = make([]uint64, 0)
	p.satMap = make(map[int64]*SatInfo)
	p.contentMap = make(map[uint64]string)
	p.contentToIdMap = make(map[string]uint64)
	p.inscriptionToNftIdMap = make(map[string]*common.Nft)
	p.nftIdToinscriptionMap = make(map[int64]*common.Nft)
	p.addedContentIdMap = make(map[uint64]bool)
	p.lastContentTypeId = p.status.ContentTypeCount

	common.Log.Infof("NftIndexer->UpdateDB takes %v", time.Since(startTime))
}
