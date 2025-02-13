#!/usr/bin/env bash

rm indexer-testnet
go build -o indexer-testnet

if [ $# -eq 0 ]; then
  nohup ./indexer-testnet -env ./testnet.env &
  disown
else
  if [ "$1" = "off" ]; then
    ./indexer-testnet -env ./testnet.env
  else
    echo "unknown parameter"
  fi
fi

