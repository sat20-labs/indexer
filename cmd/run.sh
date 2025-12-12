#!/usr/bin/env bash


go build -o indtest  -ldflags="-s -w"
./indtest

