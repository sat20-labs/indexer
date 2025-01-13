package runes

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/OLProtocol/go-bitcoind"
	"github.com/btcsuite/btcd/txscript"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/store"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
	"lukechampine.com/uint128"
)

func (s *Indexer) UpdateDB() {
	if s.wb == nil {
		return
	}

	store.SetCacheLogs(s.cacheLogs)
	store.FlushToDB()
	s.Status.Height = s.height
	s.Status.FlushToDB()

	setCount := 0
	delCount := 0
	for v := range s.cacheLogs.IterBuffered() {
		if v.Val.Type == store.DEL {
			delCount++
		} else if v.Val.Type == store.PUT {
			setCount++
		}
	}

	s.wb = nil
	s.isUpdateing = false
	store.SetCacheLogs(nil)
	s.cacheLogs = nil
	common.Log.Infof("RuneIndexer.UpdateDB-> db commit success, height:%d, set db count:%d, db del count:%d", s.Status.Height, setCount, delCount)
}

func (s *Indexer) UpdateTransfer(block *common.Block) {
	if !s.isUpdateing {
		if block.Height > 0 {
			if s.Status.Height < uint64(block.Height-1) {
				common.Log.Panicf("RuneIndexer.UpdateTransfer-> err: status.Height(%d) < block.Height-1(%d), missing intermediate blocks", s.Status.Height, block.Height-1)
			} else if s.Status.Height >= uint64(block.Height) {
				common.Log.Infof("RuneIndexer.UpdateTransfer-> cointinue next block, because status.Height(%d) > block.Height(%d)", s.Status.Height, block.Height)
				return
			}
		} else {
			if s.Status.Height > uint64(block.Height) {
				common.Log.Infof("RuneIndexer.UpdateTransfer-> cointinue next block, because status.Height(%d) > block.Height(%d)", s.Status.Height, block.Height)
				return
			}
		}
		s.isUpdateing = true
	}

	if s.wb == nil {
		s.wb = s.db.NewWriteBatch()
		store.SetWriteBatch(s.wb)
		cacheLogs := cmap.New[*store.CacheLog]()
		s.cacheLogs = &cacheLogs
		store.SetCacheLogs(s.cacheLogs)
	}

	s.height = uint64(block.Height)
	s.burnedMap = make(runestone.RuneIdLotMap)
	minimum := runestone.MinimumAtHeight(s.chaincfgParam.Net, uint64(block.Height))
	s.minimumRune = &minimum
	s.blockTime = uint64(block.Timestamp.Unix())
	common.Log.Tracef("RuneIndexer.UpdateTransfer->prepare block height:%d, minimumRune:%s(%s)",
		block.Height, s.minimumRune.String(), s.minimumRune.Value.String())
	startTime := time.Now()
	for txIndex, transaction := range block.Transactions {
		isParseOk, _ := s.index_runes(uint32(txIndex), transaction)
		if isParseOk {
			common.Log.Tracef("RuneIndexer.UpdateTransfer-> height:%d, txIndex:%d, txid:%s",
				block.Height, txIndex, transaction.Txid)
		}
	}
	sinceTime := time.Since(startTime)
	txCount := len(block.Transactions)
	common.Log.Infof("RuneIndexer.UpdateTransfer-> handle block succ, height:%d, tx count:%d, block took time:%v, tx took avg time:%v",
		block.Height, txCount, sinceTime, sinceTime/time.Duration(txCount))
	s.update()
}

