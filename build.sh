#!/usr/bin/env bash

rm -f indexer-mainnet
rm -f indexer-testnet

db_backend="${INDEXER_DB_BACKEND:-pebble}"
case "$db_backend" in
  pebble)
    build_tags=()
    ;;
  badger)
    build_tags=(-tags badger)
    ;;
  *)
    echo "unsupported INDEXER_DB_BACKEND=$db_backend (expected pebble or badger)" >&2
    exit 1
    ;;
esac

go build "${build_tags[@]}" -ldflags="-s -w" -o indexer-mainnet

cp indexer-mainnet indexer-testnet

echo build completed.
