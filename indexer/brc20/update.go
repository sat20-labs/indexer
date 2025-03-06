package brc20

import (
	"fmt"
	"strings"
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
)

// deploy
func (p *BRC20Indexer) UpdateInscribeDeploy(ticker *common.BRC20Ticker) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	name := strings.ToLower(ticker.Name)
	org, ok := p.tickerMap[name]
	if !ok {
		ticker.Id = int64(len(p.tickerMap))
		tickinfo := newTickerInfo(ticker.Name)
		tickinfo.Ticker = ticker
		p.tickerMap[name] = tickinfo
		p.tickerAdded[name] = ticker
	} else {
		// 仅更新显示内容
		p.tickerAdded[name] = org.Ticker
	}

	common.Log.Infof("UpdateInscribeDeploy %s", ticker.Name)
}

// mint
func (p *BRC20Indexer) UpdateInscribeMint(mint *common.BRC20Mint) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ticker, ok := p.tickerMap[strings.ToLower(mint.Name)]
	if !ok {
		// 正常不会走到这里，除非是数据从中间跑
		return
	}
	mint.Id = int64(len(ticker.InscriptionMap))
	ticker.MintAdded = append(ticker.MintAdded, mint)
	ticker.InscriptionMap[mint.Nft.Base.InscriptionId] = common.NewBRC20MintAbbrInfo(mint)

	action := HolderAction{
		Height:   int(mint.Nft.Base.BlockHeight),
		Utxo:     mint.Nft.Base.InscriptionId,
		NftId:    mint.Nft.Base.Id,
		FromAddr: common.INVALID_ID,
		ToAddr:   mint.Nft.OwnerAddressId,
		Ticker:   mint.Name,
		Amount:   mint.Amt,
		Action:   0,
	}
	p.holderActionList = append(p.holderActionList, &action)
	p.addHolderBalance(mint.Name, mint.Nft.OwnerAddressId, mint.Amt)
}

// transfer
func (p *BRC20Indexer) UpdateInscribeTransfer(transfer *common.BRC20Transfer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	nft := common.TransferNFT{
		NftId:  transfer.Nft.Base.Id,
		UtxoId: transfer.UtxoId,
		Amount: transfer.Amt,
	}
	transferInfo := &TransferNftInfo{
		AddressId:   transfer.Nft.OwnerAddressId,
		Ticker:      transfer.Name,
		TransferNft: &nft,
	}
	p.addTransferNft(transferInfo)

	action := HolderAction{
		Height:   int(transfer.Nft.Base.BlockHeight),
		Utxo:     transfer.Nft.Base.InscriptionId,
		NftId:    transfer.Nft.Base.Id,
		FromAddr: common.INVALID_ID,
		ToAddr:   transfer.Nft.OwnerAddressId,
		Ticker:   transfer.Name,
		Amount:   transfer.Amt,
		Action:   1,
	}
	p.holderActionList = append(p.holderActionList, &action)
}

// 增加该address下的资产数据
func (p *BRC20Indexer) addHolderBalance(ticker string, address uint64, amt common.Decimal) {
	ticker = strings.ToLower(ticker)

	info, ok := p.holderMap[address]
	if !ok {
		tickinfo := common.NewBRC20TickAbbrInfo(amt)
		tickers := make(map[string]*common.BRC20TickAbbrInfo)
		tickers[ticker] = tickinfo
		info = &HolderInfo{AddressId: address, Tickers: tickers}
		p.holderMap[address] = info
	} else {
		info.AddressId = address
		tickinfo, ok := info.Tickers[ticker]
		if !ok {
			tickinfo := common.NewBRC20TickAbbrInfo(amt)
			info.Tickers[ticker] = tickinfo
		} else {
			tickinfo.AvailableBalance = *tickinfo.AvailableBalance.Add(&amt)
		}
	}

	holders, ok := p.tickerToHolderMap[ticker]
	if !ok {
		holders = make(map[uint64]bool, 0)
		holders[address] = true
		p.tickerToHolderMap[ticker] = holders
	} else {
		holders[address] = true
		p.tickerToHolderMap[ticker] = holders
	}
}

