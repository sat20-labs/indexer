//go:build !badger

package db

import (
	"bytes"
	"errors"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/sat20-labs/indexer/common"
)

type pebblePinnedView struct {
	mutex sync.Mutex
	snapshot *pebble.Snapshot
}

func (p *pebbleDB) NewReadView() (common.DBReadView, error) {
	return &pebblePinnedView{snapshot: p.db.NewSnapshot()}, nil
}

func (p *pebbleDB) Sync() error {
	return p.db.LogData(nil, pebble.Sync)
}

func (v *pebblePinnedView) Read(key []byte) ([]byte, error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if v.snapshot == nil { return nil, errors.New("database read view closed") }
	value, closer, err := v.snapshot.Get(key)
	if errors.Is(err, pebble.ErrNotFound) { return nil, common.ErrKeyNotFound }
	if err != nil { return nil, err }
	defer closer.Close()
	return append([]byte(nil), value...), nil
}

func (v *pebblePinnedView) Scan(options common.ScanOptions, visit func([]byte, []byte) error) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if v.snapshot == nil { return errors.New("database read view closed") }
	opts := &pebble.IterOptions{LowerBound: options.Prefix, UpperBound: nextPrefix(options.Prefix)}
	it, err := v.snapshot.NewIter(opts)
	if err != nil { return err }
	defer it.Close()
	valid := false
	if options.Reverse {
		if len(options.Start) == 0 { valid = it.Last() } else if options.StartInclusive {
			valid = it.SeekGE(options.Start)
			if !valid { valid = it.Last() } else if bytes.Compare(it.Key(), options.Start) > 0 { valid = it.Prev() }
		} else { valid = it.SeekLT(options.Start) }
	} else if len(options.Start) == 0 { valid = it.First() } else {
		valid = it.SeekGE(options.Start)
		if valid && !options.StartInclusive && bytes.Equal(it.Key(), options.Start) { valid = it.Next() }
	}
	count := 0
	for valid {
		key := it.Key()
		if len(options.Prefix) > 0 && !bytes.HasPrefix(key, options.Prefix) { break }
		if options.CopyKey { key = append([]byte(nil), key...) }
		var value []byte
		if !options.KeysOnly {
			value = it.Value()
			if options.CopyValue { value = append([]byte(nil), value...) }
		}
		if err := visit(key, value); err != nil {
			if errors.Is(err, common.ErrStopScan) { return it.Error() }
			return err
		}
		count++
		if options.Limit > 0 && count >= options.Limit { break }
		if options.Reverse { valid = it.Prev() } else { valid = it.Next() }
	}
	return it.Error()
}

func (v *pebblePinnedView) Close() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if v.snapshot == nil { return nil }
	err := v.snapshot.Close()
	v.snapshot = nil
	return err
}

var _ common.SnapshotDB = (*pebbleDB)(nil)
