# ApkHub 部署指南

本文档介绍如何部署和维护 ApkHub APK 仓库。

## 目录

1. [快速开始](#快速开始)
2. [部署方式](#部署方式)
3. [自动化维护](#自动化维护)
4. [客户端配置](#客户端配置)
5. [最佳实践](#最佳实践)

## 快速开始

### 1. 安装 ApkHub CLI

```bash
# 从源码构建
git clone https://github.com/huanfeng/apkhub-cli.git
cd apkhub-cli
go build -o apkhub
sudo mv apkhub /usr/local/bin/

# 或者下载预编译版本
wget https://github.com/huanfeng/apkhub-cli/releases/latest/download/apkhub-linux-amd64
chmod +x apkhub-linux-amd64
sudo mv apkhub-linux-amd64 /usr/local/bin/apkhub
```

### 2. 创建仓库

```bash
# 创建仓库目录
mkdir my-apk-repo
cd my-apk-repo

# 初始化配置
apkhub init

# 编辑配置
vim apkhub.yaml
```

### 3. 添加 APK

```bash
# 添加单个 APK
apkhub add /path/to/app.apk

# 批量扫描目录
apkhub scan /path/to/apks/
```

## 部署方式

### 方式一：GitHub Pages

1. 创建 GitHub 仓库
2. 启用 GitHub Pages
3. 使用 GitHub Actions 自动更新

```yaml
# .github/workflows/update.yml
name: Update Repository
on:
  push:
  schedule:
    - cron: '0 0 * * *'
  workflow_dispatch:

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      # ... (参考提供的模板)
```

### 方式二：静态 Web 服务器

#### Nginx 配置

```nginx
server {
    listen 80;
    server_name apk.example.com;
    root /var/www/apkhub;

    # 启用目录浏览
    autoindex on;
    autoindex_format json;

    # CORS 支持
    add_header Access-Control-Allow-Origin *;

    # 优化大文件下载
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;

    # Gzip 压缩（不压缩 APK）
    gzip on;
    gzip_types application/json text/plain;
    gzip_exclude "\.apk$|\.xapk$|\.apkm$";

    location / {
        try_files $uri $uri/ =404;
    }

    # 缓存策略
    location ~ \.(apk|xapk|apkm)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    location ~ \.json$ {
        expires 1h;
        add_header Cache-Control "public, must-revalidate";
    }
}
```

#### Apache 配置

```apache
<VirtualHost *:80>
    ServerName apk.example.com
    DocumentRoot /var/www/apkhub

    <Directory /var/www/apkhub>
        Options Indexes FollowSymLinks
        AllowOverride None
        Require all granted

        # 启用 JSON 格式的目录列表
        IndexOptions FancyIndexing
        HeaderName /HEADER.html
        ReadmeName /README.html
    </Directory>

    # CORS 支持
    Header set Access-Control-Allow-Origin "*"

    # 文件类型
    AddType application/vnd.android.package-archive .apk
    AddType application/vnd.android.package-archive .xapk
    AddType application/vnd.android.package-archive .apkm

    # 缓存策略
    <FilesMatch "\.(apk|xapk|apkm)$">
        Header set Cache-Control "max-age=2592000, public"
    </FilesMatch>

    <FilesMatch "\.json$">
        Header set Cache-Control "max-age=3600, public"
    </FilesMatch>
</VirtualHost>
```

### 方式三：CDN 加速

1. **Cloudflare Pages**

```bash
# 使用 Wrangler CLI
npm install -g wrangler
wrangler pages publish . --project-name=my-apk-repo
```

2. **自定义 CDN 配置**

- 源站：你的静态服务器
- 缓存规则：
  - `*.apk, *.xapk, *.apkm`: 30 天
  - `*.json`: 1 小时
  - 其他文件：24 小时

### 方式四：Docker 部署

```dockerfile
# Dockerfile
FROM nginx:alpine

# 安装 apkhub
COPY --from=golang:1.21 /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

RUN apk add --no-cache git && \
    git clone https://github.com/huanfeng/apkhub-cli.git && \
    cd apkhub-cli && \
    go build -o /usr/local/bin/apkhub && \
    rm -rf /apkhub-cli /usr/local/go && \
    apk del git

# 配置 nginx
COPY nginx.conf /etc/nginx/conf.d/default.conf

# 工作目录
WORKDIR /usr/share/nginx/html

# 初始化脚本
COPY entrypoint.sh /
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
```

```bash
# entrypoint.sh
#!/bin/sh
apkhub init
apkhub scan /import || true
nginx -g 'daemon off;'
```

## 自动化维护

### 使用 Cron

```bash
# 编辑 crontab
crontab -e

# 每小时扫描新 APK
0 * * * * cd /var/www/apkhub && apkhub scan /incoming

# 每天凌晨清理旧版本
0 2 * * * cd /var/www/apkhub && apkhub clean --keep 5 -y

# 每周生成报告
0 0 * * 0 cd /var/www/apkhub && apkhub stats > stats.txt
```

### 使用 Systemd

```ini
# /etc/systemd/system/apkhub-watcher.service
[Unit]
Description=ApkHub Repository Watcher
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/var/www/apkhub
ExecStart=/usr/local/bin/apkhub-watch.sh
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
# /usr/local/bin/apkhub-watch.sh
#!/bin/bash
while true; do
    apkhub scan /incoming
    sleep 300  # 5 分钟
done
```

### 使用 GitHub Actions

参考 `.github/workflows/apkhub-update.yml` 模板。

## 客户端配置

### Android 客户端访问

```kotlin
// 获取仓库索引
val manifestUrl = "https://apk.example.com/apkhub_manifest.json"
val response = httpClient.get(manifestUrl)
val manifest = response.body<ManifestIndex>()

// 下载 APK
val apkUrl = manifest.packages["com.example.app"]
    ?.versions?.get("latest")
    ?.downloadUrl
```

### 命令行访问

```bash
# 获取索引
curl -s https://apk.example.com/apkhub_manifest.json | jq

# 下载特定 APK
wget https://apk.example.com/apks/com.example.app_100_a1b2c3d4.apk
```

### Python 脚本示例

```python
import requests
import json

# 获取仓库索引
manifest = requests.get("https://apk.example.com/apkhub_manifest.json").json()

# 查找应用
app = manifest["packages"].get("com.example.app")
if app:
    latest = app["versions"][app["latest"]]
    print(f"Latest version: {latest['version']}")
    print(f"Download: {latest['download_url']}")
```

## 最佳实践

### 1. 安全建议

- **签名验证**：始终验证 APK 签名
- **HTTPS**：使用 HTTPS 传输
- **访问控制**：对私有仓库设置认证
- **日志监控**：记录下载日志

### 2. 性能优化

- **CDN 加速**：使用 CDN 分发大文件
- **增量更新**：定期运行增量扫描
- **压缩传输**：启用 Gzip（JSON 文件）
- **并行下载**：客户端支持断点续传

### 3. 仓库管理

- **版本控制**：合理设置 `keep_versions`
- **定期清理**：删除孤立文件
- **备份策略**：定期备份仓库数据
- **监控告警**：设置存储空间告警

### 4. 目录结构建议

```
repo/
├── apks/               # APK 文件
├── infos/              # 元数据
├── incoming/           # 待处理 APK
├── backup/             # 备份目录
├── logs/               # 日志文件
├── apkhub.yaml         # 配置文件
├── apkhub_manifest.json # 主索引
├── PACKAGES.md         # 人类可读列表
└── README.md           # 仓库说明
```

### 5. 大规模部署

对于大规模部署（>1000 个 APK），建议：

1. **分片存储**：按包名首字母分目录
2. **索引优化**：使用数据库缓存
3. **异步处理**：使用消息队列
4. **负载均衡**：多节点部署

```bash
# 分片存储示例
apks/
├── a/  # com.android.*
├── b/  # com.baidu.*
├── c/  # com.company.*
└── .../
```

## 故障排查

### 常见问题

1. **扫描速度慢**
   - 使用增量扫描：`apkhub scan --full=false`
   - 禁用 APK 解析：修改配置 `parse_apk_info: false`

2. **存储空间不足**
   - 定期清理：`apkhub clean --keep 3`
   - 使用外部存储：配置 `base_url` 指向 CDN

3. **客户端下载失败**
   - 检查 CORS 配置
   - 验证文件权限
   - 查看服务器日志

### 调试命令

```bash
# 验证仓库完整性
apkhub verify --fix

# 查看详细统计
apkhub stats

# 导出诊断信息
apkhub export -f json -o debug.json

# 检查特定包
apkhub list -p com.example.app
```

## 迁移指南

### 从其他仓库迁移

```bash
# 从 F-Droid 导入
apkhub import -f fdroid fdroid-index-v1.json

# 从旧版 ApkHub 导入
apkhub import -f apkhub old-manifest.json

# 批量导入 JSON 数据
apkhub import -f json data.json \
  --map package_id=app.id \
  --map version=app.version \
  --map sha256=app.hash
```

### 数据备份

```bash
# 备份脚本
#!/bin/bash
DATE=$(date +%Y%m%d)
BACKUP_DIR="/backup/apkhub/$DATE"

mkdir -p "$BACKUP_DIR"
rsync -av --exclude='*.apk' /var/www/apkhub/ "$BACKUP_DIR/"
tar -czf "$BACKUP_DIR.tar.gz" "$BACKUP_DIR"
rm -rf "$BACKUP_DIR"

# 保留最近 7 天的备份
find /backup/apkhub -name "*.tar.gz" -mtime +7 -delete
```

## 监控和告警

### Prometheus 指标

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'apkhub'
    static_configs:
      - targets: ['localhost:9090']
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: 'apkhub_.*'
        action: keep
```

### 自定义监控脚本

```bash
#!/bin/bash
# monitor.sh

# 检查磁盘空间
USAGE=$(df -h /var/www/apkhub | awk 'NR==2 {print $5}' | sed 's/%//')
if [ $USAGE -gt 80 ]; then
    echo "Warning: Disk usage is ${USAGE}%"
fi

# 检查最近更新
LAST_UPDATE=$(stat -c %Y /var/www/apkhub/apkhub_manifest.json)
NOW=$(date +%s)
DIFF=$((NOW - LAST_UPDATE))
if [ $DIFF -gt 86400 ]; then
    echo "Warning: Repository not updated for >24 hours"
fi

# 验证仓库
apkhub verify || echo "Warning: Repository verification failed"
```

---

通过遵循本指南，你可以建立一个稳定、高效、易于维护的 APK 仓库系统。