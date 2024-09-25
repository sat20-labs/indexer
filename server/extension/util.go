package extension

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/sat20-labs/ordx/common"
	mainCommon "github.com/sat20-labs/ordx/main/common"
	"github.com/sat20-labs/ordx/share/base_indexer"
	"github.com/sat20-labs/ordx/share/bitcoin_rpc"
)

func getOrdContentUrl(inscriptionId string) (string, string) {
	previewUrl := ""
	contentUrl := ""
	txid, index, err := common.ParseOrdInscriptionID(inscriptionId)
	if err != nil {
		return "", ""
	}
	if base_indexer.ShareBaseIndexer.IsMainnet() {
		previewUrl = OrdMainnetPreviewUrl
		contentUrl = OrdMainnetContentUrl
	} else {
		previewUrl = OrdTestnet4PreviewUrl
		contentUrl = OrdTestnet4ContentUrl
		// previewUrl = OrdTestnetPreviewUrl
		// contentUrl = OrdTestnetContentUrl
	}
	previewUrl = previewUrl + txid + "i" + strconv.Itoa(index)
	contentUrl = contentUrl + txid + "i" + strconv.Itoa(index)
	return previewUrl, contentUrl
}

func calcOffset(sat int64, rngs []*common.Range) int64 {
	offset := int64(0)
	for _, rng := range rngs {
		if sat >= rng.Start && sat < rng.Start+rng.Size {
			return offset + sat - rng.Start
		}
		offset += rng.Size
	}
	return -1
}

func GetScriptPK(address string) string {
	chain := mainCommon.GetChain()
	pkScript, _ := common.AddrToPkScript(address, chain)
	return hex.EncodeToString(pkScript)
}

func newInscription(nft *common.Nft) *Inscription {
	utxo, rngs, err := base_indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(nft.UtxoId)
	if err != nil {
		common.Log.Errorf("%v", err)
		return nil
	}
	txid, index, err := common.ParseUtxo(utxo)
	if err != nil {
		common.Log.Errorf("%v", err)
		return nil
	}

	preview, content := getOrdContentUrl(nft.Base.InscriptionId)
	unspendTxoutPut, err := bitcoin_rpc.ShareBitconRpc.GetUnspendTxOutput(txid, index, false)
	if err != nil {
		common.Log.Errorf("%v", err)
		return nil
	}
	output := txid + ":" + strconv.Itoa(index)

	utxoConfirmation := unspendTxoutPut.Confirmations
	genesisTransaction := txid

	return &Inscription{
		Id:                 nft.Base.InscriptionId,
		Number:             nft.Base.Id,
		Address:            base_indexer.ShareBaseIndexer.GetAddressById(nft.OwnerAddressId),
		OutputValue:        uint64(common.GetOrdinalsSize(rngs)),
		Preview:            preview,
		Content:            content,
		ContentType:        string(nft.Base.ContentType),
		ContentLength:      uint(len(nft.Base.Content)),
		Timestamp:          nft.Base.BlockTime,
		GenesisTransaction: genesisTransaction,
		Location:           output + ":" + strconv.Itoa(int(calcOffset(nft.Base.Sat, rngs))),
		Output:             output,
		ContentBody:        string(nft.Base.Content),
		Height:             int64(nft.Base.BlockHeight),
		Confirmation:       utxoConfirmation,
	}
}

func newAbbrInscription(nft *common.Nft) *Inscription {
	_, rngs, err := base_indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(nft.UtxoId)
	if err != nil {
		common.Log.Errorf("%v", err)
		return nil
	}

	return &Inscription{
		Id:     nft.Base.InscriptionId,
		Number: nft.Base.Id,
		Offset: calcOffset(nft.Base.Sat, rngs),
	}
}

func newUtxoDataWithId(utxoId uint64, address string, bAvailable bool) *Utxo {
	utxo, rngs, err := base_indexer.ShareBaseIndexer.GetOrdinalsWithUtxoId(utxoId)
	if err != nil {
		common.Log.Errorf("%v", err)
		return nil
	}

	txid, vout, err := common.ParseUtxo(utxo)
	if err != nil {
		common.Log.Errorf("%v", err)
		return nil
	}

	data := &Utxo{
		Txid:         txid,
		Vout:         vout,
		Satoshis:     uint64(common.GetOrdinalsSize(rngs)),
		ScriptPk:     (GetScriptPK(address)),
		AddressType:  P2TR,
		Inscriptions: make([]*Inscription, 0),
		// Atomicals:    make([]*Atomical, 0),
		Runes: make([]*Rune, 0),
	}
	if !bAvailable {
		assets := base_indexer.ShareBaseIndexer.GetAssetsWithUtxo(utxoId)
		for ticker, mintinfo := range assets {
			if ticker.TypeName == common.ASSET_TYPE_EXOTIC {
				// TODO 稀有聪需要有所表示出来
				continue
			}
			for id := range mintinfo {
				nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(id)
				data.Inscriptions = append(data.Inscriptions, newAbbrInscription(nft))
			}
		}
	}

	return data
}

func newUtxoDataWithInscription(inscriptionId string) *Utxo {
	nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(inscriptionId)
	if nft == nil {
		return nil
	}
	return newUtxoDataWithId(nft.UtxoId, base_indexer.ShareBaseIndexer.GetAddressById(nft.OwnerAddressId), true)
}

func getInsctiptionList(utxo string) ([]*Inscription, error) {
	utxoId := base_indexer.ShareBaseIndexer.GetUtxoId(utxo)
	if utxoId == common.INVALID_ID {
		return nil, fmt.Errorf("can't find utxo %s", utxo)
	}
	inscriptionIdList := base_indexer.ShareBaseIndexer.GetNftsWithUtxo(utxoId)
	inscriptionList := make([]*Inscription, 0)
	for _, inscriptionId := range inscriptionIdList {
		nft := base_indexer.ShareBaseIndexer.GetNftInfoWithInscriptionId(inscriptionId)
		if nft != nil {
			inscriptionList = append(inscriptionList, newAbbrInscription(nft))
		}
	}
	return inscriptionList, nil
}
