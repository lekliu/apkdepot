# Stage 1: Build the application
FROM golang:1.21-alpine AS builder

WORKDIR /app

# --- 这是新增和修改的部分 ---
# 设置 Go 模块代理为 goproxy.cn，以解决国内网络无法访问官方代理的问题
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod 和 go.sum 文件
COPY go.mod go.sum ./

# 下载依赖，现在会使用 goproxy.cn
RUN go mod download
# --- 修改结束 ---

# 复制源代码
COPY . .

# 构建应用为静态二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /apkdepot .

# Stage 2: Create the final, minimal image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件
COPY --from=builder /apkdepot .

# 复制模板和静态资源
COPY templates ./templates
COPY static ./static

# 为 APK 创建目录，这将作为数据卷挂载点
RUN mkdir -p /app/apks

EXPOSE 8080

CMD ["./apkdepot"]