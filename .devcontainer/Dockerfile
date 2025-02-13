# DevContainer
FROM node:22-alpine
RUN apk  --no-cache --update add go git tzdata && \
    cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime && \
    apk del tzdata && \
    go install -v golang.org/x/tools/gopls@latest && \
    go install -v github.com/go-delve/delve/cmd/dlv@latest && \
    go install -v honnef.co/go/tools/cmd/staticcheck@latest && \
    wget -P /tmp https://github.com/goreleaser/goreleaser/releases/download/v1.22.1/goreleaser_1.22.1_x86_64.apk && \
    apk add --allow-untrusted --no-network --repositories-file=repo.list /tmp/goreleaser_1.22.1_x86_64.apk && \
    yarn global add purgecss

WORKDIR /workspace
