# PinAI - 轻量级大语言模型路由网关

PinAI 是一个基于 Go 语言开发的轻量级大语言模型路由网关，专为简化访问各种大语言模型（LLM）而设计。目前项目支持 OpenAI 兼容的 API 格式，可以轻松集成到您的 AI 应用中。

> [!IMPORTANT]
>
> 本项目仅供个人学习使用，不保证稳定性，且不提供任何技术支持。
>
> 根据[《生成式人工智能服务管理暂行办法》](https://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)的要求，请勿对中国地区公众提供一切未经备案的生成式人工智能服务。

## 🌟 特性

- **轻量级架构**：基于 Go 语言和 Fiber 框架构建，性能优异，资源占用少
- **OpenAI 兼容**：完全兼容 OpenAI API 格式，可直接替换现有 OpenAI 调用
- **多模型支持**：支持多种大语言模型的统一访问和管理
- **流式响应**：完整支持流式响应，提供实时交互体验
- **多数据库支持**：支持 SQLite（默认）、MySQL 和 PostgreSQL 数据库
- **易于部署**：提供 Docker 镜像，支持容器化部署

## 🚀 快速开始

### 使用 Docker（推荐）

```bash
# 拉取并运行最新版本
docker run -d -p 3000:3000 ghcr.io/meowsalty/pinai:latest
```

### 本地运行

```bash
# 克隆项目
git clone https://github.com/MeowSalty/pinai.git
cd pinai

# 运行项目
go run app.go
```

服务默认在 `http://localhost:3000` 上运行。

## 🛠️ 配置选项

PinAI 支持多种配置选项，可以通过命令行参数进行设置：

```bash
go run app.go -port=:3000 -db-type=sqlite -prod
```

主要配置参数：

- `-port`：监听端口（默认 :3000）
- `-prod`：在生产环境中启用 prefork 模式
- `-db-type`：数据库类型（sqlite、mysql、postgres）
- `-db-host`：数据库主机地址
- `-db-port`：数据库端口
- `-db-user`：数据库用户名
- `-db-pass`：数据库密码
- `-db-name`：数据库名称

## 📚 API 接口

PinAI 提供以下平台兼容的 API 接口：

### OpenAI 兼容 接口

- `GET /openai/v1/models` - 获取可用模型列表
- `POST /openai/v1/chat/completions` - 聊天补全接口（支持流式和非流式）

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

## 🙏 鸣谢

感谢所有为项目做出贡献的开发者们！
