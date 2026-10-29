MODULE := mytun
BIN := mytun
DIST := dist
GOFLAGS ?=

.PHONY: all build build-all test vet clean linux-test linux-shell darwin-run

all: build

build:
	go build $(GOFLAGS) -o $(DIST)/$(BIN) ./cmd/mytun

build-all: ## 交叉编译 darwin-arm64 + linux-amd64
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=arm64 go build -o $(DIST)/$(BIN)-darwin-arm64 ./cmd/mytun
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(DIST)/$(BIN)-linux-amd64 ./cmd/mytun
	ls -lh $(DIST)

test:
	go test -count=1 -v ./...

vet:
	go vet ./...

clean:
	rm -rf $(DIST)

linux-test: build-all ## Linux 一键自测（docker 跑 ping 回环，需 --privileged）
	./scripts/linux_e2e.sh

linux-shell: ## 进 Linux 容器手动玩
	docker run --rm -it --privileged --cap-add=NET_ADMIN \
		-v $(PWD)/$(DIST):/dist $(LINUX_IMG) bash

darwin-run: build ## macOS 本地跑（需 sudo，会提示密码）
	sudo ./$(DIST)/$(BIN) --addr 10.10.0.1/24

LINUX_IMG := debian:bookworm-slim
