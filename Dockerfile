# 使用Alpine Linux作为基础镜像
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制Go模块文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建Go应用
RUN go build -o main cmd/main.go

# 使用轻量级的Alpine Linux作为运行时镜像
FROM alpine:3.15

# 安装必要的依赖项
RUN apk add --no-cache ca-certificates

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/main .

# 暴露端口（如果需要）
EXPOSE 8080

# 运行应用
CMD ["./main"]
