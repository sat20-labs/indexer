package runes

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/OLProtocol/go-bitcoind"
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/share/base_indexer"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
	"lukechampine.com/uint128"
)

func (s *Indexer) UpdateDB() {
	if s.txn == nil {
		return
	}
	err := s.txn.Commit()
	if err != nil {
		common.Log.Panicf("RuneIndexer.UpdateDB-> txn.Commit err:%s", err.Error())
	}
	s.txn.Discard()
	s.txn = nil
	common.Log.Debugf("RuneIndexer.UpdateDB-> db commit success, height:%d", s.status.Height)
}

func (s *Indexer) setTblTxn() {
	s.status.SetTxn(s.txn)
	s.outpointToRuneBalancesTbl.SetTxn(s.txn)
	s.idToEntryTbl.SetTxn(s.txn)
	s.runeToIdTbl.SetTxn(s.txn)
	s.runeLedgerTbl.SetTxn(s.txn)
	s.runeHolderTbl.SetTxn(s.txn)
	s.runeMintHistorysTbl.SetTxn(s.txn)
}

func (s *Indexer) UpdateTransfer(block *common.Block) {
	if block.Height != 0 {
		if s.status.Height < uint64(block.Height-1) {
			common.Log.Panicf("RuneIndexer.UpdateTransfer-> err: status.Height(%d) < block.Height-1(%d), missing intermediate blocks", s.status.Height, block.Height-1)
		} else if s.status.Height >= uint64(block.Height) {
			common.Log.Infof("RuneIndexer.UpdateTransfer-> cointinue next block, because status.Height(%d) > block.Height(%d)", s.status.Height, block.Height)
			return
		}
	} else {
		if s.status.Height > uint64(block.Height) {
			common.Log.Infof("RuneIndexer.UpdateTransfer-> cointinue next block, because status.Height(%d) > block.Height(%d)", s.status.Height, block.Height)
			return
		}
	}

	s.height = uint64(block.Height)
	if s.txn == nil {
		s.txn = s.db.NewTransaction(true)
		s.setTblTxn()
	}
	s.burnedMap = make(runestone.RuneIdLotMap)
	minimum := runestone.MinimumAtHeight(s.chaincfgParam.Net, uint64(block.Height))
	s.minimumRune = &minimum
	s.blockTime = uint64(block.Timestamp.Unix())
	common.Log.Debugf("RuneIndexer.UpdateTransfer-> height:%d, hash:%s, minimumRune:%s(%v)",
		block.Height, block.Hash, s.minimumRune.String(), s.minimumRune.Value.String())
	for txIndex, transaction := range block.Transactions {
		isParseOk, isSave, _ := s.index_runes(uint32(txIndex), transaction)
		if isParseOk {
			common.Log.Debugf("RuneIndexer.UpdateTransfer-> height:%d, txIndex:%d, txid:%s, isSave:%v",
				block.Height, txIndex, transaction.Txid, isSave)
		}
		s.status.Height = uint64(block.Height)
		s.status.Update()
	}
	s.update()
}

func (s *Indexer) GetOutPoints(address string) (ret []*runestone.OutPoint) {
	utxoid_to_value_map, err := base_indexer.ShareBaseIndexer.GetUTXOsWithAddress(address)
	if err != nil {
		common.Log.Panicf("RuneIndexer.GetOutPoints-> GetUTXOsWithAddress(%s) err:%v", address, err)
	}
	for id := range utxoid_to_value_map {
		utxo, _, err := base_indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(id)
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetOutPoints-> GetOrdinalsWithUtxoId(%d) err:%v", id, err)
		}
		txid, vout, err := common.ParseUtxo(utxo)
		if err != nil {
			common.Log.Panicf("RuneIndexer.GetOutPoints-> ParseUtxo(%s) err:%v", utxo, err)
		}

		outpoint := &runestone.OutPoint{
			Txid: txid,
			Vout: uint32(vout),
		}
		ret = append(ret, outpoint)
	}
	return
}

func (s *Indexer) newRuneAsset() *runestone.RuneAsset {
	return &runestone.RuneAsset{
		Balance:   uint128.Zero,
		IsEtching: false,
		Mints:     make([]*runestone.OutPoint, 0),
		Transfers: make([]*runestone.Edict, 0),
	}
}

