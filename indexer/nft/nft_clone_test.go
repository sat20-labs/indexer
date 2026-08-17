package nft

import (
	"reflect"
	"testing"

	"github.com/sat20-labs/indexer/common"
)

func TestCloneAndSubtractPreserveContentTypeLookupState(t *testing.T) {
	source := NewNftIndexer(nil)
	source.status = &common.NftStatus{ContentTypeCount: 2}
	source.contentTypeMap = map[int]string{
		1: "text/plain",
		2: "image/png",
	}
	source.contentTypeToIdMap = map[string]int{
		"text/plain": 1,
		"image/png":  2,
	}
	source.lastContentTypeId = 1

	snapshot := source.Clone(nil)
	if snapshot.lastContentTypeId != 1 {
		t.Fatalf("snapshot lastContentTypeId = %d, want 1", snapshot.lastContentTypeId)
	}

	// Simulate a new MIME type allocated after prepareDBBuffer. Subtract must
	// advance only to the snapshot boundary and retain the newer lookup entry.
	source.status.ContentTypeCount = 3
	source.contentTypeMap[3] = "application/json"
	source.contentTypeToIdMap["application/json"] = 3

	source.Subtract(snapshot)
	if got := source.contentTypeMap[1]; got != "text/plain" {
		t.Fatalf("contentTypeMap[1] = %q, want text/plain", got)
	}
	if got := source.contentTypeToIdMap["image/png"]; got != 2 {
		t.Fatalf("contentTypeToIdMap[image/png] = %d, want 2", got)
	}
	if got := source.contentTypeToIdMap["application/json"]; got != 3 {
		t.Fatalf("new content type id = %d, want 3", got)
	}
	if source.lastContentTypeId != 2 {
		t.Fatalf("live lastContentTypeId = %d, want flushed boundary 2", source.lastContentTypeId)
	}
}

func TestContentTypeIDsToPersist(t *testing.T) {
	tests := []struct {
		name          string
		lastPersisted int
		current       int
		want          []int
	}{
		{name: "none", lastPersisted: 5, current: 5, want: nil},
		{name: "one new type", lastPersisted: 5, current: 6, want: []int{6}},
		{name: "multiple new types", lastPersisted: 5, current: 8, want: []int{6, 7, 8}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := contentTypeIDsToPersist(test.lastPersisted, test.current)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("contentTypeIDsToPersist(%d, %d) = %v, want %v",
					test.lastPersisted, test.current, got, test.want)
			}
		})
	}
}
