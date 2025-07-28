# 包管理器支持设置指南

本文档说明如何为 ApkHub CLI 设置 Homebrew 和 Scoop 包管理器支持。

## 概述

ApkHub CLI 支持通过以下包管理器安装：
- **Homebrew** (macOS/Linux)
- **Scoop** (Windows)

## 前置准备

### 1. 创建 Homebrew Tap 仓库

创建一个名为 `homebrew-tap` 的 GitHub 仓库：

```bash
# 仓库名称：apkhub/homebrew-tap
```

初始化仓库结构：
```
homebrew-tap/
├── README.md
└── Formula/
    └── .gitkeep
```

### 2. 创建 Scoop Bucket 仓库

创建一个名为 `apkhub-scoop-bucket` 的 GitHub 仓库：

```bash
# 仓库名称：huanfeng/apkhub-scoop-bucket
```

初始化仓库结构：
```
apkhub-scoop-bucket/
├── README.md
├── bucket/
│   └── .gitkeep
└── .gitkeep
```

### 3. 生成 GitHub Personal Access Tokens

需要创建两个具有 `repo` 权限的 Personal Access Tokens：

1. **HOMEBREW_TAP_GITHUB_TOKEN**: 用于推送到 homebrew-tap 仓库
2. **SCOOP_BUCKET_GITHUB_TOKEN**: 用于推送到 scoop-bucket 仓库

创建步骤：
1. 访问 https://github.com/settings/tokens
2. 点击 "Generate new token (classic)"
3. 选择 `repo` 权限
4. 生成 token 并保存

### 4. 添加 Secrets 到主仓库

在 apkhub-cli 仓库中添加 secrets：

1. 访问 `https://github.com/huanfeng/apkhub-cli/settings/secrets/actions`
2. 添加以下 secrets：
   - `HOMEBREW_TAP_GITHUB_TOKEN`
   - `SCOOP_BUCKET_GITHUB_TOKEN`

## 使用方法

### 安装 ApkHub CLI

#### 通过 Homebrew (macOS/Linux)

```bash
# 添加 tap
brew tap apkhub/tap

# 安装
brew install apkhub

# 更新
brew upgrade apkhub
```

#### 通过 Scoop (Windows)

```powershell
# 添加 bucket
scoop bucket add apkhub https://github.com/huanfeng/apkhub-scoop-bucket

# 安装
scoop install apkhub

# 更新
scoop update apkhub
```

#### 直接下载

也可以从 [Releases](https://github.com/huanfeng/apkhub-cli/releases) 页面直接下载对应平台的二进制文件。

## 发布流程

当你推送新的标签时，GoReleaser 会自动：

1. 构建所有平台的二进制文件
2. 创建 GitHub Release
3. 更新 Homebrew Formula 到 homebrew-tap 仓库
4. 更新 Scoop Manifest 到 scoop-bucket 仓库

整个过程完全自动化，无需手动干预。

## 仓库模板

### Homebrew Tap README.md 示例

```markdown
# ApkHub Homebrew Tap

## Installation

```bash
brew tap apkhub/tap
brew install apkhub
```

## Available Formulae

- `apkhub` - A command-line tool for managing distributed APK repositories
```

### Scoop Bucket README.md 示例

```markdown
# ApkHub Scoop Bucket

## Installation

```powershell
scoop bucket add apkhub https://github.com/huanfeng/apkhub-scoop-bucket
scoop install apkhub
```

## Available Apps

- `apkhub` - A command-line tool for managing distributed APK repositories
```

## 故障排除

### Homebrew 安装失败

如果遇到权限问题：
```bash
brew doctor
brew cleanup
```

### Scoop 安装失败

如果遇到下载问题：
```powershell
scoop cache rm apkhub
scoop install apkhub
```

### Token 权限不足

确保 Personal Access Token 具有以下权限：
- `repo` (完整的仓库访问权限)
- 如果仓库是组织的，可能需要 SSO 授权

## 维护说明

### 手动更新 Formula/Manifest

虽然 GoReleaser 会自动更新，但如果需要手动修改：

**Homebrew Formula** (`homebrew-tap/Formula/apkhub.rb`):
```ruby
class Apkhub < Formula
  desc "A command-line tool for managing distributed APK repositories"
  homepage "https://github.com/huanfeng/apkhub-cli"
  version "0.2.0"
  license "MIT"

  # ... 其他配置
end
```

**Scoop Manifest** (`scoop-bucket/bucket/apkhub.json`):
```json
{
    "version": "0.2.0",
    "description": "A command-line tool for managing distributed APK repositories",
    "homepage": "https://github.com/huanfeng/apkhub-cli",
    "license": "MIT",
    "architecture": {
        "64bit": {
            "url": "...",
            "hash": "..."
        }
    }
}
```

### 测试本地 Formula/Manifest

**测试 Homebrew Formula**:
```bash
brew install --build-from-source ./Formula/apkhub.rb
```

**测试 Scoop Manifest**:
```powershell
scoop install ./bucket/apkhub.json
```