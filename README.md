# PinAI - 轻量级大语言模型路由网关

PinAI 是一个基于 Go 语言开发的轻量级大语言模型路由网关，专为简化访问各种大语言模型（LLM）而设计。目前项目支持 OpenAI 和 Anthropic 兼容的 API 格式，可以轻松集成到您的 AI 应用中。

> [!IMPORTANT]
>
> 本项目仅供个人学习使用，不保证稳定性，且不提供任何技术支持。
>
> 根据[《生成式人工智能服务管理暂行办法》](https://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)的要求，请勿对中国地区公众提供一切未经备案的生成式人工智能服务。

## 🌟 特性

- **轻量级架构**：基于 Go 语言和 Fiber 框架构建，性能优异，资源占用少
- **多平台兼容**：完全兼容 OpenAI 和 Anthropic API 格式，可直接替换现有调用
- **多模型支持**：支持多种大语言模型的统一访问和管理
- **模型映射**：支持自定义模型名称映射，统一不同平台的模型名称
- **流式响应**：完整支持流式响应，提供实时交互体验
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

| 命令行参数                | 环境变量                 | 说明                                                           | 默认值    |
| ------------------------- | ------------------------ | -------------------------------------------------------------- | --------- |
| `-port`                   | `PORT`                   | 监听端口                                                       | `:3000`   |
| `-prod`                   | `PROD`                   | 在生产环境中启用 prefork 模式                                  | `false`   |
| `-enable-web`             | `ENABLE_WEB`             | 启用前端支持                                                   | `false`   |
| `-web-dir`                | `WEB_DIR`                | 前端文件目录                                                   | `web`     |
| `-enable-frontend-update` | `ENABLE_FRONTEND_UPDATE` | 启用前端更新检查                                               | `true`    |
| `-github-proxy`           | `GITHUB_PROXY`           | GitHub 代理地址，用于加速 GitHub 访问                          |           |
| `-db-type`                | `DB_TYPE`                | 数据库类型 (sqlite, mysql, postgres)                           | `sqlite`  |
| `-db-host`                | `DB_HOST`                | 数据库主机地址                                                 |           |
| `-db-port`                | `DB_PORT`                | 数据库端口                                                     |           |
| `-db-user`                | `DB_USER`                | 数据库用户名                                                   |           |
| `-db-pass`                | `DB_PASS`                | 数据库密码                                                     |           |
| `-db-name`                | `DB_NAME`                | 数据库名称                                                     |           |
| `-db-ssl-mode`            | `DB_SSL_MODE`            | PostgreSQL SSL 模式 (disable, require, verify-ca, verify-full) | `disable` |
| `-db-tls-config`          | `DB_TLS_CONFIG`          | MySQL TLS 配置 (true, false, skip-verify, preferred)           | `false`   |
| `-api-token`              | `API_TOKEN`              | API Token，用于业务接口身份验证                                |           |
| `-admin-token`            | `ADMIN_TOKEN`            | 管理 API Token，用于管理接口身份验证（可选）                   |           |
| `-model-mapping`          | `MODEL_MAPPING`          | 模型映射规则，格式：`key1:value1,key2:value2`                  |           |

> [!NOTE]
>
> - 命令行参数优先级高于环境变量。
> - 如果只设置了 `API_TOKEN` 而没有设置 `ADMIN_TOKEN`，则管理接口和业务接口将使用相同的令牌，程序启动时会输出警告。
> - 业务接口指 `/openai/v1/*` 路径下的接口，管理接口指 `/api/*` 路径下的接口。

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

## 📚 API 接口

PinAI 提供以下平台兼容的 API 接口：

### OpenAI 兼容接口

基础路径：`/openai/v1`

- `GET /openai/v1/models` - 获取可用模型列表
- `POST /openai/v1/chat/completions` - 聊天补全接口（支持流式和非流式）

**认证方式**：使用 `Authorization: Bearer <API_TOKEN>` 头进行身份验证

**使用示例**：

```bash
curl https://your-domain.com/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Anthropic 兼容接口

基础路径：`/anthropic/v1`

- `GET /anthropic/v1/models` - 获取可用模型列表
- `POST /anthropic/v1/messages` - 消息补全接口（支持流式和非流式）

**认证方式**：使用 `x-api-key: <API_TOKEN>` 头进行身份验证

**使用示例**：

```bash
curl https://your-domain.com/anthropic/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: YOUR_API_TOKEN" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-opus-20240229",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

> [!NOTE]
>
> - OpenAI 和 Anthropic 接口使用相同的 API Token（通过 `API_TOKEN` 环境变量或 `-api-token` 参数配置）
> - 两种接口格式的请求会被统一转换为内部格式处理，然后转发到相应的 AI 服务提供商
> - 模型映射功能对两种接口格式均有效

## 🏗️ 开发指南

### 技术栈

- [Go 1.23](https://golang.org/)
- [Fiber v2](https://gofiber.io/) - 高性能 Web 框架
- [GORM](https://gorm.io/) - ORM 库
- [slog](https://pkg.go.dev/log/slog) - 结构化日志

### 项目结构

```text
.
├── app.go              # 应用入口
├── router/             # 路由配置
├── handlers/           # 请求处理器
├── services/           # 业务逻辑层
├── database/           # 数据库相关
└── cmd/                # 命令行工具
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
