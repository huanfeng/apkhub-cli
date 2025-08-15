# ApkHub CLI

 [English](README.md) | [简体中文](README_zh.md)

一个分布式 APK 仓库管理工具，类似于 Windows 的 Scoop 包管理器，让您轻松创建、维护和使用 APK 仓库。

## 🎯 什么是 ApkHub？

ApkHub CLI 是一个**分布式 APK 仓库系统**，工作方式类似 Scoop 包管理器：

- **🏗️ 仓库模式**: 创建和维护 APK 仓库（类似创建 Scoop bucket）
- **📱 客户端模式**: 从多个仓库搜索、下载和安装 APK（类似使用 Scoop）
- **🌐 分布式**: 无需中央服务器 - 仓库可托管在任何地方
- **🔄 多格式**: 支持 APK、XAPK（APKPure）和 APKM（APKMirror）格式

## 🚀 核心功能

### 🏗️ 仓库管理 (`apkhub repo`)
创建和维护您自己的 APK 仓库：

- **初始化**: 使用可定制配置建立新仓库
- **扫描解析**: 自动发现和解析 APK/XAPK/APKM 文件
- **元数据提取**: 提取全面的应用信息（权限、签名、图标）
- **索引生成**: 创建标准化的 `apkhub_manifest.json` 文件
- **完整性验证**: SHA256 校验和及仓库验证
- **批量操作**: 增量更新和批量处理
- **导入导出**: 支持多种格式（JSON、CSV、Markdown、F-Droid）

### 📱 客户端操作 (`apkhub bucket`, `apkhub search`, `apkhub install`)
像包管理器一样使用 APK 仓库：

- **多仓库管理**: 管理多个 APK 源（存储桶）
- **智能搜索**: 在所有配置的仓库中查找应用
- **直接安装**: 通过 ADB 直接安装 APK 到 Android 设备
- **下载管理**: 自动验证和断点续传支持
- **离线模式**: 网络不可用时使用缓存数据工作
- **健康监控**: 跟踪仓库状态和连接性

### 🛠️ 系统工具
- **医生命令**: 全面诊断和自动修复功能
- **设备管理**: 监控和管理连接的 Android 设备
- **依赖处理**: 自动工具检测和安装

## 📦 安装

