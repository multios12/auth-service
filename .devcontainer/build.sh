#!/bin/sh

cd front
yarn build
cd ..
cp ./front/build/* ./server/static/ -R
find ./front/build -delete

cd server
GOOS=linux
GOARCH=amd64
go build -ldflags="-s -w" -trimpath -o ../dist/
cd ..