package indexer

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	atomidx "github.com/sat20-labs/indexer/indexer/atom"
	"github.com/sat20-labs/indexer/indexer/ord"
	"github.com/sat20-labs/indexer/indexer/ord/ord0_14_1"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

type mempoolUtxoState uint8

const (
	mempoolUtxoUnknown mempoolUtxoState = iota
	mempoolUtxoPlain
	mempoolUtxoNonPlain
)

type mempoolResolveStatus uint8

const (
	mempoolResolveComplete mempoolResolveStatus = iota
	mempoolResolvePending
	mempoolResolveBlocked
)

type mempoolResolvedInput struct {
	output    *common.TxOutput
	confirmed bool
	index     int
}

// resolveMempoolInputs deliberately never walks to an unconfirmed parent.
// Inputs are analyzable only when they are either confirmed in the indexer or
// are outputs already classified as plain by MiniMemPool. A known non-plain
// mempool input permanently blocks output classification for this transaction.
func (p *MiniMemPool) resolveMempoolInputs(tx *wire.MsgTx) ([]*mempoolResolvedInput, mempoolResolveStatus) {
	inputs := make([]*mempoolResolvedInput, 0, len(tx.TxIn))
	status := mempoolResolveComplete

	for i, txIn := range tx.TxIn {
		if txIn.PreviousOutPoint.Index == wire.MaxPrevOutIndex {
			continue
		}
		outpoint := txIn.PreviousOutPoint.String()
		parentID := txIn.PreviousOutPoint.Hash.String()

		p.mutex.RLock()
		parentTx, parentInPool := p.txMap[parentID]
		state, stateKnown := p.utxoStateMap[outpoint]
		plainOutput := p.knownPlainUtxoMap[outpoint]
		parentClassified := p.classifiedTxMap[parentID]
		p.mutex.RUnlock()

		if parentInPool {
			var placeholder *common.TxOutput
			vout := int(txIn.PreviousOutPoint.Index)
			if vout >= 0 && vout < len(parentTx.TxOut) {
				placeholder = common.GenerateTxOutput(parentTx, vout)
			}

			switch {
			case stateKnown && state == mempoolUtxoPlain && plainOutput != nil:
				inputs = append(inputs, &mempoolResolvedInput{output: plainOutput.Clone(), confirmed: false, index: i})
			case stateKnown && state == mempoolUtxoNonPlain:
				inputs = append(inputs, &mempoolResolvedInput{output: placeholder, confirmed: false, index: i})
				status = mempoolResolveBlocked
			case parentClassified:
				inputs = append(inputs, &mempoolResolvedInput{output: placeholder, confirmed: false, index: i})
				status = mempoolResolveBlocked
			default:
				inputs = append(inputs, &mempoolResolvedInput{output: placeholder, confirmed: false, index: i})
				if status != mempoolResolveBlocked {
					status = mempoolResolvePending
				}
			}
			continue
		}

		info := instance.GetTxOutputWithUtxoV2(outpoint, true)
		if info == nil {
			inputs = append(inputs, &mempoolResolvedInput{index: i})
			if status != mempoolResolveBlocked {
				status = mempoolResolvePending
			}
			continue
		}
		inputs = append(inputs, &mempoolResolvedInput{output: info, confirmed: true, index: i})
	}

	return inputs, status
}

func (p *MiniMemPool) allocateKnownMempoolTx(tx *wire.MsgTx, inputs []*mempoolResolvedInput) ([]*common.TxOutput, []bool, bool) {
	outputs, occupied, ok := allocateBoundMempoolOutputs(tx, inputs)
	if !ok {
		return nil, nil, false
	}

	runeOccupied, ok := allocateMempoolRunes(tx, inputs)
	if !ok {
		return nil, nil, false
	}
	for i := range occupied {
		occupied[i] = occupied[i] || runeOccupied[i]
	}

	atomOccupied, ok := allocateMempoolAtom(tx, inputs)
	if !ok {
		return nil, nil, false
	}
	for i := range occupied {
		occupied[i] = occupied[i] || atomOccupied[i]
	}

	markMempoolInscriptionOutputs(tx, inputs, occupied)
	return outputs, occupied, true
}

