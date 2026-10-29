#!/bin/bash
# Linux 端到端自测：在有 /dev/net/tun 的容器里起 mytun，ping 它，断言 echo 通路。
# 用法：./scripts/linux_e2e.sh [addr]   （宿主机执行，需 docker + colima running）
set -euo pipefail
cd "$(dirname "$0")/.."

ADDR="${1:-10.10.0.1/24}"
IP="${ADDR%%/*}"
IMG="debian:bookworm-slim"
NAME="mytun-e2e-$$"

echo "==> build linux binary"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/mytun-linux-amd64 ./cmd/mytun

echo "==> start container $NAME"
docker run -d --rm --name "$NAME" --privileged --cap-add=NET_ADMIN \
  -v "$PWD/dist:/dist:ro" "$IMG" sleep infinity >/dev/null
cleanup() { docker rm -f "$NAME" >/dev/null 2>&1 || true; }
trap cleanup EXIT

echo "==> install iproute2 + iputils-ping"
docker exec "$NAME" bash -c "apt-get update -qq && apt-get install -y -qq iproute2 iputils-ping >/dev/null && echo tools-ok"

echo "==> run mytun in background"
docker exec -d "$NAME" /dist/mytun-linux-amd64 --name mytun0 --addr "$ADDR"
sleep 2

echo "==> interface check"
docker exec "$NAME" ip addr show mytun0

echo "==> ping test (3 packets)"
docker exec "$NAME" ping -c 3 -W 2 "$IP"

echo "==> PASS: tun create + addr + icmp echo all ok"
