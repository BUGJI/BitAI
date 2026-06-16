#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT/backend"
DATA_DIR="$BACKEND_DIR/data"
LOG_DIR="$ROOT/logs"
ENV_FILE="$BACKEND_DIR/.env.local"
BACKEND_PORT=8091

FORCE=0
ADMIN_EMAIL=""
ADMIN_NAME=""
ADMIN_PASSWORD=""
APP_NAME=""
HTTP_ADDR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --force)
      FORCE=1
      shift
      ;;
    --admin-email)
      ADMIN_EMAIL="${2:-}"
      shift 2
      ;;
    --admin-name)
      ADMIN_NAME="${2:-}"
      shift 2
      ;;
    --admin-password)
      ADMIN_PASSWORD="${2:-}"
      shift 2
      ;;
    --app-name)
      APP_NAME="${2:-}"
      shift 2
      ;;
    --http-addr)
      HTTP_ADDR="${2:-}"
      shift 2
      ;;
    *)
      echo "未知参数：$1" >&2
      exit 1
      ;;
  esac
done

read_required() {
  local prompt="$1"
  local default_value="${2:-}"
  local value=""
  while true; do
    if [[ -n "$default_value" ]]; then
      read -r -p "$prompt [$default_value]: " value
      value="${value:-$default_value}"
    else
      read -r -p "$prompt: " value
    fi
    if [[ -n "${value// }" ]]; then
      printf '%s' "$value"
      return
    fi
    echo "该项不能为空，请重新输入。"
  done
}

read_plain_password() {
  local prompt="$1"
  local value=""
  while true; do
    read -r -p "$prompt: " value
    if [[ -n "$value" ]]; then
      printf '%s' "$value"
      return
    fi
    echo "密码不能为空，请重新输入。"
  done
}

new_secret() {
  local bytes="$1"
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 "$bytes"
  elif command -v python3 >/dev/null 2>&1; then
    python3 - "$bytes" <<'PY'
import base64
import os
import sys
print(base64.b64encode(os.urandom(int(sys.argv[1]))).decode())
PY
  else
    echo "需要安装 openssl 或 python3，用于生成安全密钥。" >&2
    exit 1
  fi
}

stop_port() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    local pids
    pids="$(lsof -ti "tcp:$port" || true)"
    if [[ -n "$pids" ]]; then
      kill -9 $pids 2>/dev/null || true
    fi
  elif command -v fuser >/dev/null 2>&1; then
    fuser -k "${port}/tcp" >/dev/null 2>&1 || true
  fi
}

http_ok() {
  local url="$1"
  if command -v curl >/dev/null 2>&1; then
    curl -fsS "$url" >/dev/null 2>&1
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "$url" >/dev/null 2>&1
  elif command -v python3 >/dev/null 2>&1; then
    python3 - "$url" <<'PY'
import sys
from urllib.request import urlopen
try:
    with urlopen(sys.argv[1], timeout=2) as resp:
        raise SystemExit(0 if 200 <= resp.status < 300 else 1)
except Exception:
    raise SystemExit(1)
PY
  else
    return 1
  fi
}

load_env() {
  local path="$1"
  [[ -f "$path" ]] || return 0
  while IFS='=' read -r key value; do
    [[ -z "${key// }" || "${key:0:1}" == "#" ]] && continue
    key="$(echo "$key" | xargs)"
    export "$key=$value"
  done < "$path"
}

echo "BitAPI 初始化向导"
echo "注意：初始化会停止本机 ${BACKEND_PORT} 端口服务，并清空 backend/data/bitapi.db 数据库。"
if [[ "$FORCE" != "1" ]]; then
  read -r -p "请输入 YES 继续初始化，输入其它内容将取消: " confirm
  if [[ "$confirm" != "YES" ]]; then
    echo "已取消初始化。"
    exit 0
  fi
fi