func allocateBoundMempoolOutputs(tx *wire.MsgTx, inputs []*mempoolResolvedInput) ([]*common.TxOutput, []bool, bool) {
	aggregate := common.NewTxOutput(0)
	for _, resolved := range inputs {
		if resolved == nil || resolved.output == nil {
			return nil, nil, false
		}
		input := resolved.output.Clone()
		remove := make([]common.AssetName, 0)
		for _, asset := range input.Assets {
			switch asset.Name.Protocol {
			case common.PROTOCOL_NAME_RUNES, common.PROTOCOL_NAME_ATOM:
				remove = append(remove, asset.Name)
			}
		}
		for _, name := range remove {
			input.RemoveAsset(&name)
		}

		for _, asset := range input.Assets {
			if asset.BindingSat == 0 && len(input.Offsets[asset.Name]) == 0 {
				return nil, nil, false
			}
		}
		if err := aggregate.Append(input); err != nil {
			return nil, nil, false
		}
	}

	outputs := make([]*common.TxOutput, len(tx.TxOut))
	occupied := make([]bool, len(tx.TxOut))
	remaining := aggregate
	for i, txOut := range tx.TxOut {
		if remaining == nil {
			if txOut.Value != 0 {
				return nil, nil, false
			}
			part := common.NewTxOutput(0)
			part.OutValue.PkScript = txOut.PkScript
			part.OutPointStr = fmt.Sprintf("%s:%d", tx.TxID(), i)
			outputs[i] = part
			continue
		}
		part, rest, err := remaining.Cut(txOut.Value)
		if err != nil || part == nil {
			return nil, nil, false
		}
		part.OutValue.PkScript = txOut.PkScript
		part.OutPointStr = fmt.Sprintf("%s:%d", tx.TxID(), i)
		outputs[i] = part
		occupied[i] = part.HasAsset()
		remaining = rest
	}
	return outputs, occupied, true
}

func allocateMempoolRunes(tx *wire.MsgTx, inputs []*mempoolResolvedInput) ([]bool, bool) {
	occupied := make([]bool, len(tx.TxOut))
	balances := make(map[runestone.RuneId]uint128.Uint128)
	for _, resolved := range inputs {
		if resolved == nil || !resolved.confirmed || resolved.output == nil || resolved.output.UtxoId == common.INVALID_ID {
			continue
		}
		for _, asset := range instance.RunesIndexer.GetUtxoAssets(resolved.output.UtxoId) {
			id, err := runestone.RuneIdFromString(asset.RuneId)
			if err != nil || id == nil {
				return nil, false
			}
			balances[*id] = balances[*id].Add(asset.Balance)
		}
	}

	artifact, err := (&runestone.Runestone{}).DecipherFromTx(tx)
	if err == runestone.ErrNoOpReturn {
		artifact = nil
	} else if artifact == nil && err != nil {
		return nil, false
	}

	if artifact != nil && artifact.Cenotaph != nil {
		return occupied, true
	}

	var stone *runestone.Runestone
	if artifact != nil {
		stone = artifact.Runestone
		if stone == nil {
			return nil, false
		}
		if stone.Mint != nil || stone.Etching != nil {
			return nil, false
		}
	}

	if len(balances) == 0 {
		return occupied, true
	}

	if stone != nil {
		for _, edict := range stone.Edicts {
			if edict.ID.Block == 0 && edict.ID.Tx == 0 {
				continue
			}
			balance := balances[edict.ID]
			if balance.IsZero() {
				continue
			}
			if edict.Output > uint32(len(tx.TxOut)) {
				return nil, false
			}
			if edict.Output == uint32(len(tx.TxOut)) {
				destinations := mempoolSpendableOutputIndexes(tx)
				if len(destinations) == 0 {
					balances[edict.ID] = uint128.Zero
					continue
				}
				if edict.Amount.IsZero() {
					share := balance.Div64(uint64(len(destinations)))
					remainder := balance.Mod64(uint64(len(destinations)))
					for pos, output := range destinations {
						if !share.IsZero() || uint64(pos) < remainder {
							occupied[output] = true
						}
					}
					balances[edict.ID] = uint128.Zero
					continue
				}

				for _, output := range destinations {
					if balance.IsZero() {
						break
					}
					take := edict.Amount
					if balance.Cmp(take) < 0 {
						take = balance
					}
					if !take.IsZero() {
						occupied[output] = true
						balance = balance.Sub(take)
					}
				}
				balances[edict.ID] = balance
				continue
			}

			take := edict.Amount
			if take.IsZero() || balance.Cmp(take) < 0 {
				take = balance
			}
			if !take.IsZero() {
				output := int(edict.Output)
				if !mempoolOutputUnspendable(tx.TxOut[output]) {
					occupied[output] = true
				}
				balance = balance.Sub(take)
				balances[edict.ID] = balance
			}
		}
	}

	defaultOutput := -1
	if stone != nil && stone.Pointer != nil {
		if int(*stone.Pointer) >= len(tx.TxOut) {
			return nil, false
		}
		defaultOutput = int(*stone.Pointer)
	} else {
		for i, output := range tx.TxOut {
			if !mempoolOutputUnspendable(output) {
				defaultOutput = i
				break
			}
		}
	}
	if defaultOutput >= 0 && !mempoolOutputUnspendable(tx.TxOut[defaultOutput]) {
		for _, balance := range balances {
			if !balance.IsZero() {
				occupied[defaultOutput] = true
				break
			}
		}
	}
	return occupied, true
}

