package ft

import (
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func (p *FTIndexer) BuildFreezeAuthoritySnapshot() map[string]uint64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make(map[string]uint64)
	if p.nftIndexer == nil {
		return result
	}

	for ticker, tickInfo := range p.tickerMap {
		if tickInfo == nil || tickInfo.Ticker == nil || tickInfo.Ticker.Base == nil || tickInfo.Ticker.SelfMint != 100 {
			continue
		}
		deployNft := p.nftIndexer.GetNftWithId(tickInfo.Ticker.Base.Id)
		if deployNft == nil {
			continue
		}
		result[strings.ToLower(ticker)] = deployNft.OwnerAddressId
	}
	return result
}

func (p *FTIndexer) SetFreezeAuthoritySnapshot(snapshot map[string]uint64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.freezeAuthoritySnapshot = make(map[string]uint64, len(snapshot))
	for ticker, ownerAddressId := range snapshot {
		p.freezeAuthoritySnapshot[strings.ToLower(ticker)] = ownerAddressId
	}
}

func (p *FTIndexer) SetPendingHistoricalFreezeReplay(directives []*common.FreezeDirective) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.pendingHistoricalFreezes == nil {
		p.pendingHistoricalFreezes = make(map[int][]*common.FreezeDirective)
	}
	if p.pendingHistoricalKeys == nil {
		p.pendingHistoricalKeys = make(map[string]bool)
	}
	for _, item := range directives {
		key := common.FreezeDirectiveKey(item.TxId, item.Ticker, item.AddressId, item.FreezeHeight)
		if p.pendingHistoricalKeys[key] {
			continue
		}
		d := *item
		d.Ticker = strings.ToLower(d.Ticker)
		p.pendingHistoricalFreezes[d.FreezeHeight] = append(p.pendingHistoricalFreezes[d.FreezeHeight], &d)
		p.pendingHistoricalKeys[key] = true
	}
}

func (p *FTIndexer) ConsumeReloadRequest() (int, []*common.FreezeDirective) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.reloadRequestHeight == 0 || len(p.reloadFreezeDirectives) == 0 {
		return 0, nil
	}

	result := make([]*common.FreezeDirective, 0, len(p.reloadFreezeDirectives))
	for _, item := range p.reloadFreezeDirectives {
		d := *item
		result = append(result, &d)
	}
	height := p.reloadRequestHeight
	p.reloadRequestHeight = 0
	p.reloadFreezeDirectives = make(map[string]*common.FreezeDirective)
	return height, result
}

func (p *FTIndexer) isAddressFrozen(ticker string, addressId uint64) bool {
	stateMap, ok := p.freezeStates[ticker]
	if !ok {
		return false
	}
	_, ok = stateMap[addressId]
	return ok
}

func (p *FTIndexer) touchFreezeState(ticker string, addressId uint64, state *common.FreezeState) {
	if p.freezeTouched == nil {
		p.freezeTouched = make(map[string]*common.FreezeState)
	}
	key := common.FreezeStateMapKey(ticker, addressId)
	p.freezeTouched[key] = state
	if p.freezeDeleted != nil {
		delete(p.freezeDeleted, key)
	}
}

func (p *FTIndexer) setFreezeState(ticker string, addressId uint64, state *common.FreezeState) {
	stateMap, ok := p.freezeStates[ticker]
	if !ok {
		stateMap = make(map[uint64]*common.FreezeState)
		p.freezeStates[ticker] = stateMap
	}
	stateMap[addressId] = state
	p.touchFreezeState(ticker, addressId, state)
}

func (p *FTIndexer) clearFreezeState(ticker string, addressId uint64) {
	if stateMap, ok := p.freezeStates[ticker]; ok {
		delete(stateMap, addressId)
		if len(stateMap) == 0 {
			delete(p.freezeStates, ticker)
		}
	}
	if p.freezeDeleted == nil {
		p.freezeDeleted = make(map[string]*common.FreezeState)
	}
	key := common.FreezeStateMapKey(ticker, addressId)
	p.freezeDeleted[key] = &common.FreezeState{Ticker: ticker, AddressId: addressId}
	delete(p.freezeTouched, key)
}

