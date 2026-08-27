#!/usr/bin/env bash
set -euo pipefail

rm -f indexer-mainnet
rm -f indexer-testnet

db_backend="${INDEXER_DB_BACKEND:-badger}"
case "$db_backend" in
  badger)
    go build -ldflags="-s -w" -o indexer-mainnet
    ;;
  pebble)
    go build -tags pebble -ldflags="-s -w" -o indexer-mainnet
    ;;
  *)
    echo "unsupported INDEXER_DB_BACKEND=$db_backend (expected badger or pebble)" >&2
    exit 1
    ;;
esac

cp indexer-mainnet indexer-testnet

echo "build completed with $db_backend backend."