func mempoolSpendableOutputIndexes(tx *wire.MsgTx) []int {
	result := make([]int, 0, len(tx.TxOut))
	for i, output := range tx.TxOut {
		if !mempoolOutputUnspendable(output) {
			result = append(result, i)
		}
	}
	return result
}

func mempoolOutputUnspendable(output *wire.TxOut) bool {
	if output == nil || len(output.PkScript) == 0 {
		return false
	}
	script := output.PkScript
	return script[0] == txscript.OP_RETURN || (len(script) >= 2 && script[0] == txscript.OP_FALSE && script[1] == txscript.OP_RETURN)
}

type mempoolAtomBalance struct {
	atomicalID string
	amount     int64
	inputIndex int
}

type mempoolAtomAssignment struct {
	output int
	amount int64
}

func allocateMempoolAtom(tx *wire.MsgTx, inputs []*mempoolResolvedInput) ([]bool, bool) {
	occupied := make([]bool, len(tx.TxOut))
	spent := make([]mempoolAtomBalance, 0)
	for _, resolved := range inputs {
		if resolved == nil || !resolved.confirmed || resolved.output == nil || resolved.output.UtxoId == common.INVALID_ID {
			continue
		}
		for _, balance := range instance.atomIndexer.GetUtxoBalances(resolved.output.UtxoId) {
			if balance == nil || balance.Amount <= 0 {
				continue
			}
			spent = append(spent, mempoolAtomBalance{
				atomicalID: balance.AtomicalId,
				amount:     balance.Amount,
				inputIndex: resolved.index,
			})
		}
	}

	height := instance.GetSyncHeight() + 1
	densityHeight := atomidx.AtomicalsActivationDensityMainnet
	dmintHeight := atomidx.AtomicalsActivationDmintMainnet
	coloringHeight := atomidx.AtomicalsActivationColoringMainnet
	if instance.GetChainParam().Net != wire.MainNet {
		densityHeight = atomidx.AtomicalsActivationTestnet4
		dmintHeight = atomidx.AtomicalsActivationTestnet4
		coloringHeight = atomidx.AtomicalsActivationTestnet4
	}
	atomTx := mempoolCommonTransaction(tx, inputs)
	op := atomidx.ParseOperation(atomTx, height >= densityHeight)

	if op != nil && len(tx.TxOut) > 0 && (op.Op == atomidx.OpDirectFT || op.Op == atomidx.OpMintDFT) {
		occupied[0] = true
	}
	if len(spent) == 0 {
		return occupied, true
	}

	grouped := make(map[string]int64)
	fromInput := make(map[string]map[int]bool)
	for _, item := range spent {
		grouped[item.atomicalID] += item.amount
		if fromInput[item.atomicalID] == nil {
			fromInput[item.atomicalID] = make(map[int]bool)
		}
		fromInput[item.atomicalID][item.inputIndex] = true
	}

	ids := make([]string, 0, len(grouped))
	if height >= dmintHeight {
		seen := make(map[string]bool)
		for _, item := range spent {
			if !seen[item.atomicalID] {
				seen[item.atomicalID] = true
				ids = append(ids, item.atomicalID)
			}
		}
	} else {
		for id := range grouped {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return mempoolCompareAtomicalIDs(ids[i], ids[j]) < 0 })
	}

	customActivated := height >= coloringHeight
	assignmentsByID := make(map[string][]mempoolAtomAssignment, len(ids))
	startOutput := 0
	clean := true
	isSplit := op != nil && op.Op == atomidx.OpSplit
	isCustom := op != nil && op.Op == atomidx.OpCustomColor && customActivated

	for _, id := range ids {
		amount := grouped[id]
		assignments, ok := mempoolAtomAssignRegular(tx, startOutput, amount, customActivated)
		if isSplit && fromInput[id][op.InputIndex] {
			assignments = mempoolAtomAssignSplit(tx, id, amount, op, customActivated)
			ok = len(assignments) > 0 || amount == 0
		}
		if isCustom {
			assignments = mempoolAtomAssignCustom(tx, id, amount, op)
			if len(assignments) == 0 {
				assignments, ok = mempoolAtomAssignRegular(tx, startOutput, amount, customActivated)
			}
		}
		if !ok && (!customActivated || len(assignments) == 0) && !isSplit && !isCustom {
			clean = false
			break
		}
		assignmentsByID[id] = assignments
		if len(assignments) > 0 {
			startOutput = assignments[len(assignments)-1].output + 1
		}
	}

	if !clean {
		assignmentsByID = make(map[string][]mempoolAtomAssignment, len(ids))
		for _, id := range ids {
			assignments, _ := mempoolAtomAssignRegular(tx, 0, grouped[id], customActivated)
			assignmentsByID[id] = assignments
		}
	}

	for _, assignments := range assignmentsByID {
		for _, assignment := range assignments {
			if assignment.amount <= 0 || assignment.output < 0 || assignment.output >= len(tx.TxOut) {
				continue
			}
			if mempoolOutputUnspendable(tx.TxOut[assignment.output]) {
				continue
			}
			occupied[assignment.output] = true
		}
	}
	return occupied, true
}

