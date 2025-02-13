#!/bin/sh
cd server
GOOS=linux
GOARCH=amd64
go build -ldflags="-s -w" -trimpath -o ../dist/
cd ..