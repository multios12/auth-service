#!/bin/sh

#mkdir ./server/staticResource
#wget https://cdn.jsdelivr.net/npm/bulma@1.0.2/css/bulma.min.css -O /tmp/bulma.min.css
#purgecss --css /tmp/bulma.min.css --content ./server/static/login.html  --output /tmp/bulma.css
#v=`cat /tmp/bulma.css`
#cat ./server/static/login.html | sed 's;<link rel="stylesheet".*;$v;'

cd server
GOOS=linux
GOARCH=amd64
go build -ldflags="-s -w" -trimpath -o ./dist/ -tags=release
cd ..