func mempoolCommonTransaction(tx *wire.MsgTx, inputs []*mempoolResolvedInput) *common.Transaction {
	result := &common.Transaction{TxId: tx.TxID()}
	result.Inputs = make([]*common.TxInput, 0, len(inputs))
	for _, resolved := range inputs {
		if resolved == nil || resolved.output == nil || resolved.index < 0 || resolved.index >= len(tx.TxIn) {
			continue
		}
		compiled := common.NewCompilingOutput(resolved.output)
		result.Inputs = append(result.Inputs, &common.TxInput{
			TxOutputV2: *compiled,
			Witness:    tx.TxIn[resolved.index].Witness,
			TxInIndex:  resolved.index,
		})
	}
	result.Outputs = make([]*common.TxOutputV2, len(tx.TxOut))
	for i, txOut := range tx.TxOut {
		output := common.NewTxOutputV2(txOut.Value)
		output.OutValue.PkScript = txOut.PkScript
		output.OutPointStr = fmt.Sprintf("%s:%d", tx.TxID(), i)
		output.TxOutIndex = i
		result.Outputs[i] = output
	}
	return result
}

func mempoolAtomAssignRegular(tx *wire.MsgTx, start int, amount int64, customActivated bool) ([]mempoolAtomAssignment, bool) {
	result := make([]mempoolAtomAssignment, 0)
	remaining := amount
	for i := start; i < len(tx.TxOut) && remaining > 0; i++ {
		output := tx.TxOut[i]
		if mempoolOutputUnspendable(output) {
			continue
		}
		value := output.Value
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
		result = append(result, mempoolAtomAssignment{output: i, amount: assign})
		remaining -= assign
		if !customActivated && assign < value {
			return nil, false
		}
	}
	return result, remaining == 0
}

func mempoolAtomAssignSplit(tx *wire.MsgTx, atomicalID string, amount int64, op *atomidx.Operation, customActivated bool) []mempoolAtomAssignment {
	skip, _ := mempoolAtomIntArg(op.Payload.Args, atomicalID)
	result := make([]mempoolAtomAssignment, 0)
	remaining := amount
	var skipped int64
	for i, output := range tx.TxOut {
		if skip > 0 && skipped < skip {
			skipped += output.Value
			continue
		}
		value := output.Value
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
		result = append(result, mempoolAtomAssignment{output: i, amount: assign})
		remaining -= assign
		if remaining == 0 {
			break
		}
	}
	return result
}

