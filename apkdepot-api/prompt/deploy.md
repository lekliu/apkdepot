### 流程概览

1.  **构建 (Build)**: 在 Windows 上使用 Docker 构建镜像。
2.  **保存/推送 (Save/Push)**: 将镜像导出为文件（适合内网）或推送到镜像仓库（Docker Hub/阿里云，适合公网）。
3.  **传输 (Transfer)**: 将文件或配置传输到 Ubuntu 服务器。
4.  **加载/拉取 (Load/Pull)**: 在 Ubuntu 上恢复镜像。
5.  **运行 (Run)**: 使用 `docker-compose up` 启动服务。

---

### 方案一：离线传输（推荐：无需公网带宽，最快）

这种方法直接把镜像打包成 `.tar` 文件，通过 SCP 拷贝到服务器。

#### 步骤 1：在 Windows 上构建镜像

确保你的 `apkdepot-api` 和 `apkdepot-ui` 目录下都有 `Dockerfile`。

```powershell
# 1. 构建后端镜像
docker build -t apkdepot-api:latest ./apkdepot-api

# 2. 构建前端镜像
docker build -t apkdepot-ui:latest ./apkdepot-ui
```

#### 步骤 2：导出镜像为文件

将两个镜像打包成一个 tar 包。

```powershell
# 导出镜像 (PowerShell 中可能需要稍作调整，建议用 Git Bash 或 WSL)
docker save -o apkdepot-images.tar apkdepot-api:latest apkdepot-ui:latest
```

#### 步骤 3：上传到 Ubuntu 服务器

使用 SCP 命令（Windows 10+ 自带 `scp`）将 tar 包和 `docker-compose.yml` 上传。

```powershell
# 假设服务器 IP: 192.168.1.100，用户: user
scp apkdepot-images.tar user@192.168.1.100:~/apkdepot/
scp docker-compose.yml user@192.168.1.100:~/apkdepot/
# 别忘了上传 nginx.conf (如果它不在 docker-compose.yml 的同级目录)
scp apkdepot-ui/nginx.conf user@192.168.1.100:~/apkdepot/
```

#### 步骤 4：在 Ubuntu 上加载并运行

登录到服务器，加载镜像并启动。

```bash
ssh user@192.168.1.100
cd ~/apkdepot

# 1. 加载镜像
docker load -i apkdepot-images.tar

# 2. 修改 docker-compose.yml (关键!)
# 因为我们已经有了镜像，不需要再 build 了。
# 你需要注释掉 build: ... 行，直接使用 image: ...
```

**修改后的 `docker-compose.yml` (服务器版)**:

```yaml
version: '3'
services:
  api:
    image: apkdepot-api:latest  # <-- 改这里，直接用镜像名
    # build: ./apkdepot-api    <-- 注释掉
    container_name: apkdepot-api
    restart: always
    volumes:
      - ./apks:/app/apks
    environment:
      - PORT=8080
      - JWT_SECRET=YourSecretKey
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=password

  ui:
    image: apkdepot-ui:latest  # <-- 改这里
    # build: ./apkdepot-ui    <-- 注释掉
    container_name: apkdepot-ui
    restart: always
    ports:
      - "80:80" # 生产环境通常用 80
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf
    depends_on:
      - api
```

**启动服务**:

```bash
# 创建挂载目录
mkdir apks

# 启动
docker-compose up -d
```

---

### 方案二：使用镜像仓库（标准做法）

如果你有 Docker Hub 账号或阿里云镜像仓库，这更规范。

#### 步骤 1：登录并推送 (Windows)

```powershell
# 1. 登录
docker login

# 2. 标记镜像 (Tag)
# 格式: docker tag 本地名 你的用户名/仓库名:标签
docker tag apkdepot-api:latest youruser/apkdepot-api:v1
docker tag apkdepot-ui:latest youruser/apkdepot-ui:v1

# 3. 推送
docker push youruser/apkdepot-api:v1
docker push youruser/apkdepot-ui:v1
```

#### 步骤 2：服务器拉取运行 (Ubuntu)

**`docker-compose.yml`**:

```yaml
services:
  api:
    image: youruser/apkdepot-api:v1 # 指向远程仓库
    # ...
  ui:
    image: youruser/apkdepot-ui:v1
    # ...
```

**运行**:
```bash
docker-compose pull
docker-compose up -d
```

### 总结

*   **离线传输 (方案一)**: 最快，不需要配置仓库，适合私有部署或单次部署。
*   **镜像仓库 (方案二)**: 适合频繁更新，服务器能联网，版本管理更清晰。