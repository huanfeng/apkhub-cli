#!/bin/bash
# 初始化 Homebrew Tap 仓库的脚本

# 创建目录结构
mkdir -p Formula
touch README.md

# 创建 README
cat > README.md << 'EOF'
# ApkHub Homebrew Tap

## Installation

```bash
brew tap apkhub/tap
brew install apkhub
```

## Available Formulae

- `apkhub` - A command-line tool for managing distributed APK repositories

## Manual Installation

If you want to install a specific version:

```bash
brew install apkhub/tap/apkhub@0.2.0
```

## Troubleshooting

If you encounter any issues:

```bash
brew doctor
brew update
brew reinstall apkhub
```
EOF

echo "Homebrew tap repository initialized!"
echo "Don't forget to:"
echo "1. git init"
echo "2. git add ."
echo "3. git commit -m 'Initial commit'"
echo "4. git remote add origin https://github.com/huanfeng/apkhub-homebrew-tap.git"
echo "5. git push -u origin main"