#!/bin/bash
# 初始化 Scoop Bucket 仓库的脚本

# 创建目录结构
mkdir -p bucket
touch README.md

# 创建 README
cat > README.md << 'EOF'
# ApkHub Scoop Bucket

## Installation

```powershell
scoop bucket add apkhub https://github.com/huanfeng/apkhub-scoop-bucket
scoop install apkhub
```

## Available Apps

- `apkhub` - A command-line tool for managing distributed APK repositories

## Update

To update apkhub:

```powershell
scoop update apkhub
```

## Uninstall

To uninstall apkhub:

```powershell
scoop uninstall apkhub
```

## Troubleshooting

If you encounter any issues:

```powershell
# Clear cache
scoop cache rm apkhub

# Reinstall
scoop uninstall apkhub
scoop install apkhub
```
EOF

echo "Scoop bucket repository initialized!"
echo "Don't forget to:"
echo "1. git init"
echo "2. git add ."
echo "3. git commit -m 'Initial commit'"
echo "4. git remote add origin https://github.com/huanfeng/apkhub-scoop-bucket.git"
echo "5. git push -u origin main"