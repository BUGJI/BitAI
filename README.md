# BitAPI Release 使用教程

BitAPI 是一个中文 AI API Gateway，后端使用 Go、Gin、GORM、SQLite，前端使用 Vue 3、TypeScript、Arco Design Vue。

本发布版采用手动初始化模式：发布目录不内置数据库，也不内置默认管理员密码。首次部署必须运行初始化脚本，由部署者自己填写站点名称、管理员账号、名称和密码。

初始化后也可以在后台 `系统设置 -> 产品品牌配置` 里继续修改站点名称、Logo 和版权信息。

## 一、目录说明

```text
BitAPI_Release
├─ backend/              后端源码、配置和 SQLite 数据目录
├─ frontend/             前端源码和 dist 构建产物
├─ deploy/               宝塔/Nginx 示例配置
├─ docs/                 架构文档
├─ logs/                 运行日志目录
├─ scripts/              初始化和启动脚本
├─ 一键初始化.bat         Windows 初始化
├─ 一键初始化.sh          Linux 初始化
├─ 一键运行.bat           Windows 启动
├─ 一键运行.sh            Linux 启动
└─ README.md
```

## 二、运行要求

服务器需要安装：

- Go 1.22 或更高版本
- Node.js 20 或更高版本
- npm
- Nginx，宝塔面板自带即可

查看版本：

```bash
go version
node -v
npm -v
nginx -v
```

## 三、端口说明

BitAPI 默认使用：

```text
8091
```

本地访问：

```text
http://127.0.0.1:8091
```

线上推荐方式：

```text
Nginx 直接返回 frontend/dist 静态文件
Nginx 只把 /api、/v1、/responses、/uploads、/health 转发给 127.0.0.1:8091
```

不要再使用 `5181`，不要再启动 `vite preview`。

## 四、Linux 首次初始化

进入发布目录：

```bash
cd /www/wwwroot/demo
```

给脚本执行权限：

```bash
chmod +x 一键初始化.sh 一键运行.sh scripts/init.sh scripts/run.sh
```

运行初始化：

```bash
bash 一键初始化.sh
```

脚本会用中文提示：

```text
请输入站点/产品名称
请输入管理员账号邮箱
请输入管理员名称
请输入管理员密码，输入过程会直接显示，请认真核对
请输入服务监听地址
```

密码会直接显示，这是为了避免输错一位还看不见。请在安全的终端环境里输入。

初始化会清空并重建：

```text
backend/data/bitapi.db
backend/data/bitapi.db-shm
backend/data/bitapi.db-wal
```

初始化完成后会生成：

```text
backend/.env.local
```

## 五、Windows 首次初始化

双击：

```text
一键初始化.bat
```

或者在 PowerShell 执行：

```powershell
.\一键初始化.bat
```

同样会要求填写站点名称、管理员账号、名称、密码和监听地址。

## 六、修改产品品牌

管理员登录后台后，进入 `系统设置 -> 产品品牌配置`，可以修改：

- 站点名称，例如 FitAI、AaaAI 或自己的产品名
- 浅色 Logo，显示在首页导航、登录页、控制台和后台侧边栏
- 深色 Logo，显示在首页底边栏等深色区域
- 侧边栏收缩 Logo，建议使用方形图标
- 版权信息

Logo 支持直接上传，文件会保存到 `/uploads/branding/`。Nginx 配置里需要保留 `/uploads/` 转发规则，否则上传后的 Logo 无法访问。

## 七、启动服务

Linux：

```bash
bash 一键运行.sh
```

Windows：

```powershell
.\一键运行.bat
```

启动脚本会自动：

- 停止旧的 `8080`、`5173`、`5181`、`8091` 端口进程
- 安装或刷新前端依赖
- 构建 `frontend/dist`
- 启动 Go 后端
- 等 `/health` 正常后再提示启动完成

日志位置：

```text
logs/backend.log
logs/backend.err.log
```

## 八、手动停止旧进程

如果需要手动杀掉旧服务：

```bash
lsof -ti tcp:8091 | xargs -r kill -9
```

如果服务器没有 `lsof`：

```bash
fuser -k 8091/tcp
```

确认端口是否还在：

```bash
ss -lntp | grep 8091
```

没有输出就表示旧进程已停止。

## 九、宝塔面板推荐配置

推荐使用“静态文件直出 + API 反代”的方式，不要整站反代。这样前端 JS/CSS/图片由 Nginx 直接返回，速度更快。

宝塔网站根目录设置为：

```text
/www/wwwroot/demo/frontend/dist
```

Go 后端仍然监听：

```text
127.0.0.1:8091
```

宝塔配置注意事项：

- 不要启用宝塔的整站反向代理
- 如果之前创建过反向代理，请删除或注释 `proxy/demo.bitit.cn/*.conf`
- 防盗链如果开启，需要放行 `/bitapi-assets/`
- WAF 如果拦截 JS/CSS，先临时关闭测试
- 修改 Nginx 配置后必须重载 Nginx

完整 `server` 配置示例：

