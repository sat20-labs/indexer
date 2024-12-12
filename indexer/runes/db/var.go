package db

import "github.com/dgraph-io/badger/v4"

var db *badger.DB

func SetDB(d *badger.DB) {
	db = d
}

func GetDB() *badger.DB {
	return db
}