func (p *FTIndexer) markAddressTickerFrozen(ticker string, addressId uint64, frozen bool) {
	utxos := p.utxoMap[ticker]
	for utxoId := range utxos {
		holder := p.holderInfo[utxoId]
		if holder == nil || holder.AddressId != addressId {
			continue
		}
		assetInfo := holder.Tickers[ticker]
		if assetInfo == nil || assetInfo.Frozen == frozen {
			continue
		}
		assetInfo.Frozen = frozen
		p.holderActionList = append(p.holderActionList, &HolderAction{
			UtxoId:    utxoId,
			AddressId: holder.AddressId,
			Tickers:   map[string]bool{ticker: true},
			Action:    1,
		})
	}
}

func (p *FTIndexer) applyFreezeDirective(directive *common.FreezeDirective) {
	if p.isAddressFrozen(directive.Ticker, directive.AddressId) {
		return
	}

	amount := p.getAddressTickerAmount(directive.AddressId, directive.Ticker)
	state := &common.FreezeState{
		Ticker:       directive.Ticker,
		AddressId:    directive.AddressId,
		FreezeHeight: directive.FreezeHeight,
		TxId:         directive.TxId,
	}
	p.addTickerFrozenAmount(directive.Ticker, amount)
	p.freezeHistory = append(p.freezeHistory, &common.FreezeHistory{
		Ticker:        directive.Ticker,
		AddressId:     directive.AddressId,
		TxId:          directive.TxId,
		Action:        common.FreezeActionFreeze,
		Amount:        amount,
		FreezeHeight:  directive.FreezeHeight,
		ConfirmHeight: directive.ConfirmHeight,
	})
	p.setFreezeState(directive.Ticker, directive.AddressId, state)
	p.markAddressTickerFrozen(directive.Ticker, directive.AddressId, true)
}

func (p *FTIndexer) activatePendingFreezesAtHeight(height int) {
	if len(p.pendingHistoricalFreezes) == 0 {
		return
	}
	directives := p.pendingHistoricalFreezes[height]
	if len(directives) == 0 {
		return
	}
	for _, item := range directives {
		p.applyFreezeDirective(item)
		delete(p.pendingHistoricalKeys, common.FreezeDirectiveKey(item.TxId, item.Ticker, item.AddressId, item.FreezeHeight))
	}
	delete(p.pendingHistoricalFreezes, height)
}

func (p *FTIndexer) registerReloadDirective(directive *common.FreezeDirective) {
	key := common.FreezeDirectiveKey(directive.TxId, directive.Ticker, directive.AddressId, directive.FreezeHeight)
	if p.pendingHistoricalKeys[key] {
		return
	}
	if p.reloadFreezeDirectives == nil {
		p.reloadFreezeDirectives = make(map[string]*common.FreezeDirective)
	}
	if _, ok := p.reloadFreezeDirectives[key]; ok {
		return
	}
	d := *directive
	p.reloadFreezeDirectives[key] = &d
	if p.reloadRequestHeight == 0 || directive.FreezeHeight < p.reloadRequestHeight {
		p.reloadRequestHeight = directive.FreezeHeight
	}
}

func (p *FTIndexer) canFreezeTicker(initiatorAddressId uint64, ticker string) bool {
	if initiatorAddressId == common.INVALID_ID {
		return false
	}
	if ownerAddressId, ok := p.freezeAuthoritySnapshot[ticker]; ok {
		return initiatorAddressId == ownerAddressId
	}

	tickInfo := p.tickerMap[ticker]
	if tickInfo == nil || tickInfo.Ticker == nil || tickInfo.Ticker.Base == nil {
		return false
	}
	if tickInfo.Ticker.SelfMint != 100 || p.nftIndexer == nil {
		return false
	}
	deployNft := p.nftIndexer.GetNftWithId(tickInfo.Ticker.Base.Id)
	if deployNft == nil {
		return false
	}
	return initiatorAddressId == deployNft.OwnerAddressId
}

func (p *FTIndexer) getAddressId(address string) uint64 {
	if p.nftIndexer == nil || p.nftIndexer.GetBaseIndexer() == nil {
		return common.INVALID_ID
	}
	return p.nftIndexer.GetBaseIndexer().GetAddressIdFromDB(address)
}

func (p *FTIndexer) getFreezeInitiatorAddressId(tx *common.Transaction) uint64 {
	if tx == nil || len(tx.Inputs) == 0 {
		return common.INVALID_ID
	}
	return tx.Inputs[len(tx.Inputs)-1].AddressId
}