func (s *Indexer) index_runes(tx_index uint32, tx *common.Transaction) (isParseOk bool, err error) {
	debug := 0
	if tx.Txid == "30c2c5a7cabd2f819e6969504b96988e1ffeb785b5c2b471d0f1b5d9b7c1ec51" {
		debug++
	}
	var artifact *runestone.Artifact
	artifact, err = parseArtifact(tx)
	if err != nil {
		if err != runestone.ErrNoOpReturn {
			common.Log.Tracef("RuneIndexer.index_runes-> parseArtifact(%s) err:%s", tx.Txid, err.Error())
		}
	} else {
		common.Log.Tracef("RuneIndexer.index_runes-> parseArtifact(%s) ok, tx_index:%d, artifact:%+v", tx.Txid, tx_index, artifact)
	}

	unallocated, runeIdOutPointAddressIds := s.unallocated(tx)

	type RuneIdLotMapVec map[uint32]runestone.RuneIdLotMap
	allocated := make(RuneIdLotMapVec, len(tx.Outputs))
	for outputIndex := range tx.Outputs {
		allocated[uint32(outputIndex)] = make(runestone.RuneIdLotMap)
	}

	var bornedRuneEntry *runestone.RuneEntry
	var mintAmount *runestone.Lot
	var outIndex *uint32
	var mintRuneId *runestone.RuneId

	if artifact != nil {
		isParseOk = true
		mintRuneId = artifact.Mint()
		if mintRuneId != nil {
			var err error
			mintAmount, err = s.mint(mintRuneId)
			if err == nil && mintAmount != nil {
				unallocated.GetOrDefault(mintRuneId).AddAssign(mintAmount)
				mintRuneEntry := s.idToEntryTbl.Get(mintRuneId)
				if mintRuneEntry == nil {
					common.Log.Panicf("RuneIndexer.index_runes-> mintRuneEntry is nil")
				}
			}
		}

		etchedId, etchedRune := s.etched(tx_index, tx, artifact)
		if artifact.Runestone != nil {
			if etchedId != nil {
				premine := &uint128.Uint128{}
				if artifact.Runestone.Etching.Premine != nil {
					premine = artifact.Runestone.Etching.Premine
				}
				premineAmount := runestone.NewLot(premine)
				unallocated.GetOrDefault(etchedId).AddAssign(premineAmount)
			}

			zeroId := runestone.RuneId{Block: uint64(0), Tx: uint32(0)}
			for _, edict := range artifact.Runestone.Edicts {
				amount := runestone.NewLot(&edict.Amount)
				// edicts with output values greater than the number of outputs
				// should never be produced by the edict parser
				output := edict.Output
				if output >= uint32(len(tx.Outputs)) {
					common.Log.Panicf("RuneIndexer.index_runes-> output is greater than transaction output count")
				}

				var id *runestone.RuneId
				if edict.ID.Cmp(zeroId) == 0 {
					if etchedId != nil {
						id = etchedId
					} else {
						continue
					}
				} else {
					id = &edict.ID
				}
				balance := unallocated.Get(id)
				if balance == nil {
					continue
				}

				// transfers
				if edict.ID.Cmp(zeroId) == 0 {
					if etchedRune == nil {
						common.Log.Panicf("RuneIndexer.index_runes-> etched rune not found")
					}
				} else {
					runeEntry := s.idToEntryTbl.Get(id)
					if runeEntry == nil {
						common.Log.Panicf("RuneIndexer.index_runes-> rune entry not found")
					}
				}

				allocate := func(balance *runestone.Lot, amount *runestone.Lot, output uint32) {
					if amount.Value.Cmp(uint128.Zero) > 0 {
						balance.SubAssign(*amount)
						allocated[output].GetOrDefault(id).AddAssign(amount)
					}
				}

				if output == uint32(len(tx.Outputs)) {
					// find non-OP_RETURN outputs
					var destinations []uint32
					for outputIndex, output := range tx.Outputs {
						if output.Address.PkScript[0] != txscript.OP_RETURN {
							destinations = append(destinations, uint32(outputIndex))
						}
					}
					if len(destinations) > 0 {
						if amount.Value.Cmp(uint128.Zero) == 0 {
							destinationsLen := uint128.From64(uint64(len(destinations)))
							value := balance.Div(&destinationsLen)
							remainder := balance.Rem(&destinationsLen).Value.Big().Uint64()
							for index, output := range destinations {
								if index < int(remainder) {
									one := uint128.From64(1)
									value = value.AddUint128(&one)
								}
								allocate(balance, &value, output)
							}
						} else {
							for _, output := range destinations {
								value := runestone.NewLot(&amount.Value)
								allocate(balance, value, output)
							}
						}
					}
				} else {
					// Get the allocatable amount
					var value *runestone.Lot
					if amount.Value.Cmp(uint128.Zero) == 0 {
						value = runestone.NewLot(&balance.Value)
					} else {
						if balance.Cmp(&amount.Value) < 0 {
							value = runestone.NewLot(&balance.Value)
						} else {
							value = runestone.NewLot(&amount.Value)
						}
					}
					allocate(balance, value, output)
				}
			}
		}

		if etchedRune != nil {
			s.runeToIdTbl.Insert(etchedRune, etchedId)
			bornedRuneEntry = s.create_rune_entry(tx, artifact, etchedId, etchedRune)
			s.idToEntryTbl.Insert(etchedId, bornedRuneEntry)
		}
	}

	burned := make(runestone.RuneIdLotMap)

	if artifact != nil && artifact.Cenotaph != nil {
		for id, v := range unallocated {
			burned.GetOrDefault(&id).AddAssign(v)
		}
	} else {
		var pointer *uint32
		if artifact != nil && artifact.Runestone != nil {
			pointer = artifact.Runestone.Pointer
		}
		// assign all un-allocated runes to the default output, or the first non
		// OP_RETURN output if there is no default
		find := false

		if pointer == nil {
			for index, v := range tx.Outputs {
				if v.Address.PkScript[0] != txscript.OP_RETURN {
					u32Index := uint32(index)
					outIndex = &u32Index
					find = true
					break
				}
			}
		} else if (*pointer) < uint32(len(allocated)) {
			outIndex = pointer
			find = true
		} else if (*pointer) >= uint32(len(allocated)) {
			common.Log.Panicf("RuneIndexer.index_runes-> pointer out of range")
		}
		if find {
			for id, balance := range unallocated {
				if balance.Value.Cmp(uint128.Zero) > 0 {
					allocated[*outIndex].GetOrDefault(&id).AddAssign(balance)
				}
			}
		} else {
			for id, balance := range unallocated {
				if balance.Value.Cmp(uint128.Zero) > 0 {
					burned.GetOrDefault(&id).AddAssign(balance)
				}
			}
		}
	}

	type RuneIdOutpointAddressToBalance struct {
		RuneId    *runestone.RuneId
		OutPoint  *runestone.OutPoint
		AddressId uint64
		Address   runestone.Address
		Balance   runestone.Lot
		OutIndex  uint32
	}
	type RuneBalanceArray []*RuneIdOutpointAddressToBalance
	runeBalanceArray := make(RuneBalanceArray, 0)

	type RuneIdToAddressRuneIdToMintHistoryMap map[runestone.RuneId]runestone.AddressRuneIdToMintHistory
	runeIdToAddressRuneIdToMintHistoryMap := make(RuneIdToAddressRuneIdToMintHistoryMap)

	// update outpoint balances
	for vout, balances := range allocated {
		if len(balances) == 0 {
			continue
		}
		for _, balance := range balances {
			if balance.Value.Cmp(uint128.Zero) == 0 {
				debug++
			}
		}
		// increment burned balances
		if tx.Outputs[vout].Address.PkScript[0] == txscript.OP_RETURN {
			for id, balance := range balances {
				burned.GetOrDefault(&id).AddAssign(balance)
			}
			continue
		}
		// Sort balanceArray by id so tests can assert balanceArray in a fixed order
		outpoint := &runestone.OutPoint{Txid: tx.Txid, Vout: vout, UtxoId: common.GetUtxoId(tx.Outputs[vout])}
		address, err := parseTxVoutScriptAddress(tx, int(vout), *s.chaincfgParam)
		if err != nil {
			common.Log.Panicf("RuneIndexer.index_runes-> parseTxVoutScriptAddress(%v,%v,%v) err:%v",
				tx.Txid, vout, s.chaincfgParam.Net, err)
		}
		addressId := s.BaseIndexer.GetAddressId(string(address))
		outpointToBalancesValue := &runestone.OutpointToBalancesValue{
			Utxo:       fmt.Sprintf("%s:%d", tx.Txid, vout),
			Address:    string(address),
			AddressId:  addressId,
			RuneIdLots: balances.GetSortArray(),
		}
		s.outpointToBalancesTbl.Insert(outpoint, outpointToBalancesValue)

		// update runeIdToOutputMap and runeIdToAddressMap
		for runeId, balance := range balances {
			if balance.Value.Cmp(uint128.Zero) > 0 {
				runeIdToAddressRuneIdToMintHistoryMap[runeId] = runestone.AddressRuneIdToMintHistory{
					Address: address, RuneId: &runeId, OutPoint: outpoint,
					AddressId: addressId,
				}

				runeBalanceArray = append(runeBalanceArray, &RuneIdOutpointAddressToBalance{
					RuneId:    &runeId,
					OutPoint:  outpoint,
					Balance:   *balance,
					Address:   address,
					AddressId: addressId,
					OutIndex:  vout,
				})
			}
		}
	}

	// increment entries with burned runes
	for id, amount := range burned {
		s.burnedMap.GetOrDefault(&id).AddAssign(amount)
	}

	if artifact != nil && artifact.Runestone == nil {
		return
	}

	// add for balances and holder count
	for _, runeBalance := range runeBalanceArray {
		// update runeIdToOutpointToBalance
		runeIdToOutpointToBalance := &runestone.RuneIdOutpointToBalance{
			RuneId:   runeBalance.RuneId,
			OutPoint: runeBalance.OutPoint,
			Balance:  runeBalance.Balance,
		}
		s.runeIdOutpointToBalanceTbl.Insert(runeIdToOutpointToBalance)

		// update addressOutpointToBalance
		addressOutpointToBalance := &runestone.AddressOutpointToBalance{
			AddressId: runeBalance.AddressId,
			OutPoint:  runeBalance.OutPoint,
			Address:   runeBalance.Address,
			RuneId:    runeBalance.RuneId,
			Balance:   runeBalance.Balance,
		}
		oldAddressOutpointToBalance := s.addressOutpointToBalancesTbl.Get(addressOutpointToBalance)
		if oldAddressOutpointToBalance != nil {
			addressOutpointToBalance.Balance.AddAssign(&oldAddressOutpointToBalance.Balance)
		}
		s.addressOutpointToBalancesTbl.Insert(addressOutpointToBalance)

		r := s.idToEntryTbl.Remove(runeBalance.RuneId)
		if r != nil {
			r.HolderCount++
			s.idToEntryTbl.Insert(runeBalance.RuneId, r)
		}
	}

	// clean and sub for balances
	for _, runeBalance := range runeBalanceArray {
		key := &runestone.RuneIdAddressToBalance{
			RuneId:    runeBalance.RuneId,
			AddressId: runeBalance.AddressId,
		}
		value := s.runeIdAddressToBalanceTbl.Get(key)
		if value != nil {
			value.Balance.AddAssign(&key.Balance)
		} else {
			value = &runestone.RuneIdAddressToBalance{
				RuneId:    runeBalance.RuneId,
				AddressId: runeBalance.AddressId,
				Address:   runeBalance.Address,
				Balance:   runeBalance.Balance,
			}
		}
		s.runeIdAddressToBalanceTbl.Insert(value)

		for _, v := range runeIdOutPointAddressIds {
			if v.RuneId.Cmp(*runeBalance.RuneId) != 0 {
				continue
			}
			key := &runestone.RuneIdAddressToBalance{RuneId: v.RuneId, AddressId: v.AddressId}
			value := s.runeIdAddressToBalanceTbl.Get(key)
			if value != nil {
				var amount uint128.Uint128 = uint128.Uint128{Lo: 0, Hi: 0}
				if value.Balance.Value.Cmp(runeBalance.Balance.Value) < 0 {
					debug++
					amount = uint128.Zero
				} else {
					amount = value.Balance.Value.Sub(runeBalance.Balance.Value)
				}

				if amount.Cmp(uint128.Zero) != 0 {
					value.Balance.Value = amount
					s.runeIdAddressToBalanceTbl.Insert(value)
				} else {
					s.runeIdAddressToBalanceTbl.Remove(value)
				}
			} else {
				debug++
			}
		}
	}

	// update runeIdToMintHistory
	if mintAmount != nil {
		if outIndex == nil {
			common.Log.Panicf("RuneIndexer.index_runes-> mintOutIndex is nil")
		}
		utxo := fmt.Sprintf("%s:%d", tx.Txid, *outIndex)
		output := tx.Outputs[*outIndex]
		utxoId := common.GetUtxoId(output)
		address, err := parseTxVoutScriptAddress(tx, int(*outIndex), *s.chaincfgParam)
		if err != nil {
			common.Log.Debugf("RuneIndexer.index_runes-> parseTxVoutScriptAddress(%v,%v,%v) err:%v",
				tx.Txid, outIndex, s.chaincfgParam.Net, err)
		} else {
			addressId := s.BaseIndexer.GetAddressId(string(address))
			v := &runestone.RuneIdToMintHistory{
				RuneId:    mintRuneId,
				Utxo:      runestone.Utxo(utxo),
				UtxoId:    utxoId,
				Address:   string(address),
				AddressId: addressId,
			}
			s.runeIdToMintHistoryTbl.Insert(v)
		}
	}

	// update addressRuneIdToMintHistory
	for r, h := range runeIdToAddressRuneIdToMintHistoryMap {
		v := &runestone.AddressRuneIdToMintHistory{RuneId: &r, Address: h.Address, OutPoint: h.OutPoint, AddressId: h.AddressId}
		s.addressRuneIdToMintHistoryTbl.Insert(v)
	}

	return
}

