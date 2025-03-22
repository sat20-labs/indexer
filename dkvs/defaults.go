package dkvs

import (
	"time"

	"github.com/libp2p/go-libp2p/core/protocol"
)

// 10 KiB limit defined in https://github.com/ipfs/specs/pull/319
const MaxRecordSize int = 10 << (10 * 1)

// DefaultDKVSRecordEOL specifies the time that the network will cache DKVS
// records after being published. Records should be re-published before this
// interval expires. We use the same default expiration as the DHT.
const DefaultDKVSRecordEOL time.Duration = 48 * time.Hour

const MaxDKVSRecordKeyLength int = 256

// Keys with a length of less than 32 bytes are reserved and not allowed, and the service node directly rejects them
const MinDKVSRecordKeyLength int = 32


// Because there will be errors in the time zone and timing accuracy, in order to prevent mistakes, add this error
const MaxTTL time.Duration = time.Hour * 24 * 365 * 100

// DefaultPrefix is the application specific prefix attached to all DHT protocols by default.
const DefaultPrefix protocol.ID = "/dkvs"

const DKVS_NAMESPACE = "dkvs"

const DKVS_UNSYNC_LDB = "unsynckv"

// all keys in dkvs auto added a prefix name
const DefaultKeyPrefix = "/dkvs"
