package runes

import (
	"bytes"
	"strings"

	"github.com/OLProtocol/go-bitcoind"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"github.com/sat20-labs/indexer/share/bitcoin_rpc"
)

func (s *Indexer) UpdateDB() {}

func (s *Indexer) UpdateTransfer(block *common.Block) {
	minimum := runestone.MinimumAtHeight(s.chaincfgParam.Net, uint64(block.Height))
	s.minimumRune = &minimum
	s.height = uint64(block.Height)
	for txIndex, transaction := range block.Transactions {
		err := s.index_runes(uint32(txIndex), transaction)
		if err != nil {
			common.Log.Debugf("RuneIndexer->UpdateTransfer: index_runes error: %v", err)
		}
		s.status.Height = uint64(block.Height)
		s.status.Update()
	}

}

func (s *Indexer) index_runes(tx_index uint32, tx *common.Transaction) error {
	// parent := tryGetFirstInscriptionId(tx)
	// txid := tx.Txid
	artifact, _, err := parserArtifact(tx)
	if err != nil {
		if err != runestone.ErrNoOpReturn {
			common.Log.Debugf("RuneIndexer->index_runes: parserArtifact error: %v", err)
		} else {
			common.Log.Debugf("RuneIndexer->index_runes: parserArtifact no op return")
		}
		return nil
	}
	unallocated := s.unallocated(tx)
	// allocated := make(runestone.RuneIdLogMapVec, len(tx.Outputs))

	if artifact != nil {
		mintRuneId := artifact.Mint()
		if mintRuneId != nil {
			amount, err := s.mint(mintRuneId)
			if err != nil {
				return err
			}
			unallocated.GetOrDefault(mintRuneId).Add(*amount)
		}
	}
	_, _, err = s.etched(tx_index, tx, artifact)
	if err != nil {
		return err
	}
	return nil
}

func (s *Indexer) unallocated(tx *common.Transaction) (ret runestone.RuneIdLotMap) {
	ret = make(runestone.RuneIdLotMap)
	for _, input := range tx.Inputs {
		outpointKey := &runestone.OutpointToBalanceKey{
			Txid: input.Txid,
			Vout: uint32(input.Vout),
		}
		oldValue, err := s.outpointToBalancesTbl.Remove(outpointKey)
		if err != nil {
			common.Log.Errorf("RuneIndexer->unallocated: outpointToBalancesTbl.Remove error: %v", err)
			continue
		}

		for _, val := range oldValue {
			ret[val.RuneId.ToByte()] = &val.Lot
		}

	}
	return
}

func (s *Indexer) mint(runeId *runestone.RuneId) (lot *runestone.Lot, err error) {
	runeEntry, err := s.runeIdToEntryTbl.Get(runeId)
	if err != nil {
		return nil, err
	}

	amount, err := runeEntry.Mintable(s.height)
	if err != nil {
		return nil, err
	}

	runeEntry.Mints.Add64(1)

	_, err = s.runeIdToEntryTbl.Insert(runeId, *runeEntry)
	if err != nil {
		return nil, err
	}

	lot = &runestone.Lot{
		Value: amount,
	}
	return
}

func (s *Indexer) etched(
	txIndex uint32,
	tx *common.Transaction,
	artifact *runestone.Artifact,
) (runeId *runestone.RuneId, runeData *runestone.Rune, err error) {
	if artifact.Runestone != nil {
		runeData = artifact.Runestone.Etching.Rune
		if runeData == nil {
			reserved_runes := s.status.ReservedRunes
			s.status.ReservedRunes = reserved_runes + 1
			s.status.Update()
			runeData = runestone.Reserved(s.height, txIndex)
			runeId = &runestone.RuneId{
				Block: s.height,
				Tx:    txIndex,
			}
			return runeId, runeData, nil
		} else {
			if runeData.Value.Cmp(s.minimumRune.Value) < 0 {
				return nil, nil, nil
			}
			if runeData.IsReserved() {
				return nil, nil, nil
			}
			val, err := s.runeToRuneIdTbl.Get(runeData)
			if err != nil {
				return nil, nil, err
			}
			if val == nil {
				return nil, nil, nil
			}
			isCommitsToRune, err := s.txCommitsToRune(tx, *runeData)
			if err != nil {
				return nil, nil, err
			}
			if !isCommitsToRune {
				return nil, nil, nil
			}
			return &runestone.RuneId{
				Block: s.height,
				Tx:    txIndex,
			}, runeData, nil
		}
	} else if artifact.Cenotaph != nil {
		runeData = artifact.Cenotaph.Etching
		if runeData == nil {
			return nil, nil, nil
		}
	}
	return nil, nil, nil
}

func (s *Indexer) txCommitsToRune(transaction *common.Transaction, rune runestone.Rune) (bool, error) {
	commitment := rune.Commitment()
	for _, input := range transaction.Inputs {
		tapscript, err := parseTapscript(input.Witness)
		if err != nil {
			continue
		}

		if !bytes.Equal(tapscript, commitment) {
			continue
		}

		resp, err := bitcoin_rpc.ShareBitconRpc.GetRawTransaction(input.Txid, true)
		if err != nil {
			return false, err
		}
		txInfo, _ := resp.(bitcoind.RawTransaction)
		taproot := strings.HasPrefix(txInfo.Vout[input.Vout].ScriptPubKey.Asm, "OP_1")
		// taproot := strings.HasPrefix(txInfo.Vout[input.Vout].ScriptPubKey.Hex, "51")

		if !taproot {
			continue
		}
		blockHeader, err := bitcoin_rpc.ShareBitconRpc.GetBlockheader(txInfo.BlockHash)
		if err != nil {
			return false, err
		}

		commitTxHeight := uint64(blockHeader.Height)
		currentBlockHeight, err := bitcoin_rpc.ShareBitconRpc.GetBlockCount()
		if err != nil {
			return false, err
		}
		confirmations := currentBlockHeight - commitTxHeight + 1

		if confirmations >= 6 {
			return true, nil
		}
	}

	return false, nil
}
