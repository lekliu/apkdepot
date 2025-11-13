# Stage 1: Build
FROM golang:1.21-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /apkdepot-api .

# Stage 2: Run
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /apkdepot-api .
# APKs目录会通过volume挂载，所以这里不需要创建
EXPOSE 8080
CMD ["./apkdepot-api"]