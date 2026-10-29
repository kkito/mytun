#!/usr/bin/env bash
# 验证 .env 加载器：KEY=VALUE、注释、空行、引号、变量覆盖优先级。
set -uo pipefail
LOADER="$(cd "$(dirname "$0")/.." && pwd)/load_env.sh"
fail=0
check() { # desc expected actual
  if [ "$2" = "$3" ]; then echo "ok   - $1"; else echo "FAIL - $1: want [$2] got [$3]"; fail=1; fi
}
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/.env" <<'E'
# 注释行
GOPROXY=https://goproxy.cn,direct

QUOTED="a b"
SINGLE='c d'
EMPTY=
WITH_EQ=x=y=z
E
cat > "$tmp/.env.local" <<'E'
GOPROXY=https://local.example
E

out=$(MYTUN_ENV_ROOT="$tmp" bash "$LOADER" bash -c 'echo "$GOPROXY|$QUOTED|$SINGLE|$EMPTY|$WITH_EQ"')
check "加载 .env 并让 .env.local 覆盖" "https://local.example|a b|c d||x=y=z" "$out"

# 已存在的环境变量优先级最高（不被 .env 覆盖）
out=$(MYTUN_ENV_ROOT="$tmp" GOPROXY=from-shell bash "$LOADER" bash -c 'echo "$GOPROXY"')
check "已存在的环境变量不被覆盖" "from-shell" "$out"

# 无 .env 时不报错
rm -f "$tmp/.env" "$tmp/.env.local"
out=$(MYTUN_ENV_ROOT="$tmp" bash "$LOADER" bash -c 'echo alive')
check "缺少 .env 时静默通过" "alive" "$out"

exit $fail
