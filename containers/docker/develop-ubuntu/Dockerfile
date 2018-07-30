FROM ubuntu:xenial

ENV PATH=/usr/lib/go-1.9/bin:$PATH

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends golang-1.9 git make gcc libc-dev ca-certificates && \
  git clone --depth 1 https://github.com/ethereumai/go-ethereumai && \
  (cd go-ethereumai && make geai) && \
  cp go-ethereumai/build/bin/geai /geai && \
  apt-get remove -y golang-1.9 git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /go-ethereumai

EXPOSE 9545
EXPOSE 40404

ENTRYPOINT ["/geai"]
