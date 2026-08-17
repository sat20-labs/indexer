package brc20

import "github.com/sat20-labs/indexer/common"

func resolveTransferNftLocation(transfer *TransferNftInfo,
	lookup func(int64) *common.Nft) *common.Nft {

	nft := lookup(transfer.TransferNft.NftId)
	if nft == nil {
		common.Log.Panicf("can't find transfer nft %d", transfer.TransferNft.NftId)
	}
	return nft
}

// spendInvalidTransferNft removes a terminal BRC-20 handle whose inscription
// sat was routed to a miner output as transaction fee. The BRC-20 balance has
// already reached its terminal state, so this only removes tracking state and
// records the same Transfer_Spent history used by the ordinary-output path.
func (s *BRC20Indexer) spendInvalidTransferNft(transfer *TransferNftInfo,
	height, index int, nft *common.Nft) {

	fromAddress := transfer.AddressId
	oldUtxoId := transfer.UtxoId

	s.removeTransferNft(transfer)

	action := HolderAction{
		Height:     height,
		TxIndex:    index,
		TxInIndex:  transfer.TxInIndex,
		NftId:      transfer.TransferNft.NftId,
		FromUtxoId: oldUtxoId,
		FromAddr:   fromAddress,
		ToAddr:     nft.OwnerAddressId,
		ToUtxoId:   nft.UtxoId,
		Ticker:     transfer.Ticker,
		Amount:     *transfer.TransferNft.Amount.Clone(),
		Action:     common.BRC20_Action_Transfer_Spent,
	}
	s.holderActionList = append(s.holderActionList, &action)

	common.Log.Debugf("spend %d as fee: %x -> %x, ticker = %s, %s",
		action.NftId, action.FromAddr, action.ToAddr, action.Ticker, action.Amount.String())
}
