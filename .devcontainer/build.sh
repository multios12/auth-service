#!/bin/sh

cd front
yarn build
cd ..
cp ./front/dist/* ./server/static/ -R
find ./front/dist -delete

cd server
GOOS=linux
GOARCH=amd64
go build -ldflags="-s -w" -trimpath -o ../dist/
cd ..