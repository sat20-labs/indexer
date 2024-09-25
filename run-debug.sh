#!/usr/bin/env bash

rm indexer-debug
go build -o indexer-debug

if [ $# -eq 0 ]; then
  nohup ./indexer-debug &
  disown
else
  if [ "$1" = "off" ]; then
    ./indexer-debug
  else
    echo "unknown parameter"
  fi
fi
