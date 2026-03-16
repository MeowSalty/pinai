# PinAI - 轻量级大语言模型路由网关

PinAI 是一个基于 Go 语言开发的轻量级大语言模型路由网关，专为简化访问各种大语言模型（LLM）而设计。目前项目支持 OpenAI、Anthropic 和 Gemini 兼容的 API 格式，可以轻松集成到您的 AI 应用中。

> [!IMPORTANT]
>
> 本项目仅供个人学习使用，不保证稳定性，且不提供任何技术支持。
>
> 根据[《生成式人工智能服务管理暂行办法》](https://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)的要求，请勿对中国地区公众提供一切未经备案的生成式人工智能服务。

## 🖥️ 前端项目

本项目为后端服务，配套前端项目请访问：

- **前端仓库**：[pinai-frontend](https://github.com/MeowSalty/pinai-frontend)
- **在线演示**：[https://pinai-frontend.pages.dev](https://pinai-frontend.pages.dev)

## 🌟 特性

- **轻量级架构**：基于 Go 语言和 Gin 框架构建，性能优异，资源占用少
- **多平台兼容**：完全兼容 OpenAI、Anthropic 和 Gemini API 格式，可直接替换现有调用
- **多模型支持**：支持多种大语言模型的统一访问和管理
- **模型映射**：支持自定义模型名称映射，统一不同平台的模型名称
- **流式响应**：完整支持流式响应，提供实时交互体验
- **健康状态管理**：支持平台、密钥、模型的健康状态监控和管理
- **请求统计与仪表盘**：提供概览、实时统计、调用排行、用量排行、请求日志与仪表盘接口
- **原生透传能力**：支持 [`/multi/native/*`](README.md) 原生接口，保留上游响应格式
- **请求头透传**：支持将客户端 HTTP 请求头透传到上游请求
- **可控代理能力**：可按配置启用 [`/api/proxy`](README.md) 管理代理端点
- **多数据库支持**：支持 SQLite（默认）、MySQL 和 PostgreSQL 数据库
- **数据库 TLS 支持**：支持 MySQL 和 PostgreSQL 数据库的 TLS 加密连接
- **易于部署**：提供 Docker 镜像，支持容器化部署

## 🚀 快速开始

### 使用 Docker（推荐）

```bash
# 拉取并运行最新版本（自行设置 token）
docker run -d \
  -p 3000:3000 \
  -e ENABLE_WEB=true \
  -e API_TOKEN=<业务token> \
  -e ADMIN_TOKEN=<管理token> \
  ghcr.io/meowsalty/pinai:latest
```

如果需要持久化数据，请将 PinAI 的数据目录 `/app/data` 映射到宿主机的目录

### 本地运行

```bash
# 克隆项目
git clone https://github.com/MeowSalty/pinai.git
cd pinai

# 运行项目
go run app.go -api-token=<业务token> -admin-token=<管理token> -enable-web=true
```

服务默认在 `http://localhost:3000` 上运行。

## 🛠️ 配置选项

PinAI 支持多种配置选项，可以通过命令行参数或环境变量进行设置：

### 配置参数说明

| 命令行参数                | 环境变量                 | 说明                                                           | 默认值     |
| ------------------------- | ------------------------ | -------------------------------------------------------------- | ---------- |
| `-port`                   | `PORT`                   | 监听端口                                                       | `:3000`    |
| `-prod`                   | `PROD`                   | 在生产环境中启用 prefork 模式                                  | `false`    |
| `-enable-web`             | `ENABLE_WEB`             | 启用前端支持                                                   | `false`    |
| `-web-dir`                | `WEB_DIR`                | 前端文件目录                                                   | `web`      |
| `-enable-frontend-update` | `ENABLE_FRONTEND_UPDATE` | 启用前端更新检查                                               | `true`     |
| `-passthrough-headers`    | `PASSTHROUGH_HEADERS`    | 是否透传 HTTP 请求头到上游 portal 请求                         | `true`     |
| `-github-proxy`           | `GITHUB_PROXY`           | GitHub 代理地址，用于加速 GitHub 访问                          |            |
| `-proxy-enabled`          | `PROXY_ENABLED`          | 启用管理代理端点 `/api/proxy`（仍需配置 `ADMIN_TOKEN`）        | `false`    |
| `-db-type`                | `DB_TYPE`                | 数据库类型 (sqlite, mysql, postgres)                           | `sqlite`   |
| `-db-host`                | `DB_HOST`                | 数据库主机地址                                                 |            |
| `-db-port`                | `DB_PORT`                | 数据库端口                                                     |            |
| `-db-user`                | `DB_USER`                | 数据库用户名                                                   |            |
| `-db-pass`                | `DB_PASS`                | 数据库密码                                                     |            |
| `-db-name`                | `DB_NAME`                | 数据库名称                                                     |            |
| `-db-ssl-mode`            | `DB_SSL_MODE`            | PostgreSQL SSL 模式 (disable, require, verify-ca, verify-full) | `disable`  |
| `-db-tls-config`          | `DB_TLS_CONFIG`          | MySQL TLS 配置 (true, false, skip-verify, preferred)           | `false`    |
| `-api-token`              | `API_TOKEN`              | API Token，用于业务接口身份验证                                |            |
| `-admin-token`            | `ADMIN_TOKEN`            | 管理 API Token，用于管理接口身份验证（可选）                   |            |
| `-model-mapping`          | `MODEL_MAPPING`          | 模型映射规则，格式：`key1:value1,key2:value2`                  |            |
| `-user-agent`             | `USER_AGENT`             | User-Agent 配置（见下方说明）                                  | 空（透传） |
| `-log-level`              | `LOG_LEVEL`              | 日志输出等级 (DEBUG, INFO, WARN, ERROR)                        | `INFO`     |

> [!NOTE]
>
> - 命令行参数优先级高于环境变量。
> - 如果只设置了 `API_TOKEN` 而没有设置 `ADMIN_TOKEN`，则管理接口和业务接口将使用相同的令牌，程序启动时会输出警告。
> - 业务接口包括 `/openai/v1/*`、`/anthropic/v1/*` 和 `/multi/*`，管理接口指 `/api/*` 路径下的接口。

#### 请求头透传配置说明

通过 `-passthrough-headers` 或 `PASSTHROUGH_HEADERS` 可控制是否将客户端请求头透传到上游请求，默认开启。

- `true`：透传客户端请求头，便于保留调用方上下文
- `false`：不透传客户端请求头，仅发送服务端构造的上游请求头

> [!NOTE]
>
> - 该配置主要作用于 [`/multi`](README.md) 统一网关转发流程。
> - 建议在需要精细控制上游请求头时关闭此选项。

#### 代理功能配置说明

通过 `-proxy-enabled` 或 `PROXY_ENABLED` 可以显式启用管理代理接口。

- 仅当 `PROXY_ENABLED=true` 且已配置 `ADMIN_TOKEN` 时，才会注册 `POST /api/proxy`
- 适合用于后端代发管理类请求或临时调试上游接口

> [!NOTE]
>
> - 代理接口属于管理接口，不使用 `API_TOKEN`，而是使用 `ADMIN_TOKEN` 认证。
> - 默认不启用，避免在未评估风险时暴露通用代理能力。

#### 数据库 TLS 配置说明

- PostgreSQL 使用 `-db-ssl-mode` 参数：
  - `disable`: 禁用 SSL
  - `require`: 要求 SSL（不验证证书）
  - `verify-ca`: 验证证书颁发机构
  - `verify-full`: 完全验证证书（主机名和颁发机构）

- MySQL 使用 `-db-tls-config` 参数：
  - `true`: 启用 SSL
  - `false`: 禁用 SSL
  - `skip-verify`: 启用 SSL 但跳过证书验证
  - `preferred`: 优先使用 SSL，如果服务器不支持则回退到非加密连接

#### 模型映射配置说明

模型映射功能允许您将客户端请求的模型名称映射到实际使用的模型名称。这在以下场景中非常有用：

- 将 AI 应用中固定访问的模型名称映射到拼好 AI 中的模型名称

配置格式：`原始模型名:目标模型名,原始模型名2:目标模型名2`

示例：

如果需要在 Claude Code 中使用 DeepSeek 模型，您可以将模型映射规则设置为：`claude-sonnet-4-20250514:deepseek-v3`

```bash
# 命令行方式
./pinai -model-mapping="claude-sonnet-4-20250514:deepseek-v3"

# 环境变量方式
export MODEL_MAPPING="claude-sonnet-4-20250514:deepseek-v3"
./pinai

# Docker 方式
docker run -d \
  -p 3000:3000 \
  -e MODEL_MAPPING="claude-sonnet-4-20250514:deepseek-v3" \
  ghcr.io/meowsalty/pinai:latest
```

> [!NOTE]
>
> - 如果不配置模型映射规则，将不会进行任何模型名称转换
> - 映射规则区分大小写
> - 只有在映射规则中定义的模型才会被转换，未定义的模型将保持原名称

#### GitHub 代理配置说明

如果您在访问 GitHub 时遇到网络问题，可以使用 GitHub 代理来加速前端文件的下载和更新。配置方法：

```bash
# 命令行方式
./pinai -enable-web=true -github-proxy=[GitHub 加速地址]

# 环境变量方式
export GITHUB_PROXY=[GitHub 加速地址]
./pinai -enable-web=true

# Docker 方式
docker run -d \
  -p 3000:3000 \
  -e ENABLE_WEB=true \
  -e GITHUB_PROXY=[GitHub 加速地址] \
  ghcr.io/meowsalty/pinai:latest
```

代理工作原理：

- 原始地址：`https://api.github.com/repos/...`
- 使用代理后：`[GitHub 加速地址]/https://api.github.com/repos/...`

> [!NOTE]
>
> - 代理服务仅用于加速 GitHub 访问，不会影响其他功能
> - 请选择可信的代理服务，避免使用不明来源的代理
> - 如果不设置此参数，将直接访问 GitHub

#### User-Agent 配置说明

User-Agent 配置用于控制向上游 AI 服务发送请求时的 User-Agent 头部。支持三种模式：

1. **透传模式**（默认）：不设置或设置为空字符串时，将透传客户端请求的 User-Agent
2. **默认模式**：设置为 `"default"` 时，不添加 User-Agent 头部，使用 Go `net/http` 的默认值
3. **自定义模式**：设置为其他任意字符串时，使用该字符串作为 User-Agent

配置示例：

```bash
# 命令行方式 - 透传客户端 User-Agent（默认行为）
./pinai

# 命令行方式 - 使用 fasthttp 默认 User-Agent
./pinai -user-agent="default"

# 命令行方式 - 自定义 User-Agent
./pinai -user-agent="MyCustomAgent/1.0"

# 环境变量方式
export USER_AGENT="MyCustomAgent/1.0"
./pinai

# Docker 方式
docker run -d \
  -p 3000:3000 \
  -e USER_AGENT="MyCustomAgent/1.0" \
  ghcr.io/meowsalty/pinai:latest
```

> [!NOTE]
>
> - User-Agent 配置对 OpenAI、Anthropic 和 Gemini 兼容接口均有效
> - 透传模式可以保留客户端的原始 User-Agent 信息
> - 自定义模式适用于需要统一标识的场景

## 📚 API 接口

PinAI 提供以下平台兼容的 API 接口：

### 代理接口

基础路径：`/api`

- `POST /api/proxy` - 通过后端代理访问任意上游端点

**启用条件**：需要同时满足 `PROXY_ENABLED=true` 与已配置 `ADMIN_TOKEN`

**认证方式**：使用 `Authorization: Bearer <ADMIN_TOKEN>` 头进行身份验证

**使用示例**（获取 OpenAI 模型列表）：

```bash
curl https://your-domain.com/api/proxy \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://api.openai.com/v1/models",
    "method": "GET",
    "headers": {
      "Authorization": "Bearer YOUR_UPSTREAM_TOKEN"
    }
  }'
```

### Provider 管理接口

Provider 管理接口用于管理 LLM 供应商的平台、模型、密钥和端点。

**认证方式**：使用 `Authorization: Bearer <ADMIN_TOKEN>` 头进行身份验证

#### 平台管理

| 方法   | 路径                                | 说明             |
| ------ | ----------------------------------- | ---------------- |
| POST   | `/api/platforms`                    | 创建平台         |
| GET    | `/api/platforms`                    | 获取平台列表     |
| GET    | `/api/platforms/:platformId`        | 获取平台详情     |
| PUT    | `/api/platforms/:platformId`        | 更新平台         |
| DELETE | `/api/platforms/:platformId`        | 删除平台         |
| PATCH  | `/api/platforms/:platformId/health` | 更新平台健康状态 |

#### 模型管理

| 方法   | 路径                                      | 说明             |
| ------ | ----------------------------------------- | ---------------- |
| POST   | `/api/platforms/:platformId/models`       | 添加模型         |
| POST   | `/api/platforms/:platformId/models/batch` | 批量添加模型     |
| GET    | `/api/platforms/:platformId/models`       | 获取平台模型列表 |
| PUT    | `/api/platforms/:platformId/models/batch` | 批量更新模型     |
| DELETE | `/api/platforms/:platformId/models/batch` | 批量删除模型     |
| PUT    | `/api/models/:modelId`                    | 更新模型         |
| DELETE | `/api/models/:modelId`                    | 删除模型         |
| PATCH  | `/api/models/:modelId/health`             | 更新模型健康状态 |

#### 密钥管理

| 方法   | 路径                              | 说明             |
| ------ | --------------------------------- | ---------------- |
| POST   | `/api/platforms/:platformId/keys` | 添加密钥         |
| GET    | `/api/platforms/:platformId/keys` | 获取平台密钥列表 |
| PUT    | `/api/keys/:keyId`                | 更新密钥         |
| DELETE | `/api/keys/:keyId`                | 删除密钥         |
| PATCH  | `/api/keys/:keyId/health`         | 更新密钥健康状态 |

#### 端点管理

| 方法   | 路径                                         | 说明             |
| ------ | -------------------------------------------- | ---------------- |
| POST   | `/api/platforms/:platformId/endpoints`       | 添加端点         |
| POST   | `/api/platforms/:platformId/endpoints/batch` | 批量添加端点     |
| GET    | `/api/platforms/:platformId/endpoints`       | 获取平台端点列表 |
| PUT    | `/api/platforms/:platformId/endpoints/batch` | 批量更新端点     |
| GET    | `/api/endpoints/:endpointId`                 | 获取端点详情     |
| PUT    | `/api/endpoints/:endpointId`                 | 更新端点         |
| DELETE | `/api/endpoints/:endpointId`                 | 删除端点         |

### 统计接口

统计接口用于查看 API 使用情况和请求日志。

**认证方式**：使用 `Authorization: Bearer <ADMIN_TOKEN>` 头进行身份验证

| 方法 | 路径                              | 说明               |
| ---- | --------------------------------- | ------------------ |
| GET  | `/api/stats/dashboard`            | 获取统计仪表盘     |
| GET  | `/api/stats/overview`             | 获取统计概览       |
| GET  | `/api/stats/requests`             | 获取请求日志列表   |
| GET  | `/api/stats/realtime`             | 获取实时统计       |
| GET  | `/api/stats/models/call-rank`     | 获取模型调用排名   |
| GET  | `/api/stats/platforms/call-rank`  | 获取平台调用排名   |
| GET  | `/api/stats/models/usage-rank`    | 获取模型使用量排名 |
| GET  | `/api/stats/platforms/usage-rank` | 获取平台使用量排名 |

### 健康状态接口

健康状态接口用于监控和管理平台、密钥、模型的健康状态。

**认证方式**：使用 `Authorization: Bearer <ADMIN_TOKEN>` 头进行身份验证

| 方法 | 路径                    | 说明                 |
| ---- | ----------------------- | -------------------- |
| GET  | `/api/health/summary`   | 获取健康状态摘要     |
| GET  | `/api/health/issues`    | 获取异常资源列表     |
| GET  | `/api/health/platforms` | 获取平台健康状态列表 |
| GET  | `/api/health/keys`      | 获取密钥健康状态列表 |
| GET  | `/api/health/models`    | 获取模型健康状态列表 |

### Multi 接口

Multi 接口是一个统一的 API 网关，支持 OpenAI、Anthropic 和 Gemini 三种 API 格式。系统根据请求路径、查询参数或请求头自动识别所需格式。

提供两种接口模式：

| 模式     | 路径前缀                                   | 说明                                            |
| -------- | ------------------------------------------ | ----------------------------------------------- |
| 兼容接口 | `/multi/v1`、`/multi/v1beta`               | 自动转换请求/响应格式，适合跨平台调用           |
| 原生接口 | `/multi/native/v1`、`/multi/native/v1beta` | 直接透传请求，不进行格式转换，保留原始 API 响应 |

#### 接口列表

**OpenAI 格式**：

- `GET /multi/v1/models` 或 `GET /multi/native/v1/models` - 获取模型列表
- `POST /multi/v1/chat/completions` 或 `POST /multi/native/v1/chat/completions` - 聊天补全
- `POST /multi/v1/responses` 或 `POST /multi/native/v1/responses` - Responses API

**Anthropic 格式**：

- `GET /multi/v1/models` 或 `GET /multi/native/v1/models` - 获取模型列表
- `POST /multi/v1/messages` 或 `POST /multi/native/v1/messages` - 消息补全

**Gemini 格式**：

- `GET /multi/v1beta/models` 或 `GET /multi/native/v1beta/models` - 获取模型列表
- `POST /multi/v1beta/models/{model}:generateContent` 或 `POST /multi/native/v1beta/models/{model}:generateContent` - 生成内容
- `POST /multi/v1beta/models/{model}:streamGenerateContent` 或 `POST /multi/native/v1beta/models/{model}:streamGenerateContent` - 流式生成

#### 认证方式

| 接口类型  | 认证方式                                            | 说明                              |
| --------- | --------------------------------------------------- | --------------------------------- |
| OpenAI    | `Authorization: Bearer <API_TOKEN>`                 | Bearer Token 认证                 |
| Anthropic | `x-api-key: <API_TOKEN>`                            | 需同时携带 `anthropic-version` 头 |
| Gemini    | `x-goog-api-key: <API_TOKEN>` 或 `?key=<API_TOKEN>` | 请求头或查询参数                  |

#### Provider 识别规则

系统按以下优先级识别请求的 Provider：

1. **路径识别**：根据请求路径自动识别
   - `/chat/completions`、`/responses` → OpenAI
   - `/messages` → Anthropic
   - `/generateContent`、`/streamGenerateContent`、`/v1beta/models` → Gemini

2. **查询参数**：`?provider=openai|anthropic|gemini`

3. **请求头识别**：
   - 同时携带 `x-api-key` 和 `anthropic-version` → Anthropic
   - 携带 `x-goog-api-key` 或 `key` 查询参数 → Gemini
   - 默认 → OpenAI

#### 使用示例

**OpenAI 格式**：

```bash
curl https://your-domain.com/multi/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Anthropic 格式**：

```bash
curl https://your-domain.com/multi/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_API_TOKEN" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-opus-20240229",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Gemini 格式**：

```bash
# 使用请求头认证
curl https://your-domain.com/multi/v1beta/models/gemini-pro:generateContent \
  -H "Content-Type: application/json" \
  -H "x-goog-api-key: YOUR_API_TOKEN" \
  -d '{
    "contents": [{"parts": [{"text": "Hello!"}]}]
  }'

# 使用查询参数认证
curl "https://your-domain.com/multi/v1beta/models/gemini-pro:generateContent?key=YOUR_API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [{"parts": [{"text": "Hello!"}]}]
  }'
```

> [!NOTE]
>
> - 所有接口使用相同的 API Token（通过 `API_TOKEN` 环境变量或 `-api-token` 参数配置）
> - 模型映射功能对所有格式的接口均有效
> - Gemini 接口路径中的 `{model}` 需替换为实际的模型名称
> - 原生接口只需在路径中添加 `/native` 前缀即可，如 `/multi/v1/chat/completions` → `/multi/native/v1/chat/completions`

## 🏗️ 开发指南

### 技术栈

- [Go 1.25.0](https://golang.org/)
- [Gin](https://gin-gonic.com/) - 高性能 Web 框架
- [GORM](https://gorm.io/) - ORM 库
- [slog](https://pkg.go.dev/log/slog) - 结构化日志

### 项目结构

```text
.
├── app.go              # 应用入口
├── router/             # 路由配置
├── handlers/           # 请求处理器
│   ├── health/         # 健康状态接口
│   ├── multi/          # Multi 统一接口与原生透传接口
│   ├── provider/       # Provider 管理接口
│   ├── proxy/          # 管理代理接口
│   └── stats/          # 统计接口
├── services/           # 业务逻辑层
│   ├── health/         # 健康状态服务
│   ├── portal/         # LLM 请求转发服务
│   ├── provider/       # Provider 管理服务
│   └── stats/          # 统计服务
├── database/           # 数据库相关
│   ├── query/          # GORM Gen 生成的查询代码
│   └── types/          # 数据库类型定义
├── config/             # 配置管理
├── logger/             # 日志处理
├── frontend/           # 前端相关
├── server/             # 服务器启动
└── cmd/                # 命令行工具与生成脚本
```

### 构建项目

```bash
# 构建二进制文件
go build -o pinai app.go

# 运行
./pinai
```

## 📄 许可证

本项目采用 GPLv3 许可证，详情请查看 [LICENSE](LICENSE) 文件。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来帮助改进项目。在贡献代码前，请确保：

1. 遵循项目的代码风格
2. 添加适当的测试
3. 更新相关文档

> [!NOTE]
>
> 如果需要添加支持的平台格式，请移步 [Portal](https://github.com/MeowSalty/portal)

## 🙏 鸣谢

感谢所有为项目做出贡献的开发者们！
