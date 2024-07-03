# 使用官方Go镜像作为构建环境
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 将Go模块依赖项描述文件复制到工作目录
COPY go.mod ./
COPY go.sum ./

# 下载Go模块依赖项
RUN go mod download

# 将项目源代码复制到工作目录
COPY . .

# 构建应用程序：编译Go源代码生成二进制可执行文件
# RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mserver .
RUN go build -o mserver

# 使用官方Alpine Linux镜像作为运行环境，由于它体积较小
FROM alpine:latest

RUN apk add --no-cache bash
# 安装tzdata包，添加时区数据
RUN apk add --no-cache tzdata

# 从构建环境中复制编译后的应用程序到当前工作目录
COPY --from=builder /app/mserver .

# 设置环境变量以配置时区
ENV TZ=Asia/Shanghai

# 暴露端口，该端口需要与你的Gin应用程序监听的端口一致
EXPOSE 9096

ENV GIN_MODE=release

# 运行编译后的二进制可执行文件
CMD ["./mserver"]