func (s *Indexer) initRuneLedger(tx *common.Transaction, voutIndex int) (address *runestone.Address) {
	if s.runeLedger != nil {
		return
	}
	oAddress, err := parseTxVoutScriptAddress(tx, voutIndex, *s.chaincfgParam)
	if err != nil {
		common.Log.Panicf("RuneIndexer.initRuneLedger-> parseTxVoutScriptAddress(%v,%v,%v) err:%v", tx.Txid, voutIndex, s.chaincfgParam.Net, err)
	}
	s.runeLedger = s.runeLedgerTbl.Get(oAddress)
	if s.runeLedger == nil {
		s.runeLedger = &runestone.RuneLedger{Assets: make(map[runestone.Rune]*runestone.RuneAsset)}
	}
	address = &oAddress
	return
}

func (s *Indexer) index_runes(tx_index uint32, tx *common.Transaction) (isParseOk bool, isSave bool, err error) {
	var artifact *runestone.Artifact
	artifact, err = parseArtifact(tx)
	if err != nil {
		if err != runestone.ErrNoOpReturn {
			common.Log.Debugf("RuneIndexer.index_runes-> parseArtifact(%s) err:%s", tx.Txid, err.Error())
		}
	} else {
		common.Log.Debugf("RuneIndexer.index_runes-> parseArtifact(%s) ok, tx_index:%d, artifact:%+v", tx.Txid, tx_index, artifact)
	}

	if artifact != nil {
		s.runeLedger = nil
		isParseOk = true
		unallocated := s.unallocated(tx)
		type RuneIdLogMapVec map[uint32]runestone.RuneIdLotMap
		allocated := make(RuneIdLogMapVec, len(tx.Outputs))
		for outputIndex := range tx.Outputs {
			allocated[uint32(outputIndex)] = make(runestone.RuneIdLotMap)
		}

		// var for runeledger
		var addressVout *uint32
		type TransfersRuneInfo struct {
			Rune   *runestone.Rune
			RuneId *runestone.RuneId
			Amount uint128.Uint128
			Output uint32
		}
		var transfersRuneInfos []TransfersRuneInfo
		var bornedRuneEntry *runestone.RuneEntry

		var pAddress *runestone.Address
		var mintRuneEntry *runestone.RuneEntry
		mintRuneId := artifact.Mint()
		if mintRuneId != nil {
			amount, err := s.mint(mintRuneId)
			if err == nil && amount != nil {
				unallocated.GetOrDefault(mintRuneId).AddAssign(amount)
				mintRuneEntry = s.idToEntryTbl.Get(mintRuneId)
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
							amount := balance.Div(&destinationsLen)
							remainder := balance.Rem(&destinationsLen).Value.Big().Uint64()
							for index, output := range destinations {
								if index < int(remainder) {
									one := uint128.From64(1)
									amount = amount.AddUint128(&one)
								}
								allocate(balance, &amount, output)
							}
						} else {
							for _, output := range destinations {
								allocate(balance, amount, output)
							}
						}
					}
				} else {
					// Get the allocatable amount
					if amount.Value.Cmp(uint128.Zero) == 0 {
						amount = balance
					} else {
						if balance.Cmp(amount.Value) < 0 {
							amount = balance
						}
					}
					allocate(balance, amount, output)
				}

				// ledger
				var transferRune *runestone.Rune
				var transferId *runestone.RuneId
				if edict.ID.Cmp(zeroId) == 0 {
					if etchedRune == nil {
						common.Log.Panicf("RuneIndexer.index_runes-> etched rune not found")
					}
					transferRune = etchedRune
					transferId = etchedId
				} else {
					runeEntry := s.idToEntryTbl.Get(id)
					if runeEntry == nil {
						common.Log.Panicf("RuneIndexer.index_runes-> rune entry not found")
					}
					transferRune = &runeEntry.SpacedRune.Rune
					transferId = id
				}
				if transferRune != nil {
					transfersRuneInfos = append(transfersRuneInfos,
						TransfersRuneInfo{transferRune, transferId, *amount.Value, output})
				}
			}
		}

		if etchedRune != nil {
			s.runeToIdTbl.Insert(etchedRune, etchedId)
			bornedRuneEntry = s.create_rune_entry(tx, artifact, etchedId, etchedRune)
			s.idToEntryTbl.Insert(etchedId, bornedRuneEntry)
		}

		burned := make(runestone.RuneIdLotMap)
		if artifact.Cenotaph != nil {
			for id, v := range unallocated {
				burned.GetOrDefault(&id).AddAssign(v)
			}
		} else if artifact.Runestone != nil {
			pointer := artifact.Runestone.Pointer
			// assign all un-allocated runes to the default output, or the first non
			// OP_RETURN output if there is no default
			find := false
			if pointer == nil {
				for index, v := range tx.Outputs {
					if v.Address.PkScript[0] != txscript.OP_RETURN {
						u32Index := uint32(index)
						addressVout = &u32Index
						find = true
						break
					}
				}
			} else if (*pointer) < uint32(len(allocated)) {
				addressVout = pointer
				find = true
			} else if (*pointer) >= uint32(len(allocated)) {
				common.Log.Panicf("RuneIndexer.index_runes-> pointer out of range")
			}
			if find {
				for id, balance := range unallocated {
					if balance.Value.Cmp(uint128.Zero) > 0 {
						allocated[*addressVout].GetOrDefault(&id).AddAssign(balance)
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

		// update outpoint balances
		for vout, balances := range allocated {
			if len(balances) == 0 {
				continue
			}
			// increment burned balances
			if tx.Outputs[vout].Address.PkScript[0] == txscript.OP_RETURN {
				for id, balance := range balances {
					burned.GetOrDefault(&id).AddAssign(balance)
				}
				continue
			}
			// Sort balances by id so tests can assert balances in a fixed order
			balances := balances.GetSortArray()
			outPoint := runestone.OutPoint{Txid: tx.Txid, Vout: vout}
			s.outpointToRuneBalancesTbl.Insert(&outPoint, balances)
		}

		// increment entries with burned runes
		for id, amount := range burned {
			s.burnedMap.GetOrDefault(&id).AddAssign(amount)
		}

		// ledger begin
		// mint
		if mintRuneEntry != nil {
			pAddress = s.initRuneLedger(tx, int(*addressVout))
			if s.runeLedger.Assets[mintRuneEntry.SpacedRune.Rune] == nil {
				s.runeLedger.Assets[mintRuneEntry.SpacedRune.Rune] = s.newRuneAsset()
			}
			s.runeLedger.Assets[mintRuneEntry.SpacedRune.Rune].Mints =
				append(
					s.runeLedger.Assets[mintRuneEntry.SpacedRune.Rune].Mints,
					&runestone.OutPoint{Txid: tx.Txid, Vout: tx_index},
				)
		}

		// transfer
		if len(transfersRuneInfos) > 0 {
			pAddress = s.initRuneLedger(tx, int(*addressVout))
			for _, transferRuneInfo := range transfersRuneInfos {
				if s.runeLedger.Assets[*transferRuneInfo.Rune] == nil {
					s.runeLedger.Assets[*transferRuneInfo.Rune] = s.newRuneAsset()
				}
				s.runeLedger.Assets[*transferRuneInfo.Rune].Transfers = append(s.runeLedger.Assets[*transferRuneInfo.Rune].Transfers, &runestone.Edict{
					ID:     *transferRuneInfo.RuneId,
					Amount: transferRuneInfo.Amount,
					Output: transferRuneInfo.Output,
				})
			}
		}

		// etch
		if bornedRuneEntry != nil {
			pAddress = s.initRuneLedger(tx, int(*addressVout))
			if s.runeLedger.Assets[bornedRuneEntry.SpacedRune.Rune] != nil {
				common.Log.Panicf("RuneIndexer.index_runes-> rune asset already exists, id: %v", etchedId)
			}
			s.runeLedger.Assets[bornedRuneEntry.SpacedRune.Rune] = s.newRuneAsset()
			s.runeLedger.Assets[bornedRuneEntry.SpacedRune.Rune].IsEtching = true
		}

		// update balance
		for vout, balances := range allocated {
			if len(balances) == 0 {
				continue
			}
			if tx.Outputs[vout].Address.PkScript[0] == txscript.OP_RETURN {
				continue
			}
			for id, balance := range balances {
				queryRuneEntry := s.idToEntryTbl.Get(&id)
				if queryRuneEntry == nil {
					common.Log.Panicf("RuneIndexer.index_runes-> rune entry not found, id: %v", id)
				}
				r := queryRuneEntry.SpacedRune.Rune
				if pAddress == nil {
					pAddress = s.initRuneLedger(tx, int(*addressVout))
				}
				s.runeLedger.Assets[r].Balance = s.runeLedger.Assets[r].Balance.Add(*balance.Value)
			}
		}

		// save rune ledger to db
		if s.runeLedger != nil {
			if len(s.runeLedger.Assets) > 0 {
				s.runeLedgerTbl.Insert(*pAddress, s.runeLedger)
				// update rune holder
				for r, runeAsset := range s.runeLedger.Assets {
					holders := s.runeHolderTbl.Get(&r)
					if holders == nil {
						holders = runestone.RuneHolders{}
						holders = append(holders, &runestone.RuneHolder{
							Address: *pAddress,
							Balance: runeAsset.Balance,
						})
					} else {
						for _, runeHolder := range holders {
							if runeHolder.Address == *pAddress {
								runeHolder.Balance = runeHolder.Balance.Add(runeAsset.Balance)
							}
						}
					}
					s.runeHolderTbl.Insert(&r, holders)
				}

				// update mint history
				for r, runeAsset := range s.runeLedger.Assets {
					mintHistorys := s.runeMintHistorysTbl.Get(&r)
					if mintHistorys == nil {
						mintHistorys = make(runestone.RuneMintHistorys, 0)
					}
					for _, v := range runeAsset.Mints {
						mintHistorys = append(mintHistorys, &runestone.RuneMintHistory{
							Address: *pAddress,
							Rune:    r,
							Utxo:    v.String(),
						})
					}
					if len(mintHistorys) > 0 {
						s.runeMintHistorysTbl.Insert(&r, mintHistorys)
					}
				}
				isSave = true
			}
		}
		// ledger end
	}

	return
}

func (s *Indexer) update() {
	for id, burned := range s.burnedMap {
		entry := s.idToEntryTbl.Get(&id)
		entry.Burned = entry.Burned.Add(*burned.Value)
		s.idToEntryTbl.Insert(&id, entry)
	}
}

func (s *Indexer) create_rune_entry(tx *common.Transaction, artifact *runestone.Artifact, id *runestone.RuneId, r *runestone.Rune) (entry *runestone.RuneEntry) {
	number := s.status.Number
	s.status.Number++
	s.status.Update()
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
			SpacedRune:   runestone.SpacedRune{Rune: *r, Spacers: 0},
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
			SpacedRune: runestone.SpacedRune{Rune: *r, Spacers: 0},
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
			entry.SpacedRune = runestone.SpacedRune{Rune: *r, Spacers: *artifact.Runestone.Etching.Spacers}
		}
	}
	return
}

func (s *Indexer) unallocated(tx *common.Transaction) (ret runestone.RuneIdLotMap) {
	ret = make(runestone.RuneIdLotMap)
	for _, input := range tx.Inputs {
		outpoint := &runestone.OutPoint{
			Txid: input.Txid,
			Vout: uint32(input.Vout),
		}
		oldValue := s.outpointToRuneBalancesTbl.Remove(outpoint)
		if oldValue == nil {
			continue
		}
		for _, val := range *oldValue {
			ret[val.RuneId] = &val.Lot
		}
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
		Value: amount,
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
		reserved_runes := s.status.ReservedRunes
		s.status.ReservedRunes = reserved_runes + 1
		s.status.Update()
		r = runestone.Reserved(s.height, txIndex)
	} else {
		test := r.String()
		common.Log.Debugf("etched rune: %s", test)
		// a := r.Value.Cmp(s.minimumRune.Value) < 0
		// b := r.IsReserved()
		// c := s.runeToIdTbl.Get(r) != nil
		// d := !s.txCommitsToRune(tx, *r)
		// if a || b || c || d {
		// 	r = nil
		// 	return
		// }
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
