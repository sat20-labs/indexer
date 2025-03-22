package dkvs

import (
	"errors"
	"fmt"
)

// ErrExpiredRecord should be returned when an ipns record is
// invalid due to being too old
var ErrExpiredRecord = errors.New("expired record")

// ErrUnrecognizedValidity is returned when an IpnsRecord has an
// unknown validity type.
var ErrUnrecognizedValidity = errors.New("unrecognized validity type")

// ErrInvalidPath should be returned when an ipns record path
// is not in a valid format
var ErrInvalidPath = errors.New("record path invalid")

// ErrSignature should be returned when an ipns record fails
// signature verification
var ErrSignature = errors.New("record signature verification failed")

// ErrKeyFormat should be returned when an ipns record key is
// incorrectly formatted (not a peer ID)
var ErrKeyFormat = errors.New("record key could not be parsed into peer ID")

// ErrPublicKeyNotFound should be returned when the public key
// corresponding to the ipns record path cannot be retrieved
// from the peer store
var ErrPublicKeyNotFound = errors.New("public key not found in peer store")

// ErrPublicKeyMismatch should be returned when the public key embedded in the
// record doesn't match the expected public key.
var ErrPublicKeyMismatch = errors.New("public key in record did not match expected pubkey")

// ErrBadRecord should be returned when an ipns record cannot be unmarshalled
var ErrBadRecord = errors.New("record could not be unmarshalled")


// ErrRecordSize should be returned when an ipns record is
// invalid due to being too big
var ErrRecordSize = errors.New("record exceeds allowed size limit")


// The new record is not the same as the existing record
var ErrDifferentPublicKey = errors.New("the public key of the new record is not the same as the existing record, so it cannot be set")

var ErrPublicIsNil = errors.New("public is nil in record")

// ErrLookupFailure is returned if a routing table query returns no results. This is NOT expected
// behaviour
var ErrLookupFailure = errors.New("failed to find any peer in table")

// ErrNotFound is returned when the router fails to find the requested record.
var ErrNotFound = errors.New("routing: not found")


var ErrKeyLength = errors.New("the key length is less than the default minimum key length of " + fmt.Sprintf("%d", MinDKVSRecordKeyLength) + " bytes.")

// Transfer failed
var ErrTranferFailed = errors.New("transfer failed")


var ErrInvalidValueType = errors.New("value type is invalid")


var ErrTTL = errors.New("wrong TTL value")
var ErrExceededMaxTTL = errors.New("he maximum validity period(TTL) is more than 100 years")

var ErrInvalidRecord = errors.New("record is invalid")