func (s *Indexer) update() {
	for id, burned := range s.burnedMap {
		entry := s.idToEntryTbl.Get(&id)
		entry.Burned = entry.Burned.Add(burned.Value)
		s.idToEntryTbl.Insert(&id, entry)
	}
}

func (s *Indexer) create_rune_entry(tx *common.Transaction, artifact *runestone.Artifact, id *runestone.RuneId, r *runestone.Rune) (entry *runestone.RuneEntry) {
	number := s.Status.Number
	s.Status.Number++
	parent := tryGetFirstInscriptionId(tx)
	if artifact.Cenotaph != nil {
		entry = &runestone.RuneEntry{
			RuneId:       *id,
			Burned:       uint128.Uint128{},
			Divisibility: 0,
			Etching:      tx.Txid,
			Parent:       nil,
			Terms:        nil,
			Mints:        uint128.Uint128{},
			Number:       number,
			Premine:      uint128.Uint128{},
			SpacedRune:   *runestone.NewSpacedRune(*r, 0),
			Symbol:       nil,
			Timestamp:    s.blockTime,
			Turbo:        false,
		}
	} else if artifact.Runestone != nil {
		entry = &runestone.RuneEntry{
			RuneId:     *id,
			Burned:     uint128.Uint128{},
			Etching:    tx.Txid,
			Parent:     parent,
			Terms:      artifact.Runestone.Etching.Terms,
			Mints:      uint128.Uint128{},
			Number:     number,
			SpacedRune: *runestone.NewSpacedRune(*r, 0),
			Symbol:     artifact.Runestone.Etching.Symbol,
			Timestamp:  s.blockTime,
			Turbo:      artifact.Runestone.Etching.Turbo,
		}

		if artifact.Runestone.Etching.Divisibility != nil {
			entry.Divisibility = *artifact.Runestone.Etching.Divisibility
		}
		if artifact.Runestone.Etching.Premine != nil {
			entry.Premine = *artifact.Runestone.Etching.Premine
		}
		if artifact.Runestone.Etching.Spacers != nil {
			entry.SpacedRune = *runestone.NewSpacedRune(*r, *artifact.Runestone.Etching.Spacers)
		}
	}
	return
}