### 预编译二进制文件
从 [GitHub Releases](https://github.com/huanfeng/apkhub-cli/releases) 下载最新版本：

```bash
# Linux/macOS
curl -L https://github.com/huanfeng/apkhub-cli/releases/latest/download/apkhub-linux-x86_64.tar.gz -o apkhub.tar.gz
tar xzf apkhub.tar.gz
sudo mv apkhub /usr/local/bin/
```

### 包管理器

#### Homebrew (macOS/Linux)
```bash
brew tap huanfeng/tap
brew install apkhub
```

#### Scoop (Windows)
```bash
scoop bucket add apkhub https://github.com/huanfeng/apkhub-scoop-bucket
scoop install apkhub
```

### 从源码构建
```bash
git clone https://github.com/huanfeng/apkhub-cli.git
cd apkhub-cli
go build -o apkhub
```

## 🛠️ 快速开始

### 1. 系统健康检查
```bash
# 检查系统依赖和健康状态
apkhub doctor

# 自动修复常见问题
apkhub doctor --fix
```

### 2. 🏗️ 仓库管理（创建您自己的 APK 仓库）

```bash
# 初始化新仓库
apkhub repo init

# 扫描目录中的 APK 文件
apkhub repo scan /path/to/apks

# 添加单个 APK 到仓库
apkhub repo add app.apk

# 查看仓库统计信息
apkhub repo stats

# 验证仓库完整性
apkhub repo verify

# 导出仓库数据
apkhub repo export --format csv
```

### 3. 📱 客户端操作（使用 APK 仓库）

```bash
# 添加仓库源（存储桶）
apkhub bucket add myrepo https://example.com/apkhub_manifest.json

# 列出所有配置的仓库
apkhub bucket list

# 在所有仓库中搜索应用程序
apkhub search telegram

# 获取详细应用信息
apkhub info org.telegram.messenger

# 下载 APK
apkhub download org.telegram.messenger

# 直接安装到 Android 设备
apkhub install org.telegram.messenger

# 安装本地 APK 文件
apkhub install /path/to/app.apk
```

### 4. 📱 设备管理

```bash
# 列出连接的 Android 设备
apkhub devices

# 实时监控设备状态
apkhub devices --watch

# 安装到指定设备
apkhub install --device emulator-5554 app.apk
```

## 📋 命令参考

### 🏗️ 仓库管理命令 (`apkhub repo`)
创建和维护 APK 仓库：

- `apkhub repo init` - 使用配置初始化新仓库
- `apkhub repo scan <directory>` - 扫描目录中的 APK/XAPK/APKM 文件
- `apkhub repo add <apk-file>` - 添加单个 APK 到仓库
- `apkhub repo clean` - 清理旧版本和孤立文件
- `apkhub repo stats` - 显示详细仓库统计信息
- `apkhub repo verify` - 验证仓库完整性并修复问题
- `apkhub repo export` - 导出仓库数据（JSON/CSV/Markdown）
- `apkhub repo import` - 从其他格式导入（F-Droid 等）

### 📱 客户端命令（使用仓库）
像包管理器一样使用 APK 仓库：

#### 仓库源管理
- `apkhub bucket list` - 列出所有配置的仓库源
- `apkhub bucket add <name> <url>` - 添加新的仓库源
- `apkhub bucket remove <name>` - 移除仓库源
- `apkhub bucket update` - 更新所有仓库源
- `apkhub bucket health` - 检查仓库健康状态

#### 应用发现与安装
- `apkhub search <query>` - 在所有仓库中搜索应用程序
- `apkhub info <package-id>` - 显示详细应用程序信息
- `apkhub list` - 列出所有可用包
- `apkhub download <package-id>` - 下载 APK 文件
- `apkhub install <package-id|apk-path>` - 安装应用程序到设备

#### 缓存管理
- `apkhub cache` - 管理本地仓库缓存

### 🛠️ 系统和设备命令
- `apkhub doctor` - 系统诊断和自动修复
- `apkhub devices` - 列出和管理 Android 设备
- `apkhub deps` - 检查和安装依赖
- `apkhub version` - 显示版本信息

## 🔧 配置

### 仓库配置 (`apkhub.yaml`)
```yaml
repository:
  name: "我的 APK 仓库"
  description: "个人 APK 收藏"
  base_url: "https://example.com"
  
directories:
  apks: "./apks"
  icons: "./icons"
  info: "./info"

settings:
  icon_size: 512
  keep_versions: 3
  generate_thumbnails: true
```

### 客户端配置 (`~/.apkhub/config.yaml`)
```yaml
default_bucket: "main"
buckets:
  main:
    name: "main"
    url: "https://apkhub.example.com/apkhub_manifest.json"
    enabled: true
  
client:
  download_dir: "~/Downloads/apkhub"
  cache_dir: "~/.apkhub/cache"
  cache_ttl: 3600

adb:
  path: "adb"
  default_device: ""
```

## 📊 仓库格式

生成的 `apkhub_manifest.json` 遵循以下结构：

```json
{
  "version": "1.0",
  "name": "我的 APK 仓库",
  "description": "个人 APK 收藏",
  "updated_at": "2025-01-15T10:00:00Z",
  "total_apks": 150,
  "packages": {
    "com.example.app": {
      "package_id": "com.example.app",
      "name": {
        "en": "Example App",
        "zh": "示例应用"
      },
      "description": "一个示例应用程序",
      "category": "productivity",
      "versions": {
        "1.0.0": {
          "version": "1.0.0",
          "version_code": 100,
          "min_sdk": 21,
          "target_sdk": 33,
          "size": 5242880,
          "sha256": "abc123...",
          "download_url": "https://example.com/apks/com.example.app-1.0.0.apk",
          "icon_url": "https://example.com/icons/com.example.app.png",
          "permissions": ["android.permission.INTERNET"],
          "features": ["android.hardware.camera"],
          "abis": ["arm64-v8a", "armeabi-v7a"],
          "signature": {
            "sha256": "def456...",
            "issuer": "CN=Example Corp",
            "subject": "CN=Example App"
          },
          "release_date": "2025-01-15T10:00:00Z"
        }
      }
    }
  }
}
```

## 🔍 系统要求

### 基本要求
- Go 1.22+（从源码构建时）
- 50MB+ 可用磁盘空间

### APK 解析依赖
工具使用多种解析方法以获得最大兼容性：

1. **主要方式**: 内置 Go 库（`github.com/shogo82148/androidbinary`）
2. **备用方式**: AAPT/AAPT2 命令行工具（推荐用于完全兼容性）

#### 安装 AAPT2

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install aapt
# 或者安装更新版本
sudo apt-get install google-android-build-tools-installer
```

**macOS:**
```bash
# 安装 Android SDK 命令行工具
brew install --cask android-commandlinetools
# aapt2 将位于: ~/Library/Android/sdk/build-tools/*/aapt2
```

**Windows:**
```bash
# 使用 Scoop
scoop bucket add extras
scoop install android-sdk

# 使用 Chocolatey
choco install android-sdk
```

### 设备安装所需的 ADB
**Ubuntu/Debian:**
```bash
sudo apt-get install android-tools-adb
```

**macOS:**
```bash
brew install android-platform-tools
```

**Windows:**
```bash
# 使用 Scoop
scoop install adb

# 使用 Chocolatey
choco install adb
```

## 🚀 高级用法

### 🏗️ 仓库管理工作流

#### 自动化仓库维护
```bash
# 带进度的完整仓库扫描
apkhub repo scan --recursive --progress /path/to/apks

# 增量更新（仅新增/更改的文件）
apkhub repo scan --incremental /path/to/apks

# 清理旧版本（保留最新 3 个）
apkhub repo clean --keep 3

# 验证并自动修复问题
apkhub repo verify --fix
```

#### 批量操作
```bash
# 导出仓库数据
apkhub repo export --format csv --output apps.csv
apkhub repo export --format markdown --output README.md

# 从 F-Droid 导入
apkhub repo import --format fdroid https://f-droid.org/repo/index-v1.json
```

#### CI/CD 集成
```yaml
# GitHub Actions 示例
- name: 更新 APK 仓库
  run: |
    apkhub repo scan ./apks
    apkhub repo verify --quiet
    git add apkhub_manifest.json
    git commit -m "更新仓库索引"
```

### 📱 客户端使用工作流

#### 多仓库设置
```bash
# 添加多个仓库源
apkhub bucket add official https://apkhub.example.com/apkhub_manifest.json
apkhub bucket add fdroid https://f-droid.org/repo/apkhub_manifest.json
apkhub bucket add personal https://my-repo.com/apkhub_manifest.json

# 在所有仓库中搜索
apkhub search "telegram"

# 从任何仓库安装
apkhub install org.telegram.messenger
```

#### 批量安装
```bash
# 从列表安装多个应用
cat app-list.txt | xargs -I {} apkhub install {}

# 使用特定选项安装
apkhub install --device emulator-5554 --version 1.2.3 com.example.app
```

## 🤝 贡献

1. Fork 仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [androidbinary](https://github.com/shogo82148/androidbinary) - APK 解析库
- [Cobra](https://github.com/spf13/cobra) - CLI 框架
- [Viper](https://github.com/spf13/viper) - 配置管理

## 📞 支持

- 📖 [文档](https://github.com/huanfeng/apkhub-cli/wiki)
- 🐛 [问题跟踪](https://github.com/huanfeng/apkhub-cli/issues)
- 💬 [讨论](https://github.com/huanfeng/apkhub-cli/discussions)