func mempoolAtomAssignCustom(tx *wire.MsgTx, atomicalID string, amount int64, op *atomidx.Operation) []mempoolAtomAssignment {
	raw, ok := op.Payload.Args[atomicalID]
	if !ok {
		return nil
	}
	outMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	result := make([]mempoolAtomAssignment, 0)
	remaining := amount
	for i := range tx.TxOut {
		value, ok := mempoolAtomIntArg(outMap, strconv.Itoa(i))
		if !ok || value <= 0 || remaining <= 0 {
			continue
		}
		if value > tx.TxOut[i].Value {
			value = tx.TxOut[i].Value
		}
		if value > remaining {
			value = remaining
		}
		result = append(result, mempoolAtomAssignment{output: i, amount: value})
		remaining -= value
	}
	return result
}

func mempoolAtomIntArg(args map[string]any, name string) (int64, bool) {
	v, ok := args[name]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int64:
		return t, true
	case uint64:
		if t > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func mempoolCompareAtomicalIDs(a, b string) int {
	aKey, aOK := mempoolAtomicalIDSortKey(a)
	bKey, bOK := mempoolAtomicalIDSortKey(b)
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

func mempoolAtomicalIDSortKey(id string) ([]byte, bool) {
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

func markMempoolInscriptionOutputs(tx *wire.MsgTx, inputs []*mempoolResolvedInput, occupied []bool) {
	if len(tx.TxOut) == 0 || len(inputs) == 0 {
		return
	}
	height := instance.GetSyncHeight() + 1
	inputBase := make(map[int]int64, len(inputs))
	var totalInput int64
	for _, input := range inputs {
		if input == nil || input.output == nil {
			continue
		}
		inputBase[input.index] = totalInput
		totalInput += input.output.Value()
	}
	var totalOutput int64
	for _, output := range tx.TxOut {
		totalOutput += output.Value
	}

	for i, txIn := range tx.TxIn {
		inscriptions := ord0_14_1.GetInscriptionsInTxInput(txIn.Witness, height, i)
		for _, inscription := range inscriptions {
			start := inputBase[i]
			if inscription.Inscription.Pointer != nil {
				start = int64(common.GetSatPointer(inscription.Inscription.Pointer))
				if start >= totalOutput && start > 0 {
					start = 0
				}
			}
			mempoolMarkSatRange(tx, start, start+1, occupied)

			ordxInfo, isOrdx := ord.IsOrdXProtocol(inscription)
			if !isOrdx {
				continue
			}
			basic := common.GetBasicContent(ordxInfo)
			if basic == nil || basic.Op != "mint" {
				continue
			}
			mint := common.ParseMintContent(ordxInfo)
			if mint == nil {
				continue
			}
			ticker := instance.GetTicker(mint.Ticker)
			if ticker == nil || ticker.N <= 0 {
				continue
			}
			amt := ticker.Limit
			if mint.Amt != "" {
				parsed, err := strconv.ParseInt(mint.Amt, 10, 64)
				if err != nil || parsed <= 0 || parsed > ticker.Limit {
					continue
				}
				amt = parsed
			}
			if amt <= 0 || amt%int64(ticker.N) != 0 {
				continue
			}
			sats := amt / int64(ticker.N)
			inputIndex, localOffset, ok := mempoolInputAtGlobalOffset(inputs, start)
			if !ok || inputIndex < 0 || inputIndex >= len(inputs) || inputs[inputIndex].output == nil {
				continue
			}
			if localOffset+sats > inputs[inputIndex].output.Value() {
				continue
			}
			mempoolMarkSatRange(tx, start, start+sats, occupied)
		}
	}
}

func mempoolInputAtGlobalOffset(inputs []*mempoolResolvedInput, offset int64) (int, int64, bool) {
	var base int64
	for pos, input := range inputs {
		if input == nil || input.output == nil {
			continue
		}
		value := input.output.Value()
		if offset >= base && offset < base+value {
			return pos, offset - base, true
		}
		base += value
	}
	return -1, 0, false
}

func mempoolMarkSatRange(tx *wire.MsgTx, start, end int64, occupied []bool) {
	if end <= start || start < 0 {
		return
	}
	var base int64
	for i, output := range tx.TxOut {
		next := base + output.Value
		if output.Value > 0 && start < next && end > base {
			occupied[i] = true
		}
		base = next
		if base >= end {
			break
		}
	}
}