```nginx
server
{
    listen 80;
    listen 443 ssl;
    listen 443 quic;
    listen [::]:443 ssl;
    listen [::]:443 quic;
    listen [::]:80;
    http2 on;
    http3 on;

    server_name demo.bitit.cn;
    index index.html index.htm default.html;
    root /www/wwwroot/demo/frontend/dist;

    include /www/server/panel/vhost/nginx/well-known/demo.bitit.cn.conf;
    include /www/server/panel/vhost/nginx/extension/demo.bitit.cn/*.conf;

    ssl_certificate    /www/server/panel/vhost/cert/demo.bitit.cn/fullchain.pem;
    ssl_certificate_key    /www/server/panel/vhost/cert/demo.bitit.cn/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers EECDH+CHACHA20:EECDH+CHACHA20-draft:EECDH+AES128:RSA+AES128:EECDH+AES256:RSA+AES256:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_tickets on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header Alt-Svc 'quic=":443"; h3=":443"; h3-29=":443"; h3-27=":443"' always;

    quic_retry on;
    quic_gso on;
    ssl_early_data on;
    error_page 497 https://$host$request_uri;

    set $isRedcert 1;
    if ($server_port != 443) {
        set $isRedcert 2;
    }
    if ($uri ~ /\.well-known/) {
        set $isRedcert 1;
    }
    if ($isRedcert != 1) {
        rewrite ^(/.*)$ https://$host$1 permanent;
    }

    # 重要：不要再引用宝塔自动生成的整站反向代理规则。
    # include /www/server/panel/vhost/nginx/proxy/demo.bitit.cn/*.conf;

    location ^~ /bitapi-assets/ {
        try_files $uri =404;
        expires 30d;
        add_header Cache-Control "public, max-age=2592000, immutable" always;
    }

    location = /favicon.png {
        try_files /favicon.png =404;
        expires 7d;
        add_header Cache-Control "public, max-age=604800" always;
    }

    location ^~ /api/ {
        proxy_pass http://127.0.0.1:8091;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    location ^~ /v1/ {
        proxy_pass http://127.0.0.1:8091;
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location = /responses {
        proxy_pass http://127.0.0.1:8091;
        proxy_http_version 1.1;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 600s;
        proxy_send_timeout 600s;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location ^~ /uploads/ {
        proxy_pass http://127.0.0.1:8091;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location = /health {
        proxy_pass http://127.0.0.1:8091;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
    }

    location / {
        try_files $uri $uri/ /index.html;
        add_header Cache-Control "no-cache" always;
    }

    location ~* (\.user.ini|\.htaccess|\.htpasswd|\.env.*|\.project|\.bashrc|\.bash_profile|\.bash_logout|\.DS_Store|\.gitignore|\.gitattributes|LICENSE|README\.md|CLAUDE\.md|CHANGELOG\.md|CHANGELOG|CONTRIBUTING\.md|TODO\.md|FAQ\.md|composer\.json|composer\.lock|package(-lock)?\.json|yarn\.lock|pnpm-lock\.yaml|\.\w+~|\.swp|\.swo|\.bak(up)?|\.old|\.tmp|\.temp|\.log|\.sql(\.gz)?|docker-compose\.yml|docker\.env|Dockerfile|\.csproj|\.sln|Cargo\.toml|Cargo\.lock|go\.mod|go\.sum|phpunit\.xml|pom\.xml|build\.gradl|pyproject\.toml|requirements\.txt|application(-\w+)?\.(ya?ml|properties))$
    {
        return 404;
    }

    location ~* /(\.git|\.svn|\.bzr|\.vscode|\.claude|\.idea|\.ssh|\.github|\.npm|\.yarn|\.pnpm|\.cache|\.husky|\.turbo|\.next|\.nuxt|node_modules|runtime)/ {
        return 404;
    }

    location ~ \.well-known {
        allow all;
    }

    if ($uri ~ "^/\.well-known/.*\.(php|jsp|py|js|css|lua|ts|go|zip|tar\.gz|rar|7z|sql|bak)$") {
        return 403;
    }

    access_log /www/wwwlogs/demo.bitit.cn.log;
    error_log  /www/wwwlogs/demo.bitit.cn.error.log;
}
```

改完检查 Nginx：

```bash
nginx -t
/etc/init.d/nginx reload
```

## 十、访问地址

前台首页：

```text
https://你的域名/
```

管理后台登录：

```text
https://你的域名/auth/login
```

健康检查：

```text
https://你的域名/health
```

网关地址：

```text
https://你的域名
```

Codex 或 OpenAI 风格客户端推荐：

```text
base_url = https://你的域名
```

支持接口：

```text
GET  /v1/models
POST /v1/chat/completions
POST /v1/responses
POST /responses
```

## 十一、常见问题

### 1. 页面空白，控制台提示 JS/CSS 403

通常是宝塔防盗链、WAF、整站反代或 CORS 配置导致。

处理方式：

- 确认 `/bitapi-assets/` 是 Nginx 静态文件直出
- 注释或删除宝塔整站反代 include
- 防盗链放行 `/bitapi-assets/`
- WAF 临时关闭测试
- 检查 `backend/.env.local` 里的 `BITAPI_CORS_ORIGINS`

推荐包含：

```text
https://你的域名,https://*.你的主域名
```

示例：

```text
https://demo.bitit.cn,https://*.bitit.cn
```

### 2. 初始化后登录不了

检查：

- 管理员邮箱是不是初始化时输入的邮箱
- 密码是不是初始化时输入的密码
- 查看 `logs/init-backend.err.log`
- 确认 `backend/data/bitapi.db` 已生成

### 3. 反向代理很慢

不要整站反代。推荐：

```text
Nginx 直出 frontend/dist
Go 只处理 API
```

确认没有代理到：

```text
127.0.0.1:5181
```

### 4. 如何重新初始化

重新初始化会清空数据库。

Linux：

```bash
bash 一键初始化.sh
```

Windows：

```powershell
.\一键初始化.bat
```

### 5. 如何只重启服务

Linux：

```bash
bash 一键运行.sh
```

Windows：

```powershell
.\一键运行.bat
```

## 十二、发布注意事项

正式发布前，发布目录不应包含：

```text
backend/data/*.db
backend/data/*.db-shm
backend/data/*.db-wal
logs/*.log
backend/.server.pid
frontend/.vite.pid
*.zip
```

本发布目录已经按这个规则清理。部署时请直接上传整个 `BitAPI_Release` 文件夹，不需要压缩包。
