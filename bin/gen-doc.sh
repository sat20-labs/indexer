#!/bin/bash

go get -u github.com/swaggo/swag/cmd/swag
go mod tidy
swag init server/router.go
