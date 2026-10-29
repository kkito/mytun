#!/usr/bin/env bash
# 加载项目环境变量，然后执行传入的命令。
# 用法：scripts/load_env.sh <cmd> [args...]
#
# 覆盖顺序（后者优先）：
#   已存在的 shell 环境变量  >  .env.local  >  .env
#
# 支持：注释(#)、空行、KEY=VALUE、单/双引号、值中含 '='。
set -uo pipefail

# 默认锚定项目根；MYTUN_ENV_ROOT 可覆盖（测试/特殊调用用）。
ROOT="${MYTUN_ENV_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

# 读取一个 env 文件；已存在的变量不覆盖（因此先读的 .env.local 会拦住 .env）。
load_file() {
  local file="$1" line key val
  [ -f "$file" ] || return 0
  while IFS= read -r line || [ -n "$line" ]; do
    line="${line#"${line%%[![:space:]]*}"}"   # 去左侧空白
    case "$line" in ''|'#'*) continue ;; esac
    line="${line#export }"
    [[ "$line" == *=* ]] || continue
    key="${line%%=*}"
    val="${line#*=}"
    key="${key%"${key##*[![:space:]]}"}"       # 去 key 右侧空白
    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || continue
    # 去掉成对引号
    if   [[ "$val" == \"*\" && ${#val} -ge 2 ]]; then val="${val:1:${#val}-2}"
    elif [[ "$val" == \'*\' && ${#val} -ge 2 ]]; then val="${val:1:${#val}-2}"
    fi
    [ -n "${!key+x}" ] && continue             # 已存在：不覆盖
    export "$key=$val"
  done < "$file"
}

# 注意：先去重比，.env.local 先加载以获得更高优先级。
load_file "$ROOT/.env.local"
load_file "$ROOT/.env"

[ $# -gt 0 ] || exit 0
exec "$@"