type RuneIdOutPointAddressId struct {
	RuneId    *runestone.RuneId
	OutPoint  *runestone.OutPoint
	AddressId uint64
}

func (s *Indexer) unallocated(tx *common.Transaction) (ret1 runestone.RuneIdLotMap, ret2 []*RuneIdOutPointAddressId) {
	ret1 = make(runestone.RuneIdLotMap)
	for j, input := range tx.Inputs {
		outpoint := &runestone.OutPoint{
			Txid:   input.Txid,
			Vout:   uint32(input.Vout),
			UtxoId: input.UtxoId,
		}
		oldValue := s.outpointToBalancesTbl.Remove(outpoint)
		if oldValue != nil {
			for _, val := range oldValue.RuneIdLots {
				if ret1[val.RuneId] == nil {
					ret1[val.RuneId] = runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
				}
				ret1[val.RuneId].AddAssign(&val.Lot)

				runeIdOutpointToBalance := &runestone.RuneIdOutpointToBalance{
					RuneId:   &val.RuneId,
					OutPoint: outpoint,
				}
				s.runeIdOutpointToBalanceTbl.Remove(runeIdOutpointToBalance)

				addressOutpointToBalance := &runestone.AddressOutpointToBalance{
					AddressId: oldValue.AddressId,
					OutPoint:  outpoint,
				}
				s.addressOutpointToBalancesTbl.Remove(addressOutpointToBalance)
				if ret2 == nil {
					ret2 = make([]*RuneIdOutPointAddressId, 0)
				}

				addressRuneIdToMintHistory := &runestone.AddressRuneIdToMintHistory{
					AddressId: oldValue.AddressId,
					Address:   runestone.Address(oldValue.Address),
					RuneId:    &val.RuneId,
					OutPoint:  outpoint,
				}
				s.addressRuneIdToMintHistoryTbl.Remove(addressRuneIdToMintHistory)

				oldRuneEntry := s.idToEntryTbl.Remove(&val.RuneId)
				if oldRuneEntry != nil && oldRuneEntry.HolderCount > 0 {
					oldRuneEntry.HolderCount--
					s.idToEntryTbl.Insert(&val.RuneId, oldRuneEntry)
				}

				ret2 = append(ret2, &RuneIdOutPointAddressId{
					RuneId:    &val.RuneId,
					OutPoint:  outpoint,
					AddressId: oldValue.AddressId,
				})
			}
		}
		j++
	}
	return
}

