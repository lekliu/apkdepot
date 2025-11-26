# Stage 1: Build Vue App
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./

# 1. 增加 --unsafe-perm 确保权限正确（虽然在新版npm中默认已修复，但加个保险）
RUN npm install

COPY . .

# 2. [关键修复] 显式赋予 .bin 目录下的文件执行权限
RUN chmod +x node_modules/.bin/*

RUN npm run build

# Stage 2: Serve with Nginx
FROM nginx:1.25-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]