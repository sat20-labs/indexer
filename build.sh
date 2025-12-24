#!/usr/bin/env bash

rm -f indexer-mainnet
rm -f indexer-testnet

go build -ldflags="-s -w" -o indexer-mainnet

cp indexer-mainnet indexer-testnet

echo build completed.