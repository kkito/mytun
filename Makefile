MODULE := mytun
BIN := mytun
DIST := dist
GOFLAGS ?=

# 加载 .env / .env.local（scripts/load_env.sh），并补齐 NODE_PATH=全局 npm 工具链。
# 已存在的环境变量优先级最高，所以 CI/命令行覆盖仍然生效。
ENVLOAD := ./scripts/load_env.sh
NODE_PATH ?= $(shell npm root -g 2>/dev/null)
export NODE_PATH

.PHONY: all build build-all test test-front test-all env-test vet clean web-build env-check linux-test linux-shell darwin-run

all: build

env-check: ## 打印生效的环境变量（排查构建问题用）
	@echo "GOPROXY=$${GOPROXY:-unset}"
	@$(ENVLOAD) sh -c 'echo "GOPROXY(loaded)=$${GOPROXY:-unset}"; echo "NODE_PATH(loaded)=$${NODE_PATH:-unset}"'
	@echo "go=$$(command -v go || echo MISSING)" "esbuild=$$(command -v esbuild || echo MISSING)" "vitest=$$(command -v vitest || echo MISSING)"

web-build: ## 打包前端（esbuild → build/web-dist + internal/webdist embed）
	node scripts/build_web.cjs

build: web-build
	$(ENVLOAD) go build $(GOFLAGS) -o $(DIST)/$(BIN) ./cmd/mytun

build-all: ## 交叉编译 darwin-arm64 + linux-amd64
	mkdir -p $(DIST)
	$(ENVLOAD) env GOOS=darwin GOARCH=arm64 go build -o $(DIST)/$(BIN)-darwin-arm64 ./cmd/mytun
	$(ENVLOAD) env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(DIST)/$(BIN)-linux-amd64 ./cmd/mytun
	ls -lh $(DIST)

test:
	$(ENVLOAD) go test -count=1 -v ./...

test-front: ## 前端单元测试（vitest + happy-dom，全局工具链）
	vitest run --root web

env-test: ## 验证 .env 加载器
	bash scripts/tests/env_test.sh

test-all: ## 全部测试（go test + vitest + .env 加载器）
	$(ENVLOAD) go test -count=1 ./...
	vitest run --root web
	bash scripts/tests/env_test.sh

vet:
	$(ENVLOAD) go vet ./...

clean:
	rm -rf $(DIST) build

linux-test: build-all ## Linux 一键自测（docker 跑 ping 回环，需 --privileged）
	./scripts/linux_e2e.sh

linux-shell: ## 进 Linux 容器手动玩
	docker run --rm -it --privileged --cap-add=NET_ADMIN \
		-v $(PWD)/$(DIST):/dist $(LINUX_IMG) bash

darwin-run: build ## macOS 本地跑（需 sudo，会提示密码）
	sudo ./$(DIST)/$(BIN) --addr 172.30.0.1/24

LINUX_IMG := debian:bookworm-slim
