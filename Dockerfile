# Stage 1: Build the application
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application as a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /apkdepot .

# Stage 2: Create the final, minimal image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /apkdepot .

# Copy templates and static assets
COPY templates ./templates
COPY static ./static

