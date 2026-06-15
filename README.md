# BitAPI

BitAPI 是一个中文 AI API Gateway，后端使用 Go、Gin、GORM、SQLite，前端使用 Vue 3、TypeScript、Arco Design Vue。

本版本采用“手动初始化”模式：发布包不内置数据库、不内置默认管理员密码。首次使用前必须先运行初始化脚本，由部署者自行填写管理员邮箱、管理员名称和管理员密码。

## 目录说明

```text
BitAPI
├── backend/              后端源码、配置示例、SQLite 数据目录
├── frontend/             前端源码和已构建的 dist 产物
├── docs/                 架构文档
├── logs/                 运行日志目录
├── scripts/              PowerShell 运维脚本
├── 一键初始化.bat         首次初始化或重置数据库
└── 一键运行.bat           停止旧服务并重新启动前后端
```

## 环境要求

- Windows
- Go 1.22 或更高版本
- Node.js 20 或更高版本
- npm

运行前请确认命令行能直接执行：

```powershell
go version
node -v
npm -v
```

## 首次使用

### 1. 运行初始化脚本

双击项目根目录下的：

```text
一键初始化.bat
```

初始化脚本会执行以下操作：

- 停止本机 `8080` 和 `5173` 端口上的旧 BitAPI 服务
- 删除 `backend/data/bitapi.db`、`bitapi.db-wal`、`bitapi.db-shm`
- 交互填写管理员邮箱、管理员名称、管理员密码、后端监听地址
- 生成 `backend/.env.local`
- 创建 SQLite 数据库、数据表、默认分组和管理员账号

注意：初始化会清空数据库，请只在首次部署或确认要重置数据时运行。

### 2. 运行服务

双击项目根目录下的：

```text
一键运行.bat
```

脚本会先停止旧服务，然后重新启动：

- 前端：`http://localhost:5173`
- 后端：`http://localhost:8080`

日志位置：

- 后端日志：`logs/backend.log`
- 后端错误日志：`logs/backend.err.log`
- 前端日志：`logs/frontend.log`
- 前端错误日志：`logs/frontend.err.log`

### 3. 登录后台

打开：

```text
http://localhost:5173/auth/login
```

使用初始化时填写的管理员邮箱和密码登录。

## 初始化配置

初始化完成后会生成：

```text
backend/.env.local
```

主要配置项包括：

```text
BITAPI_HTTP_ADDR=:8080
BITAPI_DATABASE_DSN=file:data/bitapi.db?_foreign_keys=on&_busy_timeout=5000
BITAPI_BOOTSTRAP_EMAIL=初始化填写的管理员邮箱
BITAPI_BOOTSTRAP_PASSWORD=初始化填写的管理员密码
BITAPI_BOOTSTRAP_NAME=初始化填写的管理员名称
BITAPI_JWT_SECRET=自动生成
BITAPI_ENCRYPTION_KEY=自动生成
```

`BITAPI_JWT_SECRET` 用于签发登录令牌，`BITAPI_ENCRYPTION_KEY` 用于加密保存上游账号凭据。已经投入使用后不要随意更换这两个值，否则会影响登录状态和已保存的上游密钥解密。

## 后台配置流程

首次登录后，建议按以下顺序配置：

1. 进入“系统设置”，配置 SMTP。注册邮箱验证码依赖 SMTP。
2. 进入“上游账号”，添加 OpenAI 兼容上游账号。
3. 进入“分组绑定”，将上游账号绑定到默认分组。
4. 进入“用户管理”，按需给用户充值或调整状态。
5. 进入“充值兑换”，创建兑换码或处理充值订单。
6. 进入“调用密钥”，创建用户调用密钥。
7. 使用调用密钥请求网关接口。

## 网关接口

默认网关地址：

```text
http://localhost:8080
```

支持的 OpenAI 兼容接口：

```text
GET  /v1/models
POST /v1/chat/completions
POST /v1/responses
POST /responses
```

请求示例：

```powershell
curl.exe http://localhost:8080/v1/responses `
  -H "Authorization: Bearer 你的调用密钥" `
  -H "Content-Type: application/json" `
  -d "{\"model\":\"gpt-5.5\",\"input\":\"你好\",\"stream\":false}"
```

Codex 对接建议使用：

```text
base_url = http://localhost:8080
```

实际请求入口为：

```text
http://localhost:8080/responses
http://localhost:8080/v1/responses
```

## 常用维护

### 重启服务

双击：

```text
一键运行.bat
```

该脚本会自动关闭旧的前后端进程并重新启动。

### 重置数据库

双击：

```text
一键初始化.bat
```

重新初始化会删除当前 SQLite 数据库，包括用户、密钥、上游账号、订单、兑换码和调用日志。

### 手动运行后端

```powershell
cd backend
Get-Content .env.local | ForEach-Object {
  if ($_ -match "^\s*([^#][^=]+)=(.*)$") {
    [Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2], "Process")
  }
}
go run .\cmd\server
```

### 手动运行前端

```powershell
cd frontend
npm.cmd install
npm.cmd run dev -- --host 0.0.0.0
```

## 发布包说明

`BitAPI_Release` 是干净发布版：

- 不包含历史 SQLite 数据库
- 不包含 `node_modules`
- 保留 `frontend/dist` 构建产物
- 保留一键初始化和一键运行脚本

首次部署发布包时，必须先运行 `一键初始化.bat`，再运行 `一键运行.bat`。
