FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/ethereumai/go-ethereumai && \
  (cd go-ethereumai && make geai) && \
  cp go-ethereumai/build/bin/geai /geai && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-ethereumai && rm -rf /var/cache/apk/*

EXPOSE 9545
EXPOSE 40404

ENTRYPOINT ["/geai"]