// 减少该address下的资产数据
func (p *BRC20Indexer) subHolderBalance(ticker string, address uint64, amt common.Decimal) error {
	ticker = strings.ToLower(ticker)

	info, ok := p.holderMap[address]
	if ok {
		info.AddressId = address
		tickinfo, ok := info.Tickers[ticker]
		if ok {
			cmp := tickinfo.AvailableBalance.Cmp(&amt)
			if cmp >= 0 {
				tickinfo.AvailableBalance = *tickinfo.AvailableBalance.Sub(&amt)
				if cmp == 0 {
					holders := p.tickerToHolderMap[ticker]
					delete(holders, address)
					p.tickerToHolderMap[ticker] = holders
				}
				return nil
			}
		}
	}
	return fmt.Errorf("no enough balance")
}

func (p *BRC20Indexer) removeTransferNft(nft *TransferNftInfo) {
	delete(p.transferNftMap, nft.TransferNft.UtxoId)
	holder, ok := p.holderMap[nft.AddressId]
	if ok {
		tickInfo, ok := holder.Tickers[nft.Ticker]
		if ok {
			delete(tickInfo.TransferableData, nft.TransferNft.UtxoId)
		} else {
			common.Log.Panic("can't find ticker info")
		}
	} else {
		common.Log.Panic("can't find ticker info")
	}
}

func (p *BRC20Indexer) addTransferNft(nft *TransferNftInfo) {
	p.transferNftMap[nft.TransferNft.UtxoId] = nft

	bValid := false
	holder, ok := p.holderMap[nft.AddressId]
	if ok {
		tickInfo, ok := holder.Tickers[nft.Ticker]
		if ok {
			tickInfo.TransferableData[nft.TransferNft.UtxoId] = nft.TransferNft
			bValid = true
		}
	}

	if !bValid {
		// 异常路径：没有可用余额，先将transfer铭文保存起来，以后可能可以用（TODO 要看BRC20协议是否这样处理）
		availableBalance := common.NewDecimal(0, nft.TransferNft.Amount.Precision)
		// 仅为了创建tickInfo
		p.addHolderBalance(nft.Ticker, nft.AddressId, *availableBalance)

		holder = p.holderMap[nft.AddressId]
		tickInfo := holder.Tickers[nft.Ticker]
		tickInfo.InvalidTransferableData[nft.TransferNft.UtxoId] = nft.TransferNft
	}
}

func (p *BRC20Indexer) UpdateTransfer(block *common.Block) {
	p.mutex.Lock()

	startTime := time.Now()

	// 检查transferNft转入到哪个输出
	inputTransferNfts := make(map[int64]*TransferNftInfo)
	for _, tx := range block.Transactions[1:] {
		hasTransfer := false
		for _, input := range tx.Inputs {
			nft, ok := p.transferNftMap[input.UtxoId]
			if ok {
				inputTransferNfts[nft.TransferNft.NftId] = nft
				hasTransfer = true
				p.removeTransferNft(nft)
			}
		}

		if hasTransfer {
			for _, output := range tx.Outputs {
				p.innerUpdateTransfer(tx.Txid, output, &inputTransferNfts)
			}
		}
	}

	if len(inputTransferNfts) != 0 {
		tx := block.Transactions[0]
		for _, output := range tx.Outputs {
			p.innerUpdateTransfer(tx.Txid, output, &inputTransferNfts)
		}
	}

	common.Log.Infof("BRC20Indexer->UpdateTransfer loop %d in %v", len(block.Transactions), time.Since(startTime))

	p.mutex.Unlock()
	p.CheckSelf(block.Height)
}