func (s *Indexer) mint(runeId *runestone.RuneId) (lot *runestone.Lot, err error) {
	runeEntry := s.idToEntryTbl.Get(runeId)
	if runeEntry == nil {
		return
	}
	var amount *uint128.Uint128
	amount, err = runeEntry.Mintable(s.height)
	if err != nil {
		return
	}
	runeEntry.Mints = runeEntry.Mints.Add64(1)
	s.idToEntryTbl.Insert(runeId, runeEntry)
	lot = &runestone.Lot{
		Value: *amount,
	}
	return
}

func (s *Indexer) etched(txIndex uint32, tx *common.Transaction, artifact *runestone.Artifact) (
	runeId *runestone.RuneId, r *runestone.Rune) {
	if artifact.Runestone != nil {
		if artifact.Runestone.Etching == nil {
			return
		}
		r = artifact.Runestone.Etching.Rune
	} else if artifact.Cenotaph != nil {
		if artifact.Cenotaph.Etching == nil {
			return
		}
		r = artifact.Cenotaph.Etching
	}

	if r == nil {
		reserved_runes := s.Status.ReservedRunes
		s.Status.ReservedRunes = reserved_runes + 1
		r = runestone.Reserved(s.height, txIndex)
	} else {
		if r.Value.Cmp(s.minimumRune.Value) < 0 ||
			r.IsReserved() ||
			s.runeToIdTbl.Get(r) != nil ||
			!s.txCommitsToRune(tx, *r) {
			r = nil
			return
		}
	}
	runeId = &runestone.RuneId{
		Block: s.height,
		Tx:    txIndex,
	}
	return runeId, r
}

