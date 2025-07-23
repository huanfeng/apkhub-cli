# 构建说明

## 开发构建

### 简单构建（不含版本信息）
```bash
go build -o apkhub
```

### 包含版本信息的构建
```bash
# 手动指定版本
go build -ldflags "-X github.com/apkhub/apkhub-cli/internal/version.Version=v1.0.0" -o apkhub

# 查看版本
./apkhub version
```

## 正式发布

本项目使用 GoReleaser 进行自动化发布。

### 自动发布（推荐）
```bash
# 1. 创建标签
git tag -a v0.2.0 -m "Release v0.2.0"

# 2. 推送标签，GitHub Actions 会自动发布
git push origin v0.2.0
```

### 本地测试构建
```bash
# 安装 GoReleaser
go install github.com/goreleaser/goreleaser@latest

# 测试构建所有平台
goreleaser build --snapshot --clean
```

### 使用发布脚本
```bash
./scripts/release.sh
```

详细说明请参考 [docs/GORELEASER.md](docs/GORELEASER.md)