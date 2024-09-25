#!/usr/bin/env bash

rm indexer-testnet
go build -o indexer-testnet

if [ $# -eq 0 ]; then
  nohup ./indexer-testnet &
  disown
else
  if [ "$1" = "off" ]; then
    ./indexer-testnet
  else
    echo "unknown parameter"
  fi
fi

