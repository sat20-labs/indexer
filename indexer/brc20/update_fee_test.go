package brc20

import (
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func TestResolveTransferNftLocationUsesNftID(t *testing.T) {
	transfer := &TransferNftInfo{
		TransferNft: &common.TransferNFT{
			NftId: 101,
			Id:    7,
		},
	}
	destination := &common.Nft{
		OwnerAddressId: 0x22,
		UtxoId:         common.ToUtxoId(1000, 0, 0),
	}

	var requestedID int64 = -1
	got := resolveTransferNftLocation(transfer, func(id int64) *common.Nft {
		requestedID = id
		return destination
	})

	if requestedID != transfer.TransferNft.NftId {
		t.Fatalf("lookup used id %d, want nft id %d", requestedID, transfer.TransferNft.NftId)
	}
	if requestedID == transfer.TransferNft.Id {
		t.Fatalf("lookup incorrectly used protocol-local transfer id %d", requestedID)
	}
	if got != destination {
		t.Fatalf("unexpected destination: got %p, want %p", got, destination)
	}
}

func TestSpendInvalidTransferNftAsFee(t *testing.T) {
	tests := []struct {
		name       string
		nftID      int64
		protocolID int64
		amount     int64
	}{
		{
			name:       "mint inscription first spend becomes fee",
			nftID:      201,
			protocolID: 3,
			amount:     1000,
		},
		{
			name:       "transfer inscription second spend becomes fee",
			nftID:      202,
			protocolID: 9,
			amount:     250,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const (
				ticker       = "ordi"
				ownerAddress = uint64(0x11)
				minerAddress = uint64(0x22)
			)
			oldUtxoID := common.ToUtxoId(900, 1, 0)
			coinbaseUtxoID := common.ToUtxoId(901, 0, 0)

			transferNFT := &common.TransferNFT{
				NftId:     tt.nftID,
				Id:        tt.protocolID,
				UtxoId:    oldUtxoID,
				Amount:    *common.NewDefaultDecimal(tt.amount),
				IsInvalid: true,
			}
			transfer := &TransferNftInfo{
				TxInIndex:   2,
				AddressId:   ownerAddress,
				UtxoId:      oldUtxoID,
				Ticker:      ticker,
				TransferNft: transferNFT,
			}

			availableBefore := common.NewDefaultDecimal(5000)
			transferableBefore := common.NewDefaultDecimal(0)
			tickInfo := &common.BRC20TickAbbrInfo{
				AvailableBalance:    availableBefore.Clone(),
				TransferableBalance: transferableBefore.Clone(),
				TransferableData: map[uint64]*common.TransferNFT{
					oldUtxoID: transferNFT,
				},
			}
			holder := &HolderInfo{
				Tickers: map[string]*common.BRC20TickAbbrInfo{
					ticker: tickInfo,
				},
			}
			indexer := &BRC20Indexer{
				holderMap: map[uint64]*HolderInfo{
					ownerAddress: holder,
				},
				transferNftMap: map[uint64]*TransferNftInfo{
					oldUtxoID: transfer,
				},
				holderActionList: make([]*HolderAction, 0, 1),
			}
			destination := &common.Nft{
				OwnerAddressId: minerAddress,
				UtxoId:         coinbaseUtxoID,
				Offset:         17,
			}

			indexer.spendInvalidTransferNft(transfer, 901, 4, destination)

			if _, ok := indexer.transferNftMap[oldUtxoID]; ok {
				t.Fatalf("terminal handle for spent utxo %d was not removed", oldUtxoID)
			}
			if _, ok := tickInfo.TransferableData[oldUtxoID]; ok {
				t.Fatalf("holder transferable data for spent utxo %d was not removed", oldUtxoID)
			}
			if tickInfo.AvailableBalance.Cmp(availableBefore) != 0 {
				t.Fatalf("available balance changed: got %s, want %s",
					tickInfo.AvailableBalance.String(), availableBefore.String())
			}
			if tickInfo.TransferableBalance.Cmp(transferableBefore) != 0 {
				t.Fatalf("transferable balance changed: got %s, want %s",
					tickInfo.TransferableBalance.String(), transferableBefore.String())
			}
			if !transferNFT.IsInvalid {
				t.Fatal("terminal handle unexpectedly became valid")
			}
			if holder.FreshTime != 1 {
				t.Fatalf("holder update count = %d, want 1", holder.FreshTime)
			}

			if len(indexer.holderActionList) != 1 {
				t.Fatalf("holder action count = %d, want 1", len(indexer.holderActionList))
			}
			action := indexer.holderActionList[0]
			if action.Action != common.BRC20_Action_Transfer_Spent {
				t.Fatalf("action = %d, want Transfer_Spent", action.Action)
			}
			if action.Height != 901 || action.TxIndex != 4 || action.TxInIndex != transfer.TxInIndex {
				t.Fatalf("unexpected action position: %+v", action)
			}
			if action.NftId != transferNFT.NftId {
				t.Fatalf("action nft id = %d, want %d", action.NftId, transferNFT.NftId)
			}
			if action.FromUtxoId != oldUtxoID || action.ToUtxoId != coinbaseUtxoID {
				t.Fatalf("unexpected utxo transition: %d -> %d", action.FromUtxoId, action.ToUtxoId)
			}
			if action.FromAddr != ownerAddress || action.ToAddr != minerAddress {
				t.Fatalf("unexpected address transition: %x -> %x", action.FromAddr, action.ToAddr)
			}
			if action.Ticker != ticker || action.Amount.Cmp(&transferNFT.Amount) != 0 {
				t.Fatalf("unexpected action asset: %+v", action)
			}
		})
	}
}
