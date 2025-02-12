# syntax=docker/dockerfile:1

#######################
# 构建阶段 (builder)
#######################
FROM golang:1.23.6-alpine AS builder
WORKDIR /daysign

# 安装 git 等依赖（如果需要）
RUN apk add --no-cache git

# 设置 GOPROXY 为国内镜像（根据需要可修改）
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod 和 go.sum 并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源码
COPY . .

# 设置编译环境变量，禁用 CGO
ENV CGO_ENABLED=0

# 编译目标二进制文件（GOOS 和 GOARCH 会在 buildx 构建时指定）
RUN go build -o app main.go

#######################
# 最终镜像
#######################
FROM alpine:latest
WORKDIR /root/

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /daysign/app .

# 如果你的程序需要监听端口，可在此开放对应端口，例如 EXPOSE 8080
EXPOSE 12048

CMD ["./app"]