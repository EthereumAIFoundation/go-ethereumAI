.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/ethereumai/go-ethereumai/cmd/swarm
	go build -o bin/geai     github.com/ethereumai/go-ethereumai/cmd/geai
	go build -o bin/bootnode github.com/ethereumai/go-ethereumai/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/ethereumai/go-ethereumai/swarm/...
