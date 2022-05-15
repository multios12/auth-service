# DevContainer
FROM node:16-alpine
RUN apk  --no-cache --update add go git tzdata && \
    cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime && \
    apk del tzdata

WORKDIR /workspace/
VOLUME [ "/workspace/front/node_modules" ]

RUN mkdir /workspace/front
COPY ./front/package.json /workspace/front/
RUN yarn --cwd /workspace/front

ENTRYPOINT [ "yarn","--cwd","/workspace/front", "start" ]