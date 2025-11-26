# Stage 1: Build
FROM golang:1.21-alpine AS builder
WORKDIR /app

# 使用国内代理加速 (可选)
ENV GOPROXY=https://goproxy.cn,direct

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码 (包括 cmd, internal 等目录)
COPY . .

# 构建指令修改：指向 main.go 所在路径 ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /apkdepot-api ./cmd/server

# Stage 2: Run
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件
COPY --from=builder /apkdepot-api .

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./apkdepot-api"]