[[ -n "$APP_NAME" ]] || APP_NAME="$(read_required "请输入站点/产品名称" "BitAPI")"
[[ -n "$ADMIN_EMAIL" ]] || ADMIN_EMAIL="$(read_required "请输入管理员账号邮箱" "admin@bitapi.local")"
[[ -n "$ADMIN_NAME" ]] || ADMIN_NAME="$(read_required "请输入管理员名称" "${APP_NAME} 管理员")"
[[ -n "$ADMIN_PASSWORD" ]] || ADMIN_PASSWORD="$(read_plain_password "请输入管理员密码，输入过程会直接显示，请认真核对")"
[[ -n "$HTTP_ADDR" ]] || HTTP_ADDR="$(read_required "请输入服务监听地址" ":${BACKEND_PORT}")"

stop_port 8080
stop_port 5173
stop_port 5181
stop_port "$BACKEND_PORT"

mkdir -p "$DATA_DIR/uploads/avatars" "$LOG_DIR"
rm -f "$DATA_DIR/bitapi.db" "$DATA_DIR/bitapi.db-shm" "$DATA_DIR/bitapi.db-wal"

JWT_SECRET="$(new_secret 48)"
ENCRYPTION_KEY="$(new_secret 32)"

cat > "$ENV_FILE" <<EOF
BITAPI_APP_NAME=$APP_NAME
BITAPI_ENV=production
BITAPI_HTTP_ADDR=$HTTP_ADDR
BITAPI_DATABASE_DSN=file:data/bitapi.db?_foreign_keys=on&_busy_timeout=5000
BITAPI_JWT_SECRET=$JWT_SECRET
BITAPI_ACCESS_TOKEN_TTL=30m
BITAPI_REFRESH_TOKEN_TTL=336h
BITAPI_CORS_ORIGINS=http://localhost:${BACKEND_PORT},http://127.0.0.1:${BACKEND_PORT},https://www.bitit.cn,https://bitit.cn,https://demo.bitit.cn,https://*.bitit.cn
BITAPI_BOOTSTRAP_EMAIL=$ADMIN_EMAIL
BITAPI_BOOTSTRAP_PASSWORD=$ADMIN_PASSWORD
BITAPI_BOOTSTRAP_NAME=$ADMIN_NAME
BITAPI_ENCRYPTION_KEY=$ENCRYPTION_KEY
BITAPI_DEFAULT_USER_BALANCE_MICROS=0
EOF
chmod 600 "$ENV_FILE" || true

load_env "$ENV_FILE"

INIT_OUT="$LOG_DIR/init-backend.log"
INIT_ERR="$LOG_DIR/init-backend.err.log"
echo "正在创建数据库表和管理员账号..."
(
  cd "$BACKEND_DIR"
  go run ./cmd/server
) >"$INIT_OUT" 2>"$INIT_ERR" &
PID=$!

DB_PATH="$DATA_DIR/bitapi.db"
DEADLINE=$((SECONDS + 90))
while [[ $SECONDS -lt $DEADLINE ]]; do
  if [[ -f "$DB_PATH" ]] && http_ok "http://127.0.0.1:${BACKEND_PORT}/health"; then
    break
  fi
  if ! kill -0 "$PID" 2>/dev/null; then
    echo "后端服务提前退出，数据库没有创建成功。请查看：$INIT_ERR" >&2
    exit 1
  fi
  sleep 1
done

if [[ ! -f "$DB_PATH" ]]; then
  echo "数据库创建超时。请查看：$INIT_ERR" >&2
  kill "$PID" 2>/dev/null || true
  exit 1
fi
if ! http_ok "http://127.0.0.1:${BACKEND_PORT}/health"; then
  echo "后端健康检查超时。请查看：$INIT_ERR" >&2
  kill "$PID" 2>/dev/null || true
  exit 1
fi

kill "$PID" 2>/dev/null || true
stop_port "$BACKEND_PORT"

echo "初始化完成。"
echo "管理员账号邮箱：$ADMIN_EMAIL"
echo "管理员密码：初始化时输入的密码"
echo "管理员名称：$ADMIN_NAME"
echo "本地登录地址：http://localhost:${BACKEND_PORT}/auth/login"
echo "配置文件位置：$ENV_FILE"
echo "提示：熟悉命令行的用户也可以使用 --admin-email --admin-name --admin-password --http-addr 参数。"
