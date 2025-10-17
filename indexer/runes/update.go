package runes

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/OLProtocol/go-bitcoind"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/indexer/runes/table"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
	"lukechampine.com/uint128"
)

func (s *Indexer) UpdateDB() {
	if s.height == 0 {
		common.Log.Warnf("RuneIndexer.UpdateDB-> err: height(%d) == 0", s.height)
		return
	}

	if s.Status.Height == s.height {
		common.Log.Warnf("RuneIndexer.UpdateDB-> err: status.Height(%d) == height(%d)", s.Status.Height, s.height)
		return
	} else if s.Status.Height > s.height {
		common.Log.Panicf("RuneIndexer.UpdateDB-> err: status.Height(%d) >= height(%d)", s.Status.Height, s.height)
	}

	s.Status.Height = s.height
	s.Status.Update()
	s.dbWrite.FlushToDB()
	s.isUpdateing = false

	if s.chaincfgParam.Net == wire.MainNet && s.height < 840000 {
		return
	}
	common.Log.Infof("RuneIndexer.UpdateDB-> db commit success, height:%d", s.Status.Height)
}

func (s *Indexer) UpdateTransfer(block *common.Block) {
	if s.chaincfgParam.Net == wire.MainNet && block.Height < 840000 {
		return
	}

	if !s.isUpdateing && block.Height > 0 && s.Status.Height > 0 {
		if s.Status.Height >= uint64(block.Height) {
			common.Log.Infof("RuneIndexer.UpdateTransfer-> cointinue next block, because status.Height(%d) > block.Height(%d)", s.Status.Height, block.Height)
			return
		} else if s.Status.Height < uint64(block.Height-1) {
			common.Log.Panicf("RuneIndexer.UpdateTransfer-> err: status.Height(%d) < block.Height-1(%d), missing intermediate blocks", s.Status.Height, block.Height-1)
		}
	}
	s.height = uint64(block.Height)
	s.isUpdateing = true

	s.HolderUpdateCount = 0
	s.HolderRemoveCount = 0

	s.burnedMap = make(table.RuneIdLotMap)
	s.minimumRune = runestone.MinimumAtHeight(s.chaincfgParam.Net, uint64(block.Height))
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
	format := "RuneIndexer.UpdateTransfer-> handle block succ, tx count:%d, update holder count:%d, remove holder count:%d, block took time:%v"
	common.Log.Infof(format, txCount, s.HolderUpdateCount, s.HolderRemoveCount, sinceTime)
	s.update()
}