func (s *Indexer) txCommitsToRune(transaction *common.Transaction, rune runestone.Rune) bool {
	commitment := rune.Commitment()
	for _, input := range transaction.Inputs {
		// extracting a tapscript does not indicate that the input being spent
		// was actually a taproot output. this is checked below, when we load the
		// output's entry from the database
		tapscript := parseTapscript(input.Witness)
		if tapscript == nil {
			continue
		}

		instructions := parseTapscriptLegacyInstructions(tapscript)
		for _, instruction := range instructions {
			// ignore errors, since the extracted script may not be valid
			if !bytes.Equal(instruction, commitment) {
				continue
			}

			var err error
			var resp any
			for {
				resp, err = bitcoin_rpc.ShareBitconRpc.GetRawTransaction(input.Txid, true)
				if err == nil {
					break
				} else {
					time.Sleep(1 * time.Second)
					common.Log.Infof("RuneIndexer.txCommitsToRune-> bitcoin_rpc.GetRawTransaction failed, try again ...")
					continue
				}
			}

			txInfo, _ := resp.(bitcoind.RawTransaction)
			hexStr := txInfo.Vout[input.Vout].ScriptPubKey.Hex
			// is_p2tr
			taproot := false
			hexBytes, err := hex.DecodeString(hexStr)
			if err != nil {
				common.Log.Panicf("RuneIndexer.txCommitsToRune-> hex.DecodeString(%s) err:%v", hexStr, err)
			}
			if len(hexBytes) == 34 && hexBytes[1] == txscript.OP_DATA_32 {
				verOpcode := int(hexBytes[0])
				if verOpcode == 0 {
					taproot = false
				} else {
					if verOpcode >= txscript.OP_1 && verOpcode <= txscript.OP_16 {
						verOpcode = verOpcode - txscript.OP_1 + 1
					}
					if verOpcode == 1 {
						taproot = true
					}
				}
			}
			if !taproot {
				continue
			}
			blockHeader, err := bitcoin_rpc.ShareBitconRpc.GetBlockheader(txInfo.BlockHash)
			if err != nil {
				return false
			}
			commitTxHeight := uint64(blockHeader.Height)
			confirmations := s.height - commitTxHeight + 1
			if confirmations >= 6 {
				return true
			}
		}
	}
	return false
}
