package ordx

import (
	"testing"

	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/share/base_indexer"
)

type atomicalsHistoryIndexer struct {
	base_indexer.Indexer
	queried common.AssetName
}

func (i *atomicalsHistoryIndexer) GetMintHistoryV2(name *common.AssetName, start, limit int) []*common.MintInfo {
	return []*common.MintInfo{{Amount: "100", InscriptionId: "native-atomical-id", Address: "minter"}}
}

func (i *atomicalsHistoryIndexer) GetMintAmountV2(name *common.AssetName) (*common.Decimal, int64) {
	i.queried = *name
	return common.NewDefaultDecimal(700), 7
}

func (i *atomicalsHistoryIndexer) GetMintAmount(name string) (int64, int64) {
	panic("legacy ORDX-only mint total queried for a protocol-qualified asset")
}

func (i *atomicalsHistoryIndexer) GetHolderAddress(string) string { return "" }

func TestAtomicalsMintHistoryUsesProtocolAwareTotal(t *testing.T) {
	indexer := &atomicalsHistoryIndexer{}
	name := common.AssetName{Protocol: common.PROTOCOL_NAME_ATOM, Type: common.ASSET_TYPE_FT, Ticker: "quark"}
	history, err := NewModel(indexer).GetMintHistoryV3(name.String(), 0, 1)
	if err != nil || history.Total != 7 || len(history.Items) != 1 || indexer.queried != name {
		t.Fatalf("protocol-aware mint history: %+v, %v, queried %+v", history, err, indexer.queried)
	}
}
