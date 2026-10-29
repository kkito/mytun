#!/bin/bash
# macOS 本地跑：sudo 起 mytun，后台 ping 3 个包验证 echo，最后清理。
# 用法：sudo ./scripts/darwin_e2e.sh [addr]
set -euo pipefail
cd "$(dirname "$0")/.."

if [ "$(id -u)" != "0" ]; then echo "请用 sudo 跑：sudo $0"; exit 1; fi

ADDR="${1:-10.10.0.1/24}"
IP="${ADDR%%/*}"

echo "==> build darwin binary"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
go build -o dist/mytun ./cmd/mytun

echo "==> start mytun ($ADDR)"
./dist/mytun --addr "$ADDR" > /tmp/mytun.log 2>&1 &
PID=$!
cleanup() { kill $PID 2>/dev/null || true; wait $PID 2>/dev/null || true; echo "--- mytun log tail ---"; tail -20 /tmp/mytun.log; }
trap cleanup EXIT
sleep 2

echo "==> ping test"
ping -c 3 -t 2 "$IP"

echo "==> PASS"
