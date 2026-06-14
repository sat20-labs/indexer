package atom

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
)

func (s *Indexer) UpdateTransfer(block *common.Block) {
	if block.Height < s.heights.Activation {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.status.Height = block.Height

	for txIndex, tx := range block.Transactions {
		op := ParseOperation(tx, block.Height >= s.heights.Density)
		spent := s.collectInputBalances(tx)
		switch {
		case op != nil && op.InputIndex == 0 && op.Op == OpDirectFT:
			s.handleDirectFT(block, txIndex, tx, op)
		case op != nil && op.InputIndex == 0 && op.Op == OpDeployDFT:
			s.handleDeployDFT(block, txIndex, tx, op)
		case op != nil && op.InputIndex == 0 && op.Op == OpMintDFT:
			s.handleMintDFT(block, txIndex, tx, op)
		}
		s.applyTransfer(block, txIndex, tx, op, spent)
	}
	s.checkPointWithBlockHeightLocked(block.Height, time.Now())
}

type spentBalance struct {
	balance    *UtxoBalance
	inputIndex int
}

func (s *Indexer) collectInputBalances(tx *common.Transaction) []*spentBalance {
	result := make([]*spentBalance, 0)
	for inputIndex, input := range tx.Inputs {
		items := s.utxoBalances[input.UtxoId]
		if len(items) == 0 {
			continue
		}
		atomicalIds := make([]string, 0, len(items))
		for atomicalId := range items {
			atomicalIds = append(atomicalIds, atomicalId)
		}
		sortAtomicalIds(atomicalIds)
		for _, atomicalId := range atomicalIds {
			balance := items[atomicalId]
			result = append(result, &spentBalance{balance: balance.Clone(), inputIndex: inputIndex})
			s.removeUtxoBalanceInMemory(balance)
		}
	}
	return result
}

func (s *Indexer) handleDirectFT(block *common.Block, txIndex int, tx *common.Transaction, op *Operation) {
	tickerName := stringArg(op.Payload.Args, "request_ticker")
	if !isValidTicker(tickerName) || len(tx.Outputs) == 0 {
		return
	}
	tickerName = strings.ToLower(tickerName)
	tickerId, ok := s.prepareTickerRegistration(tickerName, tx.Inputs[op.InputIndex], op.CommitIndex)
	if !ok {
		return
	}
	if boolArg(op.Payload.Args, "i") || boolArg(op.Payload.Args, "$immutable") {
		return
	}
	if !s.validMintEnvelope(block.Height, tx, op, true) {
		return
	}
	amount := tx.Outputs[0].OutValue.Value
	if amount <= 0 {
		return
	}
	atomicalId := compactId(op.CommitTxId, op.CommitIndex)
	locationId := compactId(tx.TxId, 0)
	ticker := &Ticker{
		Id:            tickerId,
		AtomicalId:    atomicalId,
		LocationId:    locationId,
		Name:          tickerName,
		DisplayName:   tickerName,
		Subtype:       "direct",
		MintMode:      "fixed",
		MaxSupply:     amount,
		MintedAmount:  amount,
		MintedTimes:   1,
		DeployHeight:  block.Height,
		DeployTime:    block.Timestamp.Unix(),
		DeployTx:      tx.TxId,
		DeployIndex:   txIndex,
		CommitTx:      op.CommitTxId,
		CommitTxIndex: tx.Inputs[op.InputIndex].OutTxIndex,
		CommitIndex:   op.CommitIndex,
		CommitHeight:  tx.Inputs[op.InputIndex].OutHeight,
		Bitworkc:      stringArg(op.Payload.Args, "bitworkc"),
		Bitworkr:      stringArg(op.Payload.Args, "bitworkr"),
	}
	s.addTicker(ticker)
	s.addMint(block, txIndex, tx, tickerName, atomicalId, locationId, amount, tx.Outputs[0])
}

func (s *Indexer) handleDeployDFT(block *common.Block, txIndex int, tx *common.Transaction, op *Operation) {
	tickerName := stringArg(op.Payload.Args, "request_ticker")
	if !isValidTicker(tickerName) {
		return
	}
	tickerName = strings.ToLower(tickerName)
	tickerId, ok := s.prepareTickerRegistration(tickerName, tx.Inputs[op.InputIndex], op.CommitIndex)
	if !ok {
		return
	}
	if !s.validMintEnvelope(block.Height, tx, op, true) {
		return
	}
	mintHeight, ok := intArg(op.Payload.Args, "mint_height")
	if !ok || mintHeight < DFTMintHeightMin || mintHeight > DFTMintHeightMax {
		return
	}
	mintAmount, ok := intArg(op.Payload.Args, "mint_amount")
	if !ok || mintAmount < DFTMintAmountMin || mintAmount > DFTMintAmountMax {
		return
	}
	maxMints, ok := intArg(op.Payload.Args, "max_mints")
	if !ok || maxMints < DFTMaxMintsMin {
		return
	}
	if block.Height < s.heights.Density && maxMints > DFTMaxMintsLegacy {
		return
	}
	if block.Height >= s.heights.Density && maxMints > DFTMaxMintsDensity {
		return
	}
	mintMode := "fixed"
	maxSupply := mintAmount * maxMints
	md, hasMd := intArg(op.Payload.Args, "md")
	if hasMd && md != 0 && md != 1 {
		return
	}
	if mintBitworkc := stringArg(op.Payload.Args, "mint_bitworkc"); mintBitworkc != "" {
		if _, _, ok := parseBitwork(mintBitworkc); !ok {
			return
		}
	}
	if mintBitworkr := stringArg(op.Payload.Args, "mint_bitworkr"); mintBitworkr != "" {
		if _, _, ok := parseBitwork(mintBitworkr); !ok {
			return
		}
	}
	maxg, hasMaxg := intArg(op.Payload.Args, "maxg")
	var bcs, brs int64
	if block.Height >= s.heights.Density && md == 1 {
		bv := stringArg(op.Payload.Args, "bv")
		bci, _ := intArg(op.Payload.Args, "bci")
		bri, _ := intArg(op.Payload.Args, "bri")
		var okBcs, okBrs bool
		bcs, okBcs = intArg(op.Payload.Args, "bcs")
		brs, okBrs = intArg(op.Payload.Args, "brs")
		if !okBcs {
			bcs = 64
		}
		if !okBrs {
			brs = 64
		}
		if bv == "" || len(bv) < 4 || !hexPattern.MatchString(bv) || (bci == 0 && bri == 0) {
			return
		}
		if stringArg(op.Payload.Args, "mint_bitworkc") != "" || stringArg(op.Payload.Args, "mint_bitworkr") != "" {
			return
		}
		if bci < 0 || bci > 64 || bri < 0 || bri > 64 {
			return
		}
		if bci > 0 && (bcs < 64 || bcs > 256) {
			return
		}
		if bri > 0 && (brs < 64 || brs > 256) {
			return
		}
		if maxMints > 100000 {
			return
		}
		if hasMaxg && (maxg < DFTMaxMintsMin || maxg > DFTMaxMintsDensity) {
			return
		}
		mintMode = "perpetual"
		if hasMaxg {
			maxSupply = mintAmount * maxg
		} else {
			maxg = 0
			maxSupply = -1
		}
	}
	atomicalId := compactId(op.CommitTxId, op.CommitIndex)
	ticker := &Ticker{
		Id:             tickerId,
		AtomicalId:     atomicalId,
		LocationId:     compactId(tx.TxId, 0),
		Name:           tickerName,
		DisplayName:    tickerName,
		Subtype:        "decentralized",
		MintMode:       mintMode,
		MintAmount:     mintAmount,
		MintHeight:     mintHeight,
		MaxMints:       maxMints,
		MaxMintsGlobal: maxg,
		MaxSupply:      maxSupply,
		DeployHeight:   block.Height,
		DeployTime:     block.Timestamp.Unix(),
		DeployTx:       tx.TxId,
		DeployIndex:    txIndex,
		CommitTx:       op.CommitTxId,
		CommitTxIndex:  tx.Inputs[op.InputIndex].OutTxIndex,
		CommitIndex:    op.CommitIndex,
		CommitHeight:   tx.Inputs[op.InputIndex].OutHeight,
		Bitworkc:       stringArg(op.Payload.Args, "bitworkc"),
		Bitworkr:       stringArg(op.Payload.Args, "bitworkr"),
		MintBitworkc:   stringArg(op.Payload.Args, "mint_bitworkc"),
		MintBitworkr:   stringArg(op.Payload.Args, "mint_bitworkr"),
		Bv:             stringArg(op.Payload.Args, "bv"),
	}
	ticker.Bci, _ = intArg(op.Payload.Args, "bci")
	ticker.Bri, _ = intArg(op.Payload.Args, "bri")
	ticker.Bcs = bcs
	ticker.Brs = brs
	s.addTicker(ticker)
}

func (s *Indexer) handleMintDFT(block *common.Block, txIndex int, tx *common.Transaction, op *Operation) {
	if len(tx.Outputs) == 0 {
		return
	}
	tickerName := stringArg(op.Payload.Args, "mint_ticker")
	if !isValidTicker(tickerName) {
		return
	}
	tickerName = strings.ToLower(tickerName)
	ticker := s.getTickerLocked(tickerName)
	if ticker == nil || ticker.Subtype != "decentralized" {
		return
	}
	if !tickerEffectiveAtHeight(block.Height, ticker) {
		return
	}
	if block.Height < int(ticker.MintHeight) || tx.Outputs[0].OutValue.Value != ticker.MintAmount {
		return
	}
	actualMints := s.dftMintedTimesLocked(tickerName, ticker)
	if ticker.MintMode == "fixed" && actualMints >= ticker.MaxMints {
		return
	}
	if ticker.MintMode == "perpetual" && ticker.MaxMintsGlobal > 0 && actualMints >= ticker.MaxMintsGlobal {
		return
	}
	if !s.validDftMintBitwork(block.Height, tx, op, ticker, actualMints) {
		return
	}
	locationId := compactId(tx.TxId, 0)
	ticker.MintedTimes = actualMints + 1
	ticker.MintedAmount = ticker.MintedTimes * ticker.MintAmount
	s.touchTicker(ticker)
	s.addMint(block, txIndex, tx, tickerName, ticker.AtomicalId, locationId, ticker.MintAmount, tx.Outputs[0])
}

func tickerEffectiveAtHeight(height int, ticker *Ticker) bool {
	if ticker == nil || ticker.CommitHeight <= 0 {
		return true
	}
	return ticker.CommitHeight <= height-MintTickerDelayBlocks
}

func (s *Indexer) validMintEnvelope(height int, tx *common.Transaction, op *Operation, requireBitworkc bool) bool {
	if op.InputIndex >= len(tx.Inputs) {
		return false
	}
	input := tx.Inputs[op.InputIndex]
	if input.OutHeight < s.heights.Activation {
		return false
	}
	if input.OutHeight < height-MintGeneralDelayBlocks || input.OutHeight < height-MintTickerDelayBlocks {
		return false
	}
	if height >= s.heights.Commitz && op.CommitIndex != 0 {
		return false
	}
	bitworkc := stringArg(op.Payload.Args, "bitworkc")
	if requireBitworkc && bitworkc == "" {
		return false
	}
	if bitworkc != "" {
		prefix, _, ok := parseBitwork(bitworkc)
		if !ok || (requireBitworkc && len(prefix) < 4) || !isBitworkMatch(op.CommitTxId, bitworkc) {
			return false
		}
	}
	bitworkr := stringArg(op.Payload.Args, "bitworkr")
	if bitworkr != "" && !isBitworkMatch(tx.TxId, bitworkr) {
		return false
	}
	return true
}

func (s *Indexer) dftMintedTimesLocked(tickerName string, ticker *Ticker) int64 {
	actualMints := ticker.MintedTimes
	if historyMints := int64(len(s.mintHistory[strings.ToLower(tickerName)])); historyMints > actualMints {
		actualMints = historyMints
	}
	return actualMints
}

func (s *Indexer) validDftMintBitwork(height int, tx *common.Transaction, op *Operation, ticker *Ticker, actualMints int64) bool {
	if op.InputIndex >= len(tx.Inputs) {
		return false
	}
	input := tx.Inputs[op.InputIndex]
	if input.OutHeight < s.heights.Activation || input.OutHeight < int(ticker.MintHeight) {
		return false
	}
	if height >= s.heights.Commitz && op.CommitIndex != 0 {
		return false
	}
	if ticker.MintMode == "perpetual" {
		allowHigher := height >= s.heights.Rollover
		if ticker.Bci > 0 && !isPerpetualBitworkMatch(op.CommitTxId, ticker.Bv, actualMints, ticker.MaxMints, ticker.Bci, ticker.Bcs, allowHigher) {
			return false
		}
		if ticker.Bri > 0 && !isPerpetualBitworkMatch(tx.TxId, ticker.Bv, actualMints, ticker.MaxMints, ticker.Bri, ticker.Brs, allowHigher) {
			return false
		}
		return true
	}
	if ticker.MintBitworkc != "" && !isBitworkMatch(op.CommitTxId, ticker.MintBitworkc) {
		return false
	}
	if ticker.MintBitworkr != "" && !isBitworkMatch(tx.TxId, ticker.MintBitworkr) {
		return false
	}
	return true
}

func (s *Indexer) applyTransfer(block *common.Block, txIndex int, tx *common.Transaction, op *Operation, spent []*spentBalance) {
	if len(spent) == 0 {
		return
	}
	grouped := make(map[string][]*UtxoBalance)
	for _, balance := range spent {
		grouped[balance.balance.AtomicalId] = append(grouped[balance.balance.AtomicalId], balance.balance)
	}
	customActivated := block.Height >= s.heights.CustomColoring
	isSplit := op != nil && op.Op == OpSplit
	isCustom := op != nil && op.Op == OpCustomColor && customActivated
	if !isSplit && !isCustom {
		atomicalIds := orderedTransferAtomicalIds(grouped, spent, block.Height >= s.heights.Dmint)
		s.applyRegularTransfer(block, txIndex, tx, grouped, atomicalIds)
		return
	}
	atomicalIds := sortedAtomicalIds(grouped)
	startOutput := 0
	for _, atomicalId := range atomicalIds {
		amount, ticker := groupedAmount(grouped[atomicalId])
		assignments, _ := s.assignRegular(tx, startOutput, amount, customActivated)
		if isSplit && spentFromInput(spent, atomicalId, op.InputIndex) {
			assignments = s.assignSplit(tx, atomicalId, amount, op, customActivated)
		}
		if isCustom {
			assignments = s.assignCustom(tx, atomicalId, amount, op)
			if len(assignments) == 0 {
				assignments, _ = s.assignRegular(tx, startOutput, amount, customActivated)
			}
		}
		if len(assignments) > 0 {
			startOutput = assignments[len(assignments)-1].OutputIndex + 1
		}
		s.addAssignments(block, txIndex, tx, atomicalId, ticker, grouped[atomicalId], assignments)
	}
}

func spentFromInput(spent []*spentBalance, atomicalId string, inputIndex int) bool {
	for _, item := range spent {
		if item.inputIndex == inputIndex && item.balance.AtomicalId == atomicalId {
			return true
		}
	}
	return false
}

func orderedTransferAtomicalIds(grouped map[string][]*UtxoBalance, spent []*spentBalance, fifo bool) []string {
	if !fifo {
		return sortedAtomicalIds(grouped)
	}
	result := make([]string, 0, len(grouped))
	seen := make(map[string]bool, len(grouped))
	for _, balance := range spent {
		atomicalId := balance.balance.AtomicalId
		if seen[atomicalId] {
			continue
		}
		seen[atomicalId] = true
		result = append(result, atomicalId)
	}
	return result
}

func sortedAtomicalIds(grouped map[string][]*UtxoBalance) []string {
	result := make([]string, 0, len(grouped))
	for id := range grouped {
		result = append(result, id)
	}
	sortAtomicalIds(result)
	return result
}

func sortAtomicalIds(ids []string) {
	sort.Slice(ids, func(i, j int) bool {
		return compareAtomicalIds(ids[i], ids[j]) < 0
	})
}

func compareAtomicalIds(a, b string) int {
	aKey, aOK := atomicalIdSortKey(a)
	bKey, bOK := atomicalIdSortKey(b)
	if aOK && bOK {
		return bytes.Compare(aKey, bKey)
	}
	if aOK != bOK {
		if aOK {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

func atomicalIdSortKey(id string) ([]byte, bool) {
	index := strings.IndexByte(id, 'i')
	if index != 64 {
		return nil, false
	}
	rawHash, err := hex.DecodeString(id[:64])
	if err != nil || len(rawHash) != 32 {
		return nil, false
	}
	output, err := strconv.ParseUint(id[index+1:], 10, 32)
	if err != nil {
		return nil, false
	}
	key := make([]byte, 36)
	for i := 0; i < 32; i++ {
		key[i] = rawHash[31-i]
	}
	binary.LittleEndian.PutUint32(key[32:], uint32(output))
	return key, true
}

func (s *Indexer) applyRegularTransfer(block *common.Block, txIndex int, tx *common.Transaction, grouped map[string][]*UtxoBalance, atomicalIds []string) {
	customActivated := block.Height >= s.heights.CustomColoring
	assignmentsByAtomical := make(map[string][]assignment)
	startOutput := 0
	clean := true
	for _, atomicalId := range atomicalIds {
		amount, _ := groupedAmount(grouped[atomicalId])
		assignments, ok := s.assignRegular(tx, startOutput, amount, customActivated)
		if !ok && (!customActivated || len(assignments) == 0) {
			clean = false
			break
		}
		assignmentsByAtomical[atomicalId] = assignments
		if len(assignments) > 0 {
			startOutput = assignments[len(assignments)-1].OutputIndex + 1
		}
	}
	if !clean {
		assignmentsByAtomical = make(map[string][]assignment)
		for _, atomicalId := range atomicalIds {
			amount, _ := groupedAmount(grouped[atomicalId])
			assignments, _ := s.assignRegular(tx, 0, amount, customActivated)
			assignmentsByAtomical[atomicalId] = assignments
		}
	}
	for _, atomicalId := range atomicalIds {
		_, ticker := groupedAmount(grouped[atomicalId])
		s.addAssignments(block, txIndex, tx, atomicalId, ticker, grouped[atomicalId], assignmentsByAtomical[atomicalId])
	}
}

func groupedAmount(items []*UtxoBalance) (int64, string) {
	var amount int64
	var ticker string
	for _, item := range items {
		amount += item.Amount
		ticker = item.Ticker
	}
	return amount, ticker
}

func (s *Indexer) addAssignments(block *common.Block, txIndex int, tx *common.Transaction, atomicalId, ticker string, sources []*UtxoBalance, assignments []assignment) {
	fromUtxo, fromAddr := actionSource(sources)
	for _, a := range assignments {
		if a.Amount <= 0 || a.OutputIndex >= len(tx.Outputs) {
			continue
		}
		output := tx.Outputs[a.OutputIndex]
		if isUnspendable(output) {
			continue
		}
		s.addUtxoBalanceInMemory(&UtxoBalance{
			UtxoId:     output.UtxoId,
			AddressId:  output.AddressId,
			Outpoint:   output.OutPointStr,
			AtomicalId: atomicalId,
			Ticker:     ticker,
			Amount:     a.Amount,
		})
		s.recordAction(block, txIndex, tx.TxId, ticker, atomicalId, fromUtxo, output.UtxoId, fromAddr, output.AddressId, a.Amount, "transfer")
	}
}

func actionSource(sources []*UtxoBalance) (uint64, uint64) {
	if len(sources) != 1 {
		return 0, 0
	}
	return sources[0].UtxoId, sources[0].AddressId
}

type assignment struct {
	OutputIndex int
	Amount      int64
}

func (s *Indexer) assignRegular(tx *common.Transaction, start int, amount int64, customActivated bool) ([]assignment, bool) {
	result := make([]assignment, 0)
	remaining := amount
	for i := start; i < len(tx.Outputs) && remaining > 0; i++ {
		output := tx.Outputs[i]
		if isUnspendable(output) {
			continue
		}
		value := output.OutValue.Value
		if value <= 0 {
			continue
		}
		if !customActivated && value > remaining {
			return result, false
		}
		assign := value
		if assign > remaining {
			assign = remaining
		}
		result = append(result, assignment{OutputIndex: i, Amount: assign})
		remaining -= assign
		if !customActivated && assign < value {
			return nil, false
		}
	}
	return result, remaining == 0
}

func (s *Indexer) assignSplit(tx *common.Transaction, atomicalId string, amount int64, op *Operation, customActivated bool) []assignment {
	skip, _ := intArg(op.Payload.Args, atomicalId)
	result := make([]assignment, 0)
	remaining := amount
	var skipped int64
	for i, output := range tx.Outputs {
		if skip > 0 && skipped < skip {
			skipped += output.OutValue.Value
			continue
		}
		value := output.OutValue.Value
		if !customActivated && value > remaining {
			break
		}
		assign := value
		if assign > remaining {
			assign = remaining
		}
		if assign <= 0 {
			break
		}
		result = append(result, assignment{OutputIndex: i, Amount: assign})
		remaining -= assign
		if remaining == 0 {
			break
		}
	}
	return result
}

func (s *Indexer) assignCustom(tx *common.Transaction, atomicalId string, amount int64, op *Operation) []assignment {
	raw, ok := op.Payload.Args[atomicalId]
	if !ok {
		return nil
	}
	outMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	result := make([]assignment, 0)
	remaining := amount
	for i := range tx.Outputs {
		value, ok := intArg(outMap, fmt.Sprintf("%d", i))
		if !ok || value <= 0 || remaining <= 0 {
			continue
		}
		if value > tx.Outputs[i].OutValue.Value {
			value = tx.Outputs[i].OutValue.Value
		}
		if value > remaining {
			value = remaining
		}
		result = append(result, assignment{OutputIndex: i, Amount: value})
		remaining -= value
	}
	return result
}

func isUnspendable(output *common.TxOutputV2) bool {
	if output == nil || len(output.OutValue.PkScript) == 0 {
		return false
	}
	script := output.OutValue.PkScript
	return script[0] == txscript.OP_RETURN || (len(script) >= 2 && script[0] == txscript.OP_FALSE && script[1] == txscript.OP_RETURN)
}

func (s *Indexer) prepareTickerRegistration(name string, input *common.TxInput, commitIndex int) (int64, bool) {
	existing := s.getTickerLocked(name)
	if existing == nil {
		return s.status.TickerCount, true
	}
	if input == nil || !candidateTickerPrecedes(existing, input, commitIndex) {
		return 0, false
	}
	s.removeTickerCandidateInMemory(existing)
	return existing.Id, true
}

func candidateTickerPrecedes(existing *Ticker, input *common.TxInput, commitIndex int) bool {
	if input.OutHeight != existing.CommitHeight {
		return input.OutHeight < existing.CommitHeight
	}
	if input.OutTxIndex != existing.CommitTxIndex {
		return input.OutTxIndex < existing.CommitTxIndex
	}
	return commitIndex < existing.CommitIndex
}

func (s *Indexer) removeTickerCandidateInMemory(ticker *Ticker) {
	if ticker == nil {
		return
	}
	name := strings.ToLower(ticker.Name)
	delete(s.tickerMap, name)
	delete(s.tickerTouched, name)
	if ticker.Subtype == "direct" {
		s.removeDirectTickerMintInMemory(name, ticker.AtomicalId)
	}
}

func (s *Indexer) removeDirectTickerMintInMemory(ticker, atomicalId string) {
	items := s.mintHistory[ticker]
	for i, mint := range items {
		if mint.AtomicalId != atomicalId {
			continue
		}
		balance := &UtxoBalance{
			UtxoId:     mint.UtxoId,
			AddressId:  mint.AddressId,
			Outpoint:   mint.Outpoint,
			AtomicalId: mint.AtomicalId,
			Ticker:     mint.Ticker,
			Amount:     mint.Amount,
		}
		s.removeUtxoBalanceInMemory(balance)
		s.mintHistory[ticker] = append(items[:i], items[i+1:]...)
		break
	}
	for i := 0; i < len(s.mintsAdded); i++ {
		if s.mintsAdded[i].AtomicalId == atomicalId {
			s.mintsAdded = append(s.mintsAdded[:i], s.mintsAdded[i+1:]...)
			i--
		}
	}
	for i := 0; i < len(s.actionsAdded); i++ {
		if s.actionsAdded[i].AtomicalId == atomicalId && s.actionsAdded[i].Action == "mint" {
			s.actionsAdded = append(s.actionsAdded[:i], s.actionsAdded[i+1:]...)
			i--
		}
	}
}

func (s *Indexer) addTicker(ticker *Ticker) {
	name := strings.ToLower(ticker.Name)
	s.tickerMap[name] = ticker
	s.tickerById[ticker.Id] = name
	s.tickerTouched[name] = ticker.Clone()
	s.tickerIdAdded[ticker.Id] = name
	if ticker.Id >= s.status.TickerCount {
		s.status.TickerCount = ticker.Id + 1
	}
}

func (s *Indexer) touchTicker(ticker *Ticker) {
	name := strings.ToLower(ticker.Name)
	s.tickerMap[name] = ticker
	s.tickerTouched[name] = ticker.Clone()
}

func (s *Indexer) addMint(block *common.Block, txIndex int, tx *common.Transaction, ticker, atomicalId, locationId string, amount int64, output *common.TxOutputV2) {
	if output == nil {
		return
	}
	mint := &MintInfo{
		Id:         s.status.MintCount,
		AtomicalId: atomicalId,
		LocationId: locationId,
		Ticker:     ticker,
		AddressId:  output.AddressId,
		UtxoId:     output.UtxoId,
		Outpoint:   output.OutPointStr,
		Amount:     amount,
		Height:     block.Height,
		TxIndex:    txIndex,
		TxId:       tx.TxId,
	}
	s.status.MintCount++
	s.mintHistory[ticker] = append(s.mintHistory[ticker], mint)
	s.mintsAdded = append(s.mintsAdded, mint.Clone())
	s.addUtxoBalanceInMemory(&UtxoBalance{
		UtxoId:     output.UtxoId,
		AddressId:  output.AddressId,
		Outpoint:   output.OutPointStr,
		AtomicalId: atomicalId,
		Ticker:     ticker,
		Amount:     amount,
	})
	s.recordAction(block, txIndex, tx.TxId, ticker, atomicalId, 0, output.UtxoId, 0, output.AddressId, amount, "mint")
}

func (s *Indexer) recordAction(block *common.Block, txIndex int, txid, ticker, atomicalId string, fromUtxo, toUtxo, fromAddr, toAddr uint64, amount int64, action string) {
	if amount <= 0 {
		return
	}
	history := &ActionHistory{
		Id:         s.status.ActionCount,
		Height:     block.Height,
		TxIndex:    txIndex,
		TxId:       txid,
		Ticker:     strings.ToLower(ticker),
		AtomicalId: atomicalId,
		FromUtxo:   fromUtxo,
		ToUtxo:     toUtxo,
		FromAddr:   fromAddr,
		ToAddr:     toAddr,
		Amount:     amount,
		Action:     action,
	}
	s.status.ActionCount++
	s.actionsAdded = append(s.actionsAdded, history)
}

func (s *Indexer) addUtxoBalanceInMemory(balance *UtxoBalance) {
	if balance == nil || balance.Amount <= 0 {
		return
	}
	ticker := strings.ToLower(balance.Ticker)
	balance.Ticker = ticker
	key := GetUtxoBalanceKey(balance.UtxoId, balance.AtomicalId)
	if _, ok := s.utxoBalances[balance.UtxoId]; !ok {
		s.utxoBalances[balance.UtxoId] = make(map[string]*UtxoBalance)
	}
	if existing := s.utxoBalances[balance.UtxoId][balance.AtomicalId]; existing != nil {
		s.removeUtxoBalanceFromIndexes(existing, true)
		delete(s.utxoDeleted, key)
	}
	if _, ok := s.utxoBalances[balance.UtxoId]; !ok {
		s.utxoBalances[balance.UtxoId] = make(map[string]*UtxoBalance)
	}
	s.utxoBalances[balance.UtxoId][balance.AtomicalId] = balance.Clone()
	if _, ok := s.tickerUtxos[ticker]; !ok {
		s.tickerUtxos[ticker] = make(map[uint64]int64)
	}
	s.tickerUtxos[ticker][balance.UtxoId] += balance.Amount
	if _, ok := s.holderBalances[balance.AddressId]; !ok {
		s.holderBalances[balance.AddressId] = make(map[string]int64)
	}
	s.holderBalances[balance.AddressId][ticker] += balance.Amount
	if _, ok := s.tickerHolders[ticker]; !ok {
		s.tickerHolders[ticker] = make(map[uint64]int64)
	}
	s.tickerHolders[ticker][balance.AddressId] += balance.Amount
	if t := s.getTickerLocked(ticker); t != nil {
		t.HolderCount = len(s.tickerHolders[ticker])
		s.touchTicker(t)
	}
	s.utxoTouched[key] = balance.Clone()
	s.holderTouched[GetHolderAssetKey(balance.AddressId, ticker)] = s.holderBalances[balance.AddressId][ticker]
	s.holderTouched[GetTickerHolderKey(ticker, balance.AddressId)] = s.tickerHolders[ticker][balance.AddressId]
}

func (s *Indexer) addLoadedUtxoBalanceInMemory(balance *UtxoBalance) {
	if balance == nil || balance.Amount <= 0 {
		return
	}
	ticker := strings.ToLower(balance.Ticker)
	balance.Ticker = ticker
	if _, ok := s.utxoBalances[balance.UtxoId]; !ok {
		s.utxoBalances[balance.UtxoId] = make(map[string]*UtxoBalance)
	}
	if existing := s.utxoBalances[balance.UtxoId][balance.AtomicalId]; existing != nil {
		s.removeUtxoBalanceFromIndexes(existing, false)
	}
	if _, ok := s.utxoBalances[balance.UtxoId]; !ok {
		s.utxoBalances[balance.UtxoId] = make(map[string]*UtxoBalance)
	}
	s.utxoBalances[balance.UtxoId][balance.AtomicalId] = balance.Clone()
	if _, ok := s.tickerUtxos[ticker]; !ok {
		s.tickerUtxos[ticker] = make(map[uint64]int64)
	}
	s.tickerUtxos[ticker][balance.UtxoId] += balance.Amount
	if _, ok := s.holderBalances[balance.AddressId]; !ok {
		s.holderBalances[balance.AddressId] = make(map[string]int64)
	}
	s.holderBalances[balance.AddressId][ticker] += balance.Amount
	if _, ok := s.tickerHolders[ticker]; !ok {
		s.tickerHolders[ticker] = make(map[uint64]int64)
	}
	s.tickerHolders[ticker][balance.AddressId] += balance.Amount
	if t := s.getTickerLocked(ticker); t != nil {
		t.HolderCount = len(s.tickerHolders[ticker])
	}
}

func (s *Indexer) removeUtxoBalanceInMemory(balance *UtxoBalance) {
	if balance == nil {
		return
	}
	s.removeUtxoBalanceFromIndexes(balance, true)
	key := GetUtxoBalanceKey(balance.UtxoId, balance.AtomicalId)
	s.utxoDeleted[key] = balance.Clone()
	delete(s.utxoTouched, key)
}

func (s *Indexer) removeUtxoBalanceFromIndexes(balance *UtxoBalance, trackPending bool) {
	ticker := strings.ToLower(balance.Ticker)
	if items := s.utxoBalances[balance.UtxoId]; items != nil {
		delete(items, balance.AtomicalId)
		if len(items) == 0 {
			delete(s.utxoBalances, balance.UtxoId)
		}
	}
	if items := s.tickerUtxos[ticker]; items != nil {
		items[balance.UtxoId] -= balance.Amount
		if items[balance.UtxoId] <= 0 {
			delete(items, balance.UtxoId)
		}
	}
	if items := s.holderBalances[balance.AddressId]; items != nil {
		items[ticker] -= balance.Amount
		if items[ticker] <= 0 {
			delete(items, ticker)
		}
	}
	if items := s.tickerHolders[ticker]; items != nil {
		items[balance.AddressId] -= balance.Amount
		if items[balance.AddressId] <= 0 {
			delete(items, balance.AddressId)
		}
	}
	if t := s.getTickerLocked(ticker); t != nil {
		t.HolderCount = len(s.tickerHolders[ticker])
		if trackPending {
			s.touchTicker(t)
		}
	}
	if !trackPending {
		return
	}
	s.holderTouched[GetHolderAssetKey(balance.AddressId, ticker)] = s.holderBalances[balance.AddressId][ticker]
	s.holderTouched[GetTickerHolderKey(ticker, balance.AddressId)] = s.tickerHolders[ticker][balance.AddressId]
}