func (s *Indexer) index_runes(tx_index uint32, tx *common.Transaction) (isParseOk bool, err error) {
	// if tx.Txid == "9ad1ba215e80ff9a31ef2d261365c5268686fad84493ef8461b5ef4338983d1e" {
	// 	common.Log.Trace("RuneIndexer.index_runes-> location tx")
	// }
	var artifact *runestone.Artifact
	artifact, err = parseArtifact(tx)
	if err != nil {
		if err != runestone.ErrNoOpReturn {
			common.Log.Tracef("RuneIndexer.index_runes-> parseArtifact(%s) err:%s", tx.Txid, err.Error())
		}
	} else {
		common.Log.Tracef("RuneIndexer.index_runes-> parseArtifact(%s) ok, tx_index:%d, artifact:%+v", tx.Txid, tx_index, artifact)
	}
	// if artifact != nil && artifact.Runestone != nil && artifact.Runestone.Edicts != nil {
	// 	common.Log.Infof("%v", artifact.Runestone.Etching)
	// }

	unallocated := s.unallocated(tx)

	type RuneIdLotMapVec map[uint32]table.RuneIdLotMap
	allocated := make(RuneIdLotMapVec, len(tx.Outputs))
	for outputIndex := range tx.Outputs {
		allocated[uint32(outputIndex)] = make(table.RuneIdLotMap)
	}

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
				unallocated.GetOrDefault(mintRuneId).AddAssign(mintAmount) // 铸造
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
				unallocated.GetOrDefault(etchedId).AddAssign(premineAmount) // 预分配
			}

			zeroId := runestone.RuneId{Block: uint64(0), Tx: uint32(0)}
			for _, edict := range artifact.Runestone.Edicts {
				amount := runestone.NewLot(&edict.Amount)
				// edicts with output values greater than the number of outputs
				// should never be produced by the edict parser
				output := edict.Output
				if output > uint32(len(tx.Outputs)) {
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
				// if edict.ID.Cmp(zeroId) == 0 {
				// 	if etchedRune == nil {
				// 		common.Log.Panicf("RuneIndexer.index_runes-> etched rune not found")
				// 	}
				// } else {
					runeEntry := s.idToEntryTbl.Get(id)
					if runeEntry == nil {
						common.Log.Panicf("RuneIndexer.index_runes-> rune entry not found")
					}
				//}

				allocate := func(balance *runestone.Lot, amount *runestone.Lot, output uint32) {
					if amount.Value.Cmp(uint128.Zero) > 0 {
						balance.SubAssign(*amount)
						allocated[output].GetOrDefault(id).AddAssign(amount)
					}
				}

				if output == uint32(len(tx.Outputs)) {
					// 广播分配
					// find non-OP_RETURN outputs
					var destinations []uint32
					for outputIndex, output := range tx.Outputs {
						if output.Address.PkScript[0] != txscript.OP_RETURN {
							destinations = append(destinations, uint32(outputIndex))
						}
					}
					if len(destinations) > 0 {
						if amount.Value.Cmp(uint128.Zero) == 0 {
							// 平均分配
							destinationsLen := uint128.From64(uint64(len(destinations)))
							amount := balance.Div(&destinationsLen)
							remainder := balance.Rem(&destinationsLen).Value.Big().Uint64()
							for index, output := range destinations {
								if index < int(remainder) {
									one := uint128.From64(1)
									addAmount := amount.AddUint128(&one)
									allocate(balance, &addAmount, output)
								} else {
									allocate(balance, &amount, output)
								}
							}
						} else {
							// 按指定量分配
							for _, output := range destinations {
								var lot *runestone.Lot
								if balance.Cmp(&amount.Value) > 0 {
									lot = runestone.NewLot(&amount.Value)
								} else {
									lot = balance
								}
								allocate(balance, lot, output)
							}
						}
					}
				} else {
					// 单一分配
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
			newRuneEntry := s.create_rune_entry(tx, artifact, etchedId, etchedRune)
			s.idToEntryTbl.Insert(etchedId, newRuneEntry)
		}
	}

	burned := make(table.RuneIdLotMap)

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
			common.Log.Panicf("RuneIndexer.index_runes-> pointer out of range") // 无效的符文，前面应该已经设置为Cenotaph
		}
		if find {
			for id, balance := range unallocated {
				if balance.Value.Cmp(uint128.Zero) > 0 {
					allocated[*outIndex].GetOrDefault(&id).AddAssign(balance) // 
				}
			}
		} else {
			for id, balance := range unallocated {
				if balance.Value.Cmp(uint128.Zero) > 0 {
					burned.GetOrDefault(&id).AddAssign(balance) // 没有有效的输出，直接烧了
				}
			}
		}
	}

	type RuneIdOutpointAddressToBalance struct {
		RuneId    *runestone.RuneId
		OutPoint  *table.OutPoint
		AddressId uint64
		Address   runestone.Address
		Balance   runestone.Lot
		OutIndex  uint32
	}
	type RuneBalanceArray []*RuneIdOutpointAddressToBalance
	runeBalanceArray := make(RuneBalanceArray, 0)

	// update outpoint balances
	for vout, balances := range allocated {
		if len(balances) == 0 {
			continue
		}
		// for _, balance := range balances {
		// 	if balance.Value.Cmp(uint128.Zero) == 0 {
		// 		common.Log.Panicf("RuneIndexer.index_runes-> balance is zero")
		// 	}
		// }
		// increment burned balances
		if tx.Outputs[vout].Address.PkScript[0] == txscript.OP_RETURN {
			for id, balance := range balances {
				burned.GetOrDefault(&id).AddAssign(balance)
			}
			continue
		}
		// Sort balanceArray by id so tests can assert balanceArray in a fixed order
		outpoint := &table.OutPoint{UtxoId: common.GetUtxoId(tx.Outputs[vout])}
		address, err := parseTxVoutScriptAddress(tx, int(vout), *s.chaincfgParam)
		if err != nil {
			common.Log.Panicf("RuneIndexer.index_runes-> parseTxVoutScriptAddress(%v,%v,%v) err:%v",
				tx.Txid, vout, s.chaincfgParam.Net, err)
		}
		addressId := s.BaseIndexer.GetAddressId(string(address))
		outpointToBalancesValue := &table.OutpointToBalancesValue{
			UtxoId:     outpoint.UtxoId,
			AddressId:  addressId,
			RuneIdLots: balances.GetSortArray(),
		}
		s.outpointToBalancesTbl.Insert(outpoint, outpointToBalancesValue)

		// update runeIdToOutputMap and runeIdToAddressMap
		for runeId, balance := range balances {
			if balance.Value.Cmp(uint128.Zero) > 0 {
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

	// if artifact != nil && artifact.Runestone == nil { 有默认的转移
	// 	return
	// }

	// if len(burned) > 0 { // 有燃烧的符文，不影响其他正常转移的符文
	// 	return
	// }

	// add for balances and holder count
	for _, runeBalance := range runeBalanceArray {
		// update runeIdToOutpointToBalance
		runeIdToOutpointToBalance := &table.RuneIdOutpointToBalance{
			RuneId:   runeBalance.RuneId,
			OutPoint: runeBalance.OutPoint,
			Balance:  runeBalance.Balance,
		}
		s.runeIdOutpointToBalanceTbl.Insert(runeIdToOutpointToBalance)

		// update addressOutpointToBalance
		// addressOutpointToBalance := &table.AddressOutpointToBalance{
		// 	AddressId: runeBalance.AddressId,
		// 	OutPoint:  runeBalance.OutPoint,
		// 	RuneId:    runeBalance.RuneId,
		// 	Balance:   runeBalance.Balance,
		// }
		
		runeIdAddressToCountKey := &table.RuneIdAddressToCount{
			RuneId:    runeBalance.RuneId,
			AddressId: runeBalance.AddressId,
		}
		runeIdAddressToCountValue := s.runeIdAddressToCountTbl.Remove(runeIdAddressToCountKey)
		if runeIdAddressToCountValue == nil {
			runeIdAddressToCountValue = &table.RuneIdAddressToCount{
				RuneId:    runeBalance.RuneId,
				AddressId: runeBalance.AddressId,
				Count:     0,
			}
		}
		runeIdAddressToCountValue.Count++
		s.runeIdAddressToCountTbl.Insert(runeIdAddressToCountValue)
		if runeIdAddressToCountValue.Count == 1 {
			r := s.idToEntryTbl.Remove(runeBalance.RuneId)
			r.HolderCount++
			s.HolderUpdateCount++
			s.idToEntryTbl.Insert(runeBalance.RuneId, r)
			common.Log.Tracef("insert addressid %d, block %d, HolderCount: %d", runeBalance.AddressId, runeBalance.RuneId.Block, r.HolderCount)
		} else {
			common.Log.Tracef("update addressid %d, block %d, HolderCount: %d", runeBalance.AddressId, runeBalance.RuneId.Block, runeIdAddressToCountValue.Count)
		}
		//s.addressOutpointToBalancesTbl.Insert(addressOutpointToBalance)
	}

	// clean and sub for balances
	for _, runeBalance := range runeBalanceArray {
		key := &table.RuneIdAddressToBalance{
			RuneId:    runeBalance.RuneId,
			AddressId: runeBalance.AddressId,
		}

		value := s.runeIdAddressToBalanceTbl.Get(key)
		if value != nil {
			value.Balance.AddAssign(&runeBalance.Balance)
		} else {
			value = &table.RuneIdAddressToBalance{
				RuneId:    runeBalance.RuneId,
				AddressId: runeBalance.AddressId,
				Balance:   runeBalance.Balance,
			}
		}
		s.runeIdAddressToBalanceTbl.Insert(value)
	}

	// update runeIdToMintHistory
	if mintAmount != nil && artifact.Runestone != nil {
		if outIndex == nil {
			common.Log.Panicf("RuneIndexer.index_runes-> mintOutIndex is nil")
		}
		output := tx.Outputs[*outIndex]
		utxoId := common.GetUtxoId(output)
		address, err := parseTxVoutScriptAddress(tx, int(*outIndex), *s.chaincfgParam)
		if err != nil {
			common.Log.Panicf("RuneIndexer.index_runes-> parseTxVoutScriptAddress(%v,%v,%v) err:%v",
				tx.Txid, outIndex, s.chaincfgParam.Net, err)
		} else {
			addressId := s.BaseIndexer.GetAddressId(string(address))
			v := &table.RuneIdToMintHistory{
				RuneId:    mintRuneId,
				UtxoId:    utxoId,
				AddressId: addressId,
				Amount:    *mintAmount,
			}
			s.runeIdToMintHistoryTbl.Insert(v)
		}
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
	OutPoint  *table.OutPoint
	AddressId uint64
}

func (s *Indexer) unallocated(tx *common.Transaction) (ret1 table.RuneIdLotMap) {
	ret1 = make(table.RuneIdLotMap)
	for _, input := range tx.Inputs {
		outpoint := &table.OutPoint{
			UtxoId: input.UtxoId,
		}
		oldValue := s.outpointToBalancesTbl.Remove(outpoint)
		if oldValue != nil {
			for _, val := range oldValue.RuneIdLots {
				if ret1[val.RuneId] == nil {
					ret1[val.RuneId] = runestone.NewLot(&uint128.Uint128{Lo: 0, Hi: 0})
				}
				ret1[val.RuneId].AddAssign(&val.Lot)

				runeIdOutpointToBalance := &table.RuneIdOutpointToBalance{
					RuneId:   &val.RuneId,
					OutPoint: outpoint,
				}
				s.runeIdOutpointToBalanceTbl.Remove(runeIdOutpointToBalance)

				runeIdAddressToCountKey := &table.RuneIdAddressToCount{
					RuneId:    &val.RuneId,
					AddressId: oldValue.AddressId,
					//Address:   runestone.Address(oldValue.Address),
				}
				runeIdAddressToCountValue := s.runeIdAddressToCountTbl.Remove(runeIdAddressToCountKey)
				if runeIdAddressToCountValue != nil {
					if runeIdAddressToCountValue.Count-1 == 0 {
						oldRuneEntry := s.idToEntryTbl.Remove(&val.RuneId)
						common.Log.Tracef("remove addressid %d, block %d, HolderCount: %d", oldValue.AddressId, val.RuneId.Block, oldRuneEntry.HolderCount-1)
						if oldRuneEntry.HolderCount == 0 {
							common.Log.Panic("unallocated-> oldRuneEntry.HolderCount == 0")
						}
						oldRuneEntry.HolderCount--
						s.HolderRemoveCount++
						s.idToEntryTbl.Insert(&val.RuneId, oldRuneEntry)
					} else {
						runeIdAddressToCountValue.Count--
						s.runeIdAddressToCountTbl.Insert(runeIdAddressToCountValue)
					}
				}

				// addressOutpointToBalance := &table.AddressOutpointToBalance{
				// 	AddressId: oldValue.AddressId,
				// 	OutPoint:  outpoint,
				// }
				// s.addressOutpointToBalancesTbl.Remove(addressOutpointToBalance)

				key := &table.RuneIdAddressToBalance{RuneId: &val.RuneId, AddressId: oldValue.AddressId}
				oldruneIdAddressToBalanceValue := s.runeIdAddressToBalanceTbl.Get(key)
				if oldruneIdAddressToBalanceValue == nil {
					common.Log.Panicf("address %s has missing rune %s in tx %s", input.Address.Addresses[0], val.RuneId.String(), tx.Txid)
				}
				var amount uint128.Uint128 = uint128.Uint128{Lo: 0, Hi: 0}
				if oldruneIdAddressToBalanceValue.Balance.Value.Cmp(val.Lot.Value) < 0 {
					//amount = uint128.Zero
					common.Log.Panicf("address %s has incorrect rune value in tx %s", input.Address.Addresses[0], tx.Txid)
				} else {
					amount = oldruneIdAddressToBalanceValue.Balance.Value.Sub(val.Lot.Value)
				}
				if !amount.IsZero() {
					oldruneIdAddressToBalanceValue.Balance.Value = amount
					s.runeIdAddressToBalanceTbl.Insert(oldruneIdAddressToBalanceValue)
				} else {
					s.runeIdAddressToBalanceTbl.Remove(oldruneIdAddressToBalanceValue)
				}
				
			}
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

		instructions := parseTapscriptLegacyInstructions(tapscript, commitment)
		for _, instruction := range instructions {
			// ignore errors, since the extracted script may not be valid
			if !bytes.Equal(instruction, commitment) {
				continue
			}

			var err error
			var resp any
			for {
				resp, err = bitcoin_rpc.ShareBitconRpc.GetTx(input.Txid)
				if err == nil {
					break
				} else {
					time.Sleep(1 * time.Second)
					common.Log.Infof("RuneIndexer.txCommitsToRune-> bitcoin_rpc.GetRawTransaction failed, try again ...")
					continue
				}
			}

			txInfo, _ := resp.(*bitcoind.RawTransaction)
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
			blockHeader, err := bitcoin_rpc.ShareBitconRpc.GetBlockHeader(txInfo.BlockHash)
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
