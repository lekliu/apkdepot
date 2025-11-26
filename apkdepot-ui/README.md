# ApkDepot 使用与开发指南 (v1.0)

ApkDepot 是一个轻量级的私有化 APK 托管与分发平台，支持版本管理、灰度发布和强制更新控制。

---

## 📖 第一部分：用户使用手册 (User Guide)

### 1. 平台访问
*   **地址**: `http://<服务器IP>:8888`
*   **功能**: 默认展示所有已上传的 APK 列表，支持下载。

### 2. 管理员登录
*   点击右上角的 **"Login"**。
*   输入部署时设置的账号密码（默认: `admin` / `password`，请在 `docker-compose.yml` 中修改）。
*   登录成功后，你将看到 **"Upload"** 入口以及每行 APK 的 **"Edit"** (策略管理) 和 **"Delete"** 按钮。

### 3. 上传新版本
1.  点击顶部导航栏的 **"Upload"**。
2.  选择你的 `.apk` 文件（文件名任意，系统会自动重命名为 `包名_版本号.apk`）。
3.  上传成功后，系统会自动解析包名、版本号、图标等信息，并更新元数据。
  *   **注意**: 如果上传的版本号大于当前记录的最新版，它将自动成为新的 **Latest Version**。默认发布状态为 **暂停 (0% 灰度)**。

### 4. 发布策略管理 (核心功能)
在列表页，找到状态为 **Latest** 的 APK（通常是第一行），点击 **"Edit"** 按钮。

*   **Rollout Rate (灰度比例)**:
  *   拖动滑块或直接输入数字来控制发布范围。
  *   `0%`: 暂停发布，任何人都检测不到此更新。
  *   `10%`: 只有约 10% 的设备能检测到更新（基于设备ID哈希）。
  *   `100%`: 全量发布，所有旧版本设备都会收到提示。
*   **Min Force Version Code (强制基线)**:
  *   输入一个整数（VersionCode）。
  *   所有**低于**此版本号的用户，APP端会弹出**不可关闭**的强制更新窗口。
  *   *示例*: 最新版是 102，你发现 100 版有严重Bug，于是将基线设为 `101`。那么 100 版用户会被强制升级，101 版用户则可选择性升级。

---

## 🛠 第二部分：Android 客户端接入开发指南

要在你的 Android 应用中集成自动更新功能，请遵循以下步骤。

### 1. 基础配置
**权限声明 (`AndroidManifest.xml`)**:
```xml
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.REQUEST_INSTALL_PACKAGES" />
```
**FileProvider 配置**: (用于 Android 7.0+ 安装 APK)
```xml
<provider
    android:name="androidx.core.content.FileProvider"
    android:authorities="${applicationId}.provider"
    android:exported="false"
    android:grantUriPermissions="true">
    <meta-data
        android:name="android.support.FILE_PROVIDER_PATHS"
        android:resource="@xml/provider_paths" />
</provider>
```
`res/xml/provider_paths.xml`:
```xml
<paths>
    <external-cache-path name="external_cache_files" path="." />
</paths>
```

### 2. 核心工具类 (`UpdateManager.kt`)
你需要封装一个单例来处理检查和更新逻辑。

#### 2.1 获取稳定 DeviceID
为了保证灰度的一致性，你需要一个持久化的唯一ID。
```kotlin
fun getStableDeviceId(context: Context): String {
    val prefs = PreferenceManager.getDefaultSharedPreferences(context)
    var uuid = prefs.getString("device_id", null)
    if (uuid == null) {
        uuid = UUID.randomUUID().toString()
        prefs.edit().putString("device_id", uuid).apply()
    }
    return uuid!!
}
```

#### 2.2 检查更新接口
**API**: `GET /api/check-update`
**参数**:
*   `packageName`: `com.example.app`
*   `versionCode`: `100` (当前本地版本)
*   `deviceId`: `abc-123...` (用于灰度计算)

**响应 JSON**:
```json
{
  "hasUpdate": true,        // 是否有新版本
  "force": false,           // 是否强制更新
  "versionCode": 102,       // 新版本号
  "versionName": "1.0.2",
  "downloadUrl": "/apks/com.example.app_102.apk" // 相对路径
}
```

#### 2.3 实现逻辑 (伪代码)
```kotlin
fun checkUpdate(context: Context) {
    val url = "$SERVER_URL/api/check-update?..."
    
    httpClient.get(url) { response ->
        if (response.hasUpdate) {
            // 拼接完整下载地址
            val fullUrl = "$SERVER_URL${response.downloadUrl}"
            
            if (response.force) {
                showForceUpdateDialog(context, fullUrl)
            } else {
                showOptionalUpdateDialog(context, fullUrl)
            }
        }
    }
}
```

### 3. 高级功能：版本回退/手动选择

为测试人员或高级用户提供一个“版本列表”页面。

**API**: `GET /api/version-list?packageName=...`
**响应**:
```json
{
  "minForceVersionCode": 101, // 安全基线
  "versions": [
    { "versionCode": 102, "versionName": "1.0.2", ... },
    { "versionCode": 101, "versionName": "1.0.1", ... }
  ]
}
```

**开发建议**:
1.  使用 `RecyclerView` 展示列表。
2.  **安全校验**: 在 `onBindViewHolder` 中，如果 `item.versionCode < response.minForceVersionCode`，则**禁用按钮并变灰**，提示“版本过低不可用”，防止用户回退到有风险的旧版本。

### 4. 常见问题排查

*   **Q: 上传后为什么收不到更新？**
  *   A: 检查管理后台，确认 **Rollout Rate** 是否大于 0。如果是新版本，默认是 0% (暂停发布)。
*   **Q: 为什么有的手机能收到，有的收不到？**
  *   A: 这是灰度发布的特性。灰度基于 DeviceID。如果你设置了 50%，那么只有一半的设备能收到。若想全员测试，请设为 100%。
*   **Q: 404 Not Found?**
  *   A: 检查 App 端配置的服务器地址是否多加了 `/api` 后缀。正确格式应为 `http://192.168.x.x:8888`。