#!/usr/bin/env bash
# BitAPI Docker 快速初始化脚本
# 生成安全的随机密钥，创建 .env 文件
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="$ROOT/.env"

if [[ -f "$ENV_FILE" ]]; then
  echo "⚠️  .env 已存在，将被覆盖。"
  read -r -p "输入 YES 继续: " confirm
  if [[ "$confirm" != "YES" ]]; then
    echo "已取消。"
    exit 0
  fi
fi

echo ""
echo "════════════════════════════════════"
echo "  BitAPI Docker 初始化"
echo "════════════════════════════════════"
echo ""

read -r -p "站点名称 [BitAPI]: " APP_NAME
APP_NAME="${APP_NAME:-BitAPI}"

read -r -p "管理员邮箱 [admin@bitapi.local]: " ADMIN_EMAIL
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@bitapi.local}"

read -r -p "管理员昵称 [${APP_NAME} 管理员]: " ADMIN_NAME
ADMIN_NAME="${ADMIN_NAME:-${APP_NAME} 管理员}"

read -r -p "管理员密码 (明文显示，请注意安全) [bitapi-admin]: " ADMIN_PASSWORD
ADMIN_PASSWORD="${ADMIN_PASSWORD:-bitapi-admin}"

read -r -p "对外端口 [8091]: " HTTP_PORT
HTTP_PORT="${HTTP_PORT:-8091}"

JWT_SECRET=$(openssl rand -base64 48 2>/dev/null || python3 -c "import base64,os; print(base64.b64encode(os.urandom(48)).decode())")
ENCRYPTION_KEY=$(openssl rand -base64 32 2>/dev/null || python3 -c "import base64,os; print(base64.b64encode(os.urandom(32)).decode())")

cat > "$ENV_FILE" <<EOF
# BitAPI Docker 环境变量 (由 docker-init.sh 自动生成)
BITAPI_APP_NAME=$APP_NAME
BITAPI_ENV=production
BITAPI_HTTP_PORT=$HTTP_PORT
BITAPI_JWT_SECRET=$JWT_SECRET
BITAPI_ENCRYPTION_KEY=$ENCRYPTION_KEY
BITAPI_ACCESS_TOKEN_TTL=30m
BITAPI_REFRESH_TOKEN_TTL=336h
BITAPI_CORS_ORIGINS=http://localhost:$HTTP_PORT,http://127.0.0.1:$HTTP_PORT
BITAPI_BOOTSTRAP_EMAIL=$ADMIN_EMAIL
BITAPI_BOOTSTRAP_PASSWORD=$ADMIN_PASSWORD
BITAPI_BOOTSTRAP_NAME=$ADMIN_NAME
BITAPI_DEFAULT_USER_BALANCE_MICROS=0
EOF
chmod 600 "$ENV_FILE"

echo ""
echo "════════════════════════════════════"
echo "  ✅ 初始化完成！"
echo "════════════════════════════════════"
echo ""
echo "接下来执行："
echo "  docker compose up -d --build"
echo ""
echo "登录地址："
echo "  http://localhost:${HTTP_PORT}/auth/login"
echo ""
echo "管理员账号：${ADMIN_EMAIL}"
echo "管理员密码：${ADMIN_PASSWORD}"