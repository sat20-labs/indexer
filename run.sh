#!/usr/bin/env bash

rm indexer-mainnet
go build -o indexer-mainnet

if [ $# -eq 0 ]; then
  nohup ./indexer-mainnet -env ./mainnet.env > ./nohup_mainnet.log 2>&1 &
  disown
else
  if [ "$1" = "off" ]; then
    ./indexer-mainnet -env ./mainnet.env
  else
    echo "unknown parameter"
  fi
fi

