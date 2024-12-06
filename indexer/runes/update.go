package runes

import (
	"time"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/runes/runestone"
	"lukechampine.com/uint128"
)

func (s *Indexer) UpdateTransfer(block *common.Block) {
	for txIndex, transaction := range block.Transactions {
		parent := tryGetFirstInscriptionId(transaction)
		// handle runestone
		artifact, voutIndex, err := parserArtifact(transaction)
		if err != nil {
			common.Log.Errorf("RuneIndexer->UpdateTransfer: parserArtifact error: %v", err)
			continue
		}

		if artifact.Runestone != nil {
			if artifact.Runestone.Etching != nil { // etching， 当 mint 时，必须要存在这个Etching，否则MintError::Unmintable
				r := artifact.Runestone.Etching.Rune
				(*s.runeMap)[runestone.RuneId{
					Block: uint64(block.Height),
					Tx:    uint32(voutIndex),
				}] = artifact.Runestone.Etching.Rune

				runeInfo := (*s.runeInfoMap)[*r]
				if runeInfo != nil {
					continue
				}
				runeInfo = &RuneInfo{
					Etching: artifact.Runestone.Etching,
					Parent:  parent,
				}
				// pass rust source code
				if runeInfo.Etching.Divisibility == nil {
					zero := uint8(0)
					runeInfo.Etching.Divisibility = &zero
				}
				// pass rust source code
				if runeInfo.Etching.Premine == nil {
					runeInfo.Etching.Premine = &uint128.Zero
				}
				if runeInfo.Etching.Rune == nil {
					r := runestone.Reserved(uint64(block.Height), uint32(txIndex))
					runeInfo.Etching.Rune = &r
				}
				// pass rust source code
				if runeInfo.Etching.Spacers == nil {
					zero := uint32(0)
					runeInfo.Etching.Spacers = &zero
				}
				// runeInfo.Etching.Symbol: default is None,可选项，如果没有可以不显示，一般都是有的, 另外显示 pile时会默认显示 ¤
				if runeInfo.Etching.Symbol == nil {
					runeInfo.Etching.Symbol = nil
				}
				// Terms: 可选项 没有设置，显示时就是无
				if runeInfo.Etching.Terms == nil {
					runeInfo.Etching.Terms = &runestone.Terms{
						// Amount: &one, // 可选项 没有设置，显示时就是不显示或者无，在 mint时，amount为 0，也是可以 mint操作的
						// Cap:    &uint128.Max, // 可选项，没有设置时， 显示就是 0
						// Height: [2]*uint64{nil, nil}, // 可选项，没有设置时，显示就是无
						// Offset: [2]*uint64{nil, nil}, // 可选项，没有设置时，显示就是无
					}
				}

				(*s.runeInfoMap)[*r] = runeInfo

				address, err := parseTxVoutScriptAddress(transaction, voutIndex, *s.chaincfgParam)
				if err != nil {
					common.Log.Errorf("RuneIndexer->UpdateTransfer: parseTxVoutScriptAddress error: %v", err)
					continue
				}
				addressAsset := (*s.addressAssetMap)[address]
				if addressAsset == nil {
					addressAsset = &AddressAsset{}
					(*s.addressAssetMap)[address] = addressAsset
				}
				if addressAsset.Mints == nil {
					addressAsset.Mints = &MintMap{}
				}
				if addressAsset.Transfers == nil {
					addressAsset.Transfers = &TransferMap{}
				}
				if addressAsset.Cenotaphs == nil {
					addressAsset.Cenotaphs = &CenotaphMap{}
				}
				if addressAsset.Assets == nil {
					addressAsset.Assets = &AssetMap{}
				}
				asset := (*addressAsset.Assets)[*r]
				if asset == nil {
					asset = &Asset{
						IsEtching: true,
					}
					(*addressAsset.Assets)[*r] = asset
				}
			}

			if artifact.Runestone.Mint != nil { // mint
				r := (*s.runeMap)[*artifact.Runestone.Mint]
				runeInfo := (*s.runeInfoMap)[*r]
				if runeInfo == nil {
					common.Log.Errorf("RuneIndexer->UpdateTransfer: runeInfo is nil, rune: %s", r)
					continue
				}
				(*s.mintMap)[*r] = append((*s.mintMap)[*r], artifact.Runestone.Mint)
				address, err := parseTxVoutScriptAddress(transaction, voutIndex, *s.chaincfgParam)
				if err != nil {
					common.Log.Errorf("RuneIndexer->UpdateTransfer: parseTxVoutScriptAddress error: %v", err)
					continue
				}
				addressAsset := (*s.addressAssetMap)[address]
				if addressAsset == nil {
					common.Log.Errorf("RuneIndexer->UpdateTransfer: addressAsset is nil, address: %s", address)
					continue
				}
				if addressAsset.Mints == nil {
					addressAsset.Mints = &MintMap{}
				}
				(*addressAsset.Mints)[*r] = append((*addressAsset.Mints)[*r], artifact.Runestone.Mint)
				if addressAsset.Assets == nil {
					addressAsset.Assets = &AssetMap{}
				}
				asset := (*addressAsset.Assets)[*r]
				if asset == nil {
					asset = &Asset{}
					(*addressAsset.Assets)[*r] = asset
				}
				runeInfo = (*s.runeInfoMap)[*r]
				if runeInfo == nil {
					common.Log.Errorf("RuneIndexer->UpdateTransfer: runeInfo is nil, rune: %s", r)
					continue
				}

				asset.Amount.Add(*(runeInfo.Etching.Terms.Amount))

			}

			if len(artifact.Runestone.Edicts) > 0 { // transfer

			}

		} else if artifact.Cenotaph != nil {

		}
	}

}

func (p *Indexer) UpdateDB() {
	//common.Log.Infof("OrdxIndexer->UpdateDB start...")
	startTime := time.Now()

	wb := p.db.NewWriteBatch()
	defer wb.Cancel()

	// for _, v := range p.tickerAdded {
	// 	key := GetTickerKey(v.Name)
	// 	err := common.SetDB([]byte(key), v, wb)
	// 	if err != nil {
	// 		common.Log.Panicf("Error setting %s in db %v", key, err)
	// 	}
	// }

	// for _, ticker := range p.runesMap {
	// 	for _, v := range ticker.MintAdded {
	// 		key := GetMintHistoryKey(ticker.Name, v.Base.InscriptionId)
	// 		err := common.SetDB([]byte(key), v, wb)
	// 		if err != nil {
	// 			common.Log.Panicf("Error setting %s in db %v", key, err)
	// 		}
	// 	}
	// }

	err := wb.Flush()
	if err != nil {
		common.Log.Panicf("Error ordxwb flushing writes to db %v", err)
	}

	// reset memory buffer
	// p.tickerAdded = make(map[string]*common.Ticker)
	// for _, info := range p.runesMap {
	// 	info.MintAdded = make([]*common.Mint, 0)
	// }

	common.Log.Infof("OrdxIndexer->UpdateDB takse: %v", time.Since(startTime))
}
