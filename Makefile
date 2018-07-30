# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geai android ios geai-cross swarm evm all test clean
.PHONY: geai-linux geai-linux-386 geai-linux-amd64 geai-linux-mips64 geai-linux-mips64le
.PHONY: geai-linux-arm geai-linux-arm-5 geai-linux-arm-6 geai-linux-arm-7 geai-linux-arm64
.PHONY: geai-darwin geai-darwin-386 geai-darwin-amd64
.PHONY: geai-windows geai-windows-386 geai-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

geai:
	build/env.sh go run build/ci.go install ./cmd/geai
	@echo "Done building."
	@echo "Run \"$(GOBIN)/geai\" to launch geai."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/geai.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geai.framework\" to use the library."

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

geai-cross: geai-linux geai-darwin geai-windows geai-android geai-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/geai-*

geai-linux: geai-linux-386 geai-linux-amd64 geai-linux-arm geai-linux-mips64 geai-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-*

geai-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/geai
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep 386

geai-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/geai
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep amd64

geai-linux-arm: geai-linux-arm-5 geai-linux-arm-6 geai-linux-arm-7 geai-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep arm

geai-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/geai
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep arm-5

geai-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/geai
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep arm-6

geai-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/geai
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep arm-7

geai-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/geai
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep arm64

geai-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/geai
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep mips

geai-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/geai
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep mipsle

geai-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/geai
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep mips64

geai-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/geai
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/geai-linux-* | grep mips64le

geai-darwin: geai-darwin-386 geai-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/geai-darwin-*

geai-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/geai
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/geai-darwin-* | grep 386

geai-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/geai
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geai-darwin-* | grep amd64

geai-windows: geai-windows-386 geai-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/geai-windows-*

geai-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/geai
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/geai-windows-* | grep 386

geai-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/geai
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/geai-windows-* | grep amd64
