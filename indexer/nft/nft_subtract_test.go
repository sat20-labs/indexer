package nft

import (
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func TestSubtractKeepsUpdatedCollectionAndGallery(t *testing.T) {
	idx := NewNftIndexer(nil)
	idx.status = &common.NftStatus{}
	idx.utxoMap = make(map[uint64]map[int64]int64)
	idx.satMap = make(map[int64]*SatInfo)
	idx.contentMap = make(map[uint64]string)
	idx.contentToIdMap = make(map[string]uint64)
	idx.addedContentIdMap = make(map[uint64]bool)
	idx.inscriptionToNftIdMap = make(map[string]*common.Nft)
	idx.nftIdToinscriptionMap = make(map[int64]*common.Nft)
	idx.contentTypeMap = make(map[int]string)
	idx.contentTypeToIdMap = make(map[string]int)
	idx.nftAdded = make([]*common.Nft, 0)
	idx.utxoDeled = make([]uint64, 0)
	idx.collectionMap = map[int64]*CollectionInfo{
		1: {Id: 1, NftId: 1, InscriptionId: "collection", Items: []int64{10}},
		3: {Id: 3, NftId: 3, InscriptionId: "unchanged-collection", Items: []int64{30}},
	}
	idx.galleryMap = map[int64]*GalleryInfo{
		2: {Id: 2, NftId: 2, InscriptionId: "gallery", Items: []int64{20}},
		4: {Id: 4, NftId: 4, InscriptionId: "unchanged-gallery", Items: []int64{40}},
	}

	backup := idx.Clone(nil)
	idx.collectionMap[1].Items = append(idx.collectionMap[1].Items, 11)
	idx.galleryMap[2].Items = append(idx.galleryMap[2].Items, 21)

	idx.Subtract(backup)

	if got := idx.collectionMap[1]; got == nil || len(got.Items) != 2 || got.Items[1] != 11 {
		t.Fatalf("updated collection should remain pending after subtract: %+v", got)
	}
	if got := idx.galleryMap[2]; got == nil || len(got.Items) != 2 || got.Items[1] != 21 {
		t.Fatalf("updated gallery should remain pending after subtract: %+v", got)
	}
	if got := idx.collectionMap[3]; got != nil {
		t.Fatalf("unchanged collection should be removed after subtract: %+v", got)
	}
	if got := idx.galleryMap[4]; got != nil {
		t.Fatalf("unchanged gallery should be removed after subtract: %+v", got)
	}
}
