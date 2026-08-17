#!/usr/bin/env bash

export INDEXER_DB_BACKEND=badger

if ! ./build.sh; then
  exit 1
fi

if [ $# -eq 0 ]; then
  nohup ./indexer-testnet -env ./testnet.env > ./nohup_testnet.log 2>&1 &
  disown
else
  if [ "$1" = "off" ]; then
    ./indexer-testnet -env ./testnet.env
  else
    echo "unknown parameter"
  fi
fi