func (p *BRC20Indexer) innerUpdateTransfer(txId string, output *common.Output, inputTransferNfts *map[int64]*TransferNftInfo) {

	// 检查是否存在nft。如果存在，就更新对应的holder数据
	utxoId := common.GetUtxoId(output)
	ids := p.nftIndexer.GetNftsWithUtxo(utxoId)
	for _, nft := range ids {
		transferNft, ok := (*inputTransferNfts)[nft.Base.Id]
		if ok {
			// transfer occur

			err := p.subHolderBalance(transferNft.Ticker, transferNft.AddressId,
				transferNft.TransferNft.Amount)
			if err != nil {
				return
			}
			p.addHolderBalance(transferNft.Ticker, transferNft.AddressId,
				transferNft.TransferNft.Amount)
			action := HolderAction{
				Height:   output.Height,
				Utxo:     common.ToUtxo(txId, int(output.N)),
				NftId:    transferNft.TransferNft.NftId,
				FromAddr: transferNft.AddressId,
				ToAddr:   nft.OwnerAddressId,
				Ticker:   transferNft.Ticker,
				Amount:   transferNft.TransferNft.Amount,
				Action:   2,
			}
			p.holderActionList = append(p.holderActionList, &action)

			delete((*inputTransferNfts), nft.Base.Id)
		}
	}
}

// 跟basic数据库同步
func (p *BRC20Indexer) UpdateDB() {
	//common.Log.Infof("BRC20Indexer->UpdateDB start...")
	startTime := time.Now()

	wb := p.db.NewWriteBatch()
	defer wb.Cancel()

	// new ticker
	for _, v := range p.tickerAdded {
		key := GetTickerKey(v.Name)
		err := db.SetDB([]byte(key), v, wb)
		if err != nil {
			common.Log.Panicf("Error setting %s in db %v", key, err)
		}
	}

	// mint history
	for _, ticker := range p.tickerMap {
		for _, v := range ticker.MintAdded {
			key := GetMintHistoryKey(ticker.Name, v.Nft.Base.InscriptionId)
			err := db.SetDB([]byte(key), v, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", key, err)
			}
		}
	}

	// holder status
	// transfer history
	for _, action := range p.holderActionList {
		switch action.Action {
		case 0: // inscribe-mint
		case 1: // inscribe-transfer
		case 2: // transfer
		}
		// 更新holder数据
		if action.FromAddr != common.INVALID_ID {
			fromKey := GetHolderInfoKey(action.FromAddr, action.Ticker)
			value, ok := p.holderMap[action.FromAddr]
			if ok {
				err := db.SetDB([]byte(fromKey), value, wb)
				if err != nil {
					common.Log.Panicf("Error setting %s in db %v", fromKey, err)
				}
			} else {
				//已经被删除
				err := wb.Delete([]byte(fromKey))
				if err != nil {
					common.Log.Infof("Error deleting db %s: %v\n", fromKey, err)
				}
			}
		}
		toKey := GetHolderInfoKey(action.ToAddr, action.Ticker)
		value, ok := p.holderMap[action.ToAddr]
		if ok {
			err := db.SetDB([]byte(toKey), value, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", toKey, err)
			}
		} else {
			//已经被删除
			err := wb.Delete([]byte(toKey))
			if err != nil {
				common.Log.Infof("Error deleting db %s: %v\n", toKey, err)
			}
		}

		if action.Action == 2 {
			// 保存历史记录
			history := common.BRC20TransferHistory{
				Height:   action.Height,
				Utxo:     action.Utxo,
				NftId:    action.NftId,
				FromAddr: action.FromAddr,
				ToAddr:   action.ToAddr,
				Ticker:   action.Ticker,
				Amount:   action.Amount.String(),
			}

			key := GetTransferHistoryKey(action.Ticker, action.Utxo)
			err := db.SetDB([]byte(key), &history, wb)
			if err != nil {
				common.Log.Panicf("Error setting %s in db %v", key, err)
			}
		}
	}

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	}

	// reset memory buffer
	p.holderActionList = make([]*HolderAction, 0)
	p.tickerAdded = make(map[string]*common.BRC20Ticker)
	for _, info := range p.tickerMap {
		info.MintAdded = make([]*common.BRC20Mint, 0)
	}

	common.Log.Infof("BRC20Indexer->UpdateDB takse: %v", time.Since(startTime))
}
