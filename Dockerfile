# 构建应用程序二进制文件
FROM golang:1.23-alpine AS build

# 安装 CGO 构建依赖
RUN apk add --no-cache build-base

WORKDIR /go/src/pinai

# 复制所有代码和相关文件以编译
COPY . .

# 下载所有依赖项
RUN go mod download

# 构建应用
RUN go build -ldflags="-s -w" -o pinai .

# 将二进制文件移动到'最终镜像'以减小镜像大小
FROM alpine:latest as release

WORKDIR /app
# 从构建阶段复制二进制文件
COPY --from=build /go/src/pinai/pinai .

# 添加必要的包
RUN apk add --no-cache ca-certificates \
    && chmod +x /app/pinai

# 暴露端口
EXPOSE 3000

CMD ["/app/pinai"]