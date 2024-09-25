#!/bin/bash

# protoc --go_out=. ./ordinals/indexer/pb/*.proto
# protoc --go_out=paths=source_relative:. ./ordinals/indexer/pb/*.proto
# protoc --go_out=paths=source_relative:. ./ordinals/pb/*.proto
protoc --go_out=paths=source_relative:. ./indexer/ns/pb/*.proto
protoc --go_out=paths=source_relative:. ./common/pb/*.proto
