#!/usr/bin/env bash

rm indexer-mainnet
go build -o indexer-mainnet

if [ $# -eq 0 ]; then
  nohup ./indexer-mainnet &
  disown
else
  if [ "$1" = "off" ]; then
    ./indexer-mainnet
  else
    echo "unknown parameter"
  fi
fi

