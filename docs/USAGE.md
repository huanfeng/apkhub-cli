# ApkHub CLI 使用指南

## 仓库结构

ApkHub 使用标准化的目录结构：

```
repo/
├── apks/                    # APK 文件存储目录
│   ├── com.example.app_100_a1b2c3d4.apk
│   └── com.example.app_101_a1b2c3d4.apk
├── infos/                   # APK 信息 JSON 文件
│   ├── com.example.app.json         # 包信息（不含版本号）
│   └── com.example.app.png          # 应用图标
├── apkhub_manifest.json     # 主索引文件
└── apkhub.yaml             # 配置文件
```

## 文件命名规范

APK 文件按以下格式命名：
```
{package_id}_{version_code}_{signature_hash}_{variant}.apk
```

示例：
- `com.example.app_100_a1b2c3d4.apk` - 基础版本
- `com.example.app_100_a1b2c3d4_arm64v8a.apk` - 特定架构版本

## 基本使用

### 1. 初始化仓库

```bash
# 创建配置文件
./apkhub repo init

# 编辑配置文件
vim apkhub.yaml
```

### 2. 添加单个 APK

```bash
# 添加 APK 到仓库（会提示确认）
./apkhub repo add app.apk

# 跳过确认提示
./apkhub repo add app.apk -y

# 复制而不是移动文件
./apkhub repo add app.apk -c
```

### 3. 批量扫描目录

```bash
# 增量扫描（默认）- 只处理新增或修改的文件
./apkhub repo scan /path/to/apks

# 完整扫描 - 重新处理所有文件
./apkhub repo scan /path/to/apks --full

# 非递归扫描
./apkhub repo scan /path/to/apks -r=false
```

### 4. 验证仓库完整性

```bash
# 检查仓库完整性
./apkhub repo verify

# 检查并修复问题（删除孤立文件）
./apkhub repo verify --fix
```

## 增量更新机制

扫描命令支持增量更新，通过以下方式判断文件是否需要更新：

1. **新文件**：不存在对应的 info JSON 文件
2. **修改文件**：文件修改时间晚于 info 中记录的时间
3. **未变化文件**：跳过处理，提高效率

使用 `--full` 参数可强制重新扫描所有文件。

## 部署建议

### 静态文件服务

1. 使用 Nginx 部署：

```nginx
server {
    listen 80;
    server_name apk.example.com;
    root /var/www/apkhub;
    
    location / {
        autoindex on;
        autoindex_format json;
    }
    
    # 允许跨域访问（如需要）
    add_header Access-Control-Allow-Origin *;
}
```

2. 使用 Python 简单服务器测试：

```bash
cd /path/to/repo
python3 -m http.server 8000
```

### 自动化维护

使用 cron 定期扫描：

```bash
# 每小时扫描一次
0 * * * * cd /path/to/repo && /path/to/apkhub repo scan /incoming/apks
```

## 新增命令

### 1. 查看仓库统计

```bash
# 显示仓库详细统计信息
./apkhub repo stats
```

显示内容包括：
- 包和 APK 数量统计
- SDK 版本分布
- 架构 (ABI) 分布
- 签名分析
- 最大的 APK 文件

### 2. 列出仓库内容

```bash
# 列出所有包
./apkhub repo list

# 按大小排序
./apkhub repo list --sort size

# 显示所有版本
./apkhub repo list -v

# 查看特定包详情
./apkhub repo list -p com.example.app
```

### 3. 清理仓库

```bash
# 预览清理（不实际删除）
./apkhub repo clean --dry-run

# 保留最新 3 个版本
./apkhub repo clean --keep 3

# 自动确认删除
./apkhub repo clean -y

# 只清理孤立文件
./apkhub repo clean --orphans
```

### 4. 导出数据

```bash
# 导出为 JSON（默认）
./apkhub repo export

# 导出为 CSV
./apkhub repo export -f csv -o packages.csv

# 导出为 Markdown
./apkhub repo export -f md -o README.md

# 导出为 F-Droid 格式
./apkhub repo export -f fdroid -o index-v1.json

# 自定义 CSV 字段
./apkhub repo export -f csv --fields package_id,version,size_mb,sha256
```

## 客户端访问

客户端应该：

1. 获取 `apkhub_manifest.json` 文件
2. 解析包列表和版本信息
3. 根据 `download_url` 下载 APK 文件
4. 验证 SHA256 校验和

## 配置说明

### repository 配置

- `base_url`: 如果设置，会作为所有下载 URL 的前缀
- `signature_handling`: 
  - `mark`: 标记不同签名的版本
  - `separate`: 为不同签名创建独立条目
  - `reject`: 拒绝不同签名的 APK

### scanning 配置

- `recursive`: 是否递归扫描子目录
- `parse_apk_info`: 是否解析 APK 详细信息

## 客户端功能（新）

ApkHub 现在提供了类似 Scoop 的客户端功能，可以搜索、下载和安装应用。

### 1. Bucket 管理

Bucket 是 APK 仓库源，你可以添加多个 bucket：

```bash
# 列出所有 bucket
./apkhub bucket list

# 添加新 bucket
./apkhub bucket add main https://apk.example.com
./apkhub bucket add fdroid https://f-droid.org/repo

# 删除 bucket
./apkhub bucket remove fdroid

# 更新 bucket 索引
./apkhub bucket update
./apkhub bucket update main  # 更新特定 bucket

# 启用/禁用 bucket
./apkhub bucket enable main
./apkhub bucket disable fdroid
```

### 2. 搜索应用

```bash
# 基础搜索
./apkhub search chrome
./apkhub search "google chrome"

# 高级搜索
./apkhub search chrome --min-sdk 21
./apkhub search messaging --category communication
./apkhub search game --limit 10
```

### 3. 查看应用信息

```bash
# 显示应用详情
./apkhub info com.android.chrome

# 显示所有可用版本
./apkhub info org.telegram.messenger
```

### 4. 下载应用

```bash
# 下载最新版本
./apkhub download com.android.chrome

# 下载特定版本
./apkhub download com.android.chrome --version 100.0.4896.127

# 强制重新下载
./apkhub download com.android.chrome --force

# 跳过校验和验证
./apkhub download com.android.chrome --no-verify
```

### 5. 安装应用

```bash
# 安装应用（自动下载并安装）
./apkhub install com.android.chrome

# 安装本地 APK 文件
./apkhub install app.apk

# 指定设备安装
./apkhub install com.android.chrome --device emulator-5554

# 安装特定版本
./apkhub install com.android.chrome --version 100.0.4896.127

# 允许降级安装
./apkhub install com.android.chrome --downgrade
```

### 6. 客户端配置

客户端配置保存在 `~/.apkhub/config.yaml`：

```yaml
default_bucket: main
buckets:
  main:
    name: "Main Repository"
    url: "https://apk.example.com"
    enabled: true
    
client:
  download_dir: "~/.apkhub/downloads"
  cache_dir: "~/.apkhub/cache"
  cache_ttl: 3600  # 1 hour
  
adb:
  path: "adb"
  default_device: ""  # 留空自动检测
```

### 7. 典型工作流

```bash
# 1. 添加仓库源
./apkhub bucket add myrepo https://myapks.com

# 2. 搜索应用
./apkhub search telegram

# 3. 查看详情
./apkhub info org.telegram.messenger

# 4. 安装应用
./apkhub install org.telegram.messenger
```