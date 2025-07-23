# 发布指南 - 使用 GoReleaser

GoReleaser 是一个专为 Go 项目设计的发布自动化工具，能够简化多平台构建和发布流程。

## 为什么选择 GoReleaser？

1. **自动处理版本号**：从 Git 标签自动获取版本
2. **跨平台构建**：一个命令构建所有平台
3. **自动生成 Release Notes**：从 commit 信息生成
4. **与 GitHub 无缝集成**：自动创建 Release 和上传文件

## 安装 GoReleaser

### macOS
```bash
brew install goreleaser
```

### Linux/Windows
```bash
go install github.com/goreleaser/goreleaser@latest
```

### 或使用二进制文件
访问 https://github.com/goreleaser/goreleaser/releases 下载对应平台的二进制文件。

## 本地使用

### 1. 测试构建（不发布）
```bash
# 在项目根目录执行
goreleaser build --snapshot --clean
```

这会在 `dist/` 目录生成所有平台的二进制文件：
```
dist/
├── apkhub_darwin_amd64_v1/
│   └── apkhub
├── apkhub_darwin_arm64/
│   └── apkhub
├── apkhub_linux_amd64_v1/
│   └── apkhub
├── apkhub_linux_arm64/
│   └── apkhub
├── apkhub_windows_amd64_v1/
│   └── apkhub.exe
└── apkhub_windows_arm64/
    └── apkhub.exe
```

### 2. 本地完整发布流程模拟
```bash
# 模拟完整的发布流程（不会真正上传到 GitHub）
goreleaser release --snapshot --clean
```

### 3. 查看版本信息
```bash
# 运行构建的二进制文件
./dist/apkhub_linux_amd64_v1/apkhub version
```

## 正式发布流程

### 1. 确保代码已提交
```bash
git add .
git commit -m "准备发布 v0.2.0"
git push
```

### 2. 创建并推送标签
```bash
# 创建标签
git tag -a v0.2.0 -m "Release v0.2.0"

# 推送标签
git push origin v0.2.0
```

### 3. 自动发布（推荐）

推送标签后，GitHub Actions 会自动运行 GoReleaser 完成发布。

### 4. 手动发布（如需要）

如果需要从本地发布：

```bash
# 设置 GitHub Token
export GITHUB_TOKEN=你的token

# 执行发布
goreleaser release --clean
```

## GoReleaser 功能详解

### 版本号处理

GoReleaser 自动从 Git 标签获取版本号：
- `v1.0.0` → 版本号为 `v1.0.0`
- 自动获取 commit hash 和构建时间
- 所有信息自动注入到二进制文件中

### 构建产物

每次发布会生成：
1. 各平台二进制文件的压缩包
2. SHA256 校验和文件
3. 自动生成的 Release Notes

### 配置文件说明

`.goreleaser.yml` 主要配置：

```yaml
builds:
  - ldflags:
      # 自动注入版本信息
      - -X github.com/apkhub/apkhub-cli/internal/version.Version={{.Version}}
      - -X github.com/apkhub/apkhub-cli/internal/version.Commit={{.Commit}}
      - -X github.com/apkhub/apkhub-cli/internal/version.BuildDate={{.Date}}
```

## 常见问题

### 1. 如何测试配置是否正确？
```bash
goreleaser check
```

### 2. 如何只构建特定平台？
```bash
# 只构建 Linux amd64
goreleaser build --single-target --snapshot --clean
```

### 3. 如何自定义 Release Notes？
在 `.goreleaser.yml` 中配置 `changelog` 部分，或使用 `--release-notes` 参数指定文件。

### 4. 构建失败怎么办？
- 检查 Go 版本是否匹配
- 运行 `go mod tidy` 确保依赖正确
- 查看错误日志定位问题

## 版本号最佳实践

1. **遵循语义化版本**：`v主版本.次版本.修订版本`
2. **使用 v 前缀**：如 `v1.0.0` 而不是 `1.0.0`
3. **预发布版本**：使用 `v1.0.0-beta.1` 格式
4. **始终打标签再发布**：确保版本可追溯