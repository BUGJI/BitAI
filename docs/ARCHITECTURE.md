# BitAPI 架构说明

## 目录结构

```text
BitAPI/
├── backend/
│   ├── cmd/server/
│   ├── internal/config/
│   ├── internal/db/
│   ├── internal/models/
│   ├── internal/http/
│   ├── internal/services/
│   └── internal/pkg/
├── frontend/
│   ├── src/api/
│   ├── src/app/
│   ├── src/layouts/
│   ├── src/pages/
│   ├── src/stores/
│   ├── src/styles/
│   └── src/utils/
├── docs/
└── deploy/
```

## 后端

- `cmd/server/main.go`：加载配置，打开 SQLite，执行迁移并初始化管理员账号。
- `internal/models`：定义用户、刷新令牌、调用密钥、分组、上游账号、绑定关系、使用日志、计费幂等和系统设置等 GORM 模型。
- `internal/db`：负责 SQLite 连接、PRAGMA 配置、自动迁移和默认数据。
- `internal/http`：包含 Gin 路由、中间件、统一响应和处理器。
- `internal/http/respond`：统一输出中文响应文案，并兼容常见底层错误的中文转换。
- `internal/services/auth`：处理密码校验、JWT 访问令牌和 SQLite 刷新令牌。
- `internal/services/keys`：处理调用密钥生成、哈希、列表和删除。
- `internal/services/gateway`：处理调用密钥鉴权、分组解析、模型映射、上游选择和代理编排。
- `internal/services/adapters`：实现供应商协议适配层，当前优先支持兼容模型接口的聊天补全和模型列表。
- `internal/services/billing`：处理使用日志、幂等扣费、余额扣减和调用密钥额度统计。
- `internal/services/payments`：处理人工充值订单和兑换码兑换。
- `internal/services/ratelimit`：提供单节点进程内请求频率限制。
- `internal/services/monitor`：执行上游账号健康检查。

## 前端

- 使用 Vue 3、TypeScript 和 Vite。
- 使用 ArcoDesignVue 提供布局、菜单、卡片、表格、表单、弹窗、标签、统计和开关。
- 使用 Pinia 保存登录态和用户信息。
- 使用 Vue Router 区分登录注册、用户控制台和管理后台。
- 所有产品界面仅提供中文文案，Arco 组件全局启用中文 locale。
- 页面包括登录、注册、控制台、调用密钥、费用中心、使用明细、管理概览、用户、分组、上游账号、调用日志、充值兑换和系统设置。

## 数据库

SQLite 使用 WAL 模式、外键、忙等待超时、NORMAL 同步模式，并由 GORM AutoMigrate 管理结构。

核心表：

- `users`：账号、角色、状态、余额、令牌版本和 TOTP 标记。
- `refresh_tokens`：刷新令牌哈希、令牌族、令牌版本、过期时间和吊销状态。
- `api_keys`：密钥哈希、可见前缀、用户、分组、额度、滚动用量窗口和过期时间。
- `groups`：供应商平台、计费模式、模型路由、模型映射、模型列表、限制和功能开关。
- `upstream_accounts`：供应商凭据、基础地址、调度状态、优先级和权重。凭据通过 AES-GCM 加密保存，管理接口仅返回掩码。
- `group_accounts`：分组到上游账号的绑定关系，包含权重、优先级和启用状态。
- `usage_logs`：请求、用户、密钥、上游、模型、令牌、费用、状态码和耗时流水。
- `billing_dedups`：以请求编号为键的计费幂等表。
- `settings`：系统设置和公开设置。
- `payment_orders`：人工充值订单。
- `redeem_codes`：一次性或多次使用的余额兑换码。
- `redeem_code_usages`：用户兑换记录。

## 接口

公开接口：

- `GET /health`
- `GET /api/v1/public/settings`

鉴权接口：

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/me`

用户接口：

- `GET /api/v1/user/api-keys`
- `POST /api/v1/user/api-keys`
- `DELETE /api/v1/user/api-keys/:id`
- `GET /api/v1/user/usage`
- `GET /api/v1/user/orders`
- `POST /api/v1/user/orders`
- `POST /api/v1/user/redeem`

管理接口：

- `GET /api/v1/admin/stats`
- `GET /api/v1/admin/users`
- `PATCH /api/v1/admin/users/:id`
- `POST /api/v1/admin/users/:id/recharge`
- `GET /api/v1/admin/groups`
- `POST /api/v1/admin/groups`
- `PATCH /api/v1/admin/groups/:id`
- `DELETE /api/v1/admin/groups/:id`
- `GET /api/v1/admin/upstream-accounts`
- `POST /api/v1/admin/upstream-accounts`
- `PATCH /api/v1/admin/upstream-accounts/:id`
- `DELETE /api/v1/admin/upstream-accounts/:id`
- `POST /api/v1/admin/upstream-accounts/:id/check`
- `GET /api/v1/admin/group-accounts`
- `POST /api/v1/admin/group-accounts`
- `PATCH /api/v1/admin/group-accounts/:id`
- `DELETE /api/v1/admin/group-accounts/:id`
- `GET /api/v1/admin/usage`
- `GET /api/v1/admin/settings`
- `POST /api/v1/admin/settings`
- `GET /api/v1/admin/orders`
- `POST /api/v1/admin/orders/:id/mark-paid`
- `GET /api/v1/admin/redeem-codes`
- `POST /api/v1/admin/redeem-codes`

网关接口：

- `GET /v1/models`
- `POST /v1/chat/completions`

## 权限

- 管理接口要求 JWT 登录，并且角色为 `owner`、`admin` 或 `operator`。
- 用户接口要求有效 JWT，并且只操作当前登录用户的数据。
- 网关接口要求在兼容模型接口的 `Authorization: Bearer ...` 请求头中传入 BitAPI 调用密钥。
- 调用密钥只保存 SHA-256 哈希和可见前缀。
- 上游凭据使用 `BITAPI_ENCRYPTION_KEY` 加密，生产环境必须提供稳定的非默认密钥。
- 网关请求会检查用户和分组的每分钟请求数限制，当前实现适合单节点部署。

## 模型适配层

当前适配层优先支持兼容模型接口：

1. 解析请求模型和流式标记。
2. 按分组的 `model_mapping_json` 重写模型名称。
3. 转发非流式响应，并解析 `usage`。
4. 以 `text/event-stream` 转发流式响应。
5. 从分组的 `model_list_json` 暴露模型列表。

适配层隔离在 `internal/services/adapters` 中，后续可加入 Anthropic、Gemini、Bedrock 或自定义供应商，而不与 Gin 或 GORM 强耦合。

## 计费

计费在 SQLite 事务中完成：

1. 向 `billing_dedups` 写入请求编号，保证幂等。
2. 锁定并读取用户行。
3. 根据令牌用量估算费用。
4. 扣减 `users.balance_micros`。
5. 增加调用密钥额度和滚动窗口用量。
6. 写入 `usage_logs` 记录。

流式响应会在流结束后写入流水。若上游未返回用量，BitAPI 会记录 0 令牌，并对成功调用写入一个很小的兜底估算费用。

人工充值订单由用户创建，管理员标记为已支付。兑换码由管理员生成，以哈希形式保存，并在兑换时通过事务为用户入账。
