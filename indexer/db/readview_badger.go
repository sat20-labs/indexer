//go:build badger

package db

import (
	"bytes"
	"errors"
	"sync"

	"github.com/dgraph-io/badger/v4"
	"github.com/sat20-labs/indexer/common"
)

type badgerPinnedView struct {
	mutex sync.Mutex
	txn *badger.Txn
}

func (b *badgerDB) NewReadView() (common.DBReadView, error) {
	if b.db.IsClosed() { return nil, errors.New("database closed") }
	return &badgerPinnedView{txn: b.db.NewTransaction(false)}, nil
}

func (b *badgerDB) Sync() error { return b.db.Sync() }

func (v *badgerPinnedView) Read(key []byte) ([]byte, error) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if v.txn == nil { return nil, errors.New("database read view closed") }
	item, err := v.txn.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) { return nil, common.ErrKeyNotFound }
	if err != nil { return nil, err }
	return item.ValueCopy(nil)
}

func (v *badgerPinnedView) Scan(options common.ScanOptions, visit func([]byte, []byte) error) error {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if v.txn == nil { return errors.New("database read view closed") }
	opts := badger.DefaultIteratorOptions
	opts.Reverse = options.Reverse
	opts.Prefix = options.Prefix
	opts.PrefetchValues = !options.KeysOnly
	if options.PrefetchSize > 0 { opts.PrefetchSize = options.PrefetchSize }
	it := v.txn.NewIterator(opts)
	defer it.Close()
	if len(options.Start) > 0 {
		it.Seek(options.Start)
		if it.Valid() && !options.StartInclusive && bytes.Equal(it.Item().Key(), options.Start) { it.Next() }
	} else if options.Reverse {
		if upper := badgerPrefixSuccessor(options.Prefix); upper != nil {
			it.Seek(upper)
			if it.Valid() && !bytes.HasPrefix(it.Item().Key(), options.Prefix) { it.Next() }
		} else { it.Rewind() }
	} else if len(options.Prefix) > 0 { it.Seek(options.Prefix) } else { it.Rewind() }
	count := 0
	for ; it.Valid(); it.Next() {
		item := it.Item()
		key := item.Key()
		if len(options.Prefix) > 0 && !bytes.HasPrefix(key, options.Prefix) { break }
		if options.CopyKey { key = item.KeyCopy(nil) }
		var err error
		if options.KeysOnly { err = visit(key, nil) } else {
			err = item.Value(func(value []byte) error {
				if options.CopyValue { value = append([]byte(nil), value...) }
				return visit(key, value)
			})
		}
		if err != nil {
			if errors.Is(err, common.ErrStopScan) { return nil }
			return err
		}
		count++
		if options.Limit > 0 && count >= options.Limit { break }
	}
	return nil
}

func (v *badgerPinnedView) Close() error {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	if v.txn != nil { v.txn.Discard(); v.txn = nil }
	return nil
}

var _ common.SnapshotDB = (*badgerDB)(nil)
