# 使用Go 1.23作为构建环境
FROM golang:1.23-alpine AS builder

# 安装git和其他构建依赖
RUN apk add --no-cache git build-base

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN go build -ldflags="-s -w" -o pinai .

# 使用alpine作为运行环境
FROM alpine:latest

# 安装ca-certificates以支持HTTPS请求
RUN apk --no-cache add ca-certificates

# 创建工作目录
WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/pinai .

# 暴露端口
EXPOSE 3000

# 运行应用
CMD ["./pinai"]