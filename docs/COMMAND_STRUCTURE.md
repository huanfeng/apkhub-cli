# ApkHub 命令结构

ApkHub CLI 现在采用更清晰的命令结构，将仓库管理功能和客户端功能分离。

## 仓库管理命令 (repo)

所有仓库管理相关的命令都在 `repo` 子命令下：

```bash
apkhub repo init              # 初始化仓库配置
apkhub repo add <apk>         # 添加单个 APK 到仓库
apkhub repo scan <dir>        # 批量扫描目录中的 APK
apkhub repo verify            # 验证仓库完整性
apkhub repo clean             # 清理旧版本
apkhub repo list              # 列出仓库中的包
apkhub repo stats             # 显示仓库统计信息
apkhub repo export            # 导出仓库数据
apkhub repo import            # 导入其他格式的数据
apkhub repo parse <apk>       # 解析单个 APK 文件
```

### 使用示例

```bash
# 初始化新仓库
apkhub repo init

# 扫描目录并添加 APK
apkhub repo scan /downloads/apks/

# 查看仓库统计
apkhub repo stats

# 清理旧版本（保留最新3个）
apkhub repo clean --keep 3

# 导出为 CSV
apkhub repo export -f csv -o packages.csv
```

## 客户端命令

客户端功能用于搜索、下载和安装应用：

```bash
apkhub bucket <subcommand>    # 管理仓库源
apkhub search <query>         # 搜索应用
apkhub info <package-id>      # 查看应用详情
apkhub download <package-id>  # 下载应用
apkhub install <package-id>   # 安装应用
```

### 使用示例

```bash
# 添加仓库源
apkhub bucket add main https://apk.example.com

# 搜索应用
apkhub search telegram

# 查看详情
apkhub info org.telegram.messenger

# 安装应用
apkhub install org.telegram.messenger
```

## 命令对比

| 旧命令 | 新命令 | 说明 |
|--------|--------|------|
| `apkhub init` | `apkhub repo init` | 初始化仓库 |
| `apkhub add` | `apkhub repo add` | 添加 APK |
| `apkhub scan` | `apkhub repo scan` | 扫描目录 |
| `apkhub verify` | `apkhub repo verify` | 验证仓库 |
| `apkhub clean` | `apkhub repo clean` | 清理旧版本 |
| `apkhub list` | `apkhub repo list` | 列出包 |
| `apkhub stats` | `apkhub repo stats` | 统计信息 |
| `apkhub export` | `apkhub repo export` | 导出数据 |
| `apkhub import` | `apkhub repo import` | 导入数据 |
| `apkhub parse` | `apkhub repo parse` | 解析 APK |

客户端命令（bucket, search, info, download, install）保持不变。

## 设计理念

这种命令结构设计参考了 Git 等工具的模式：
- **repo** 子命令：管理本地 APK 仓库（类似 git 的本地仓库操作）
- **客户端命令**：与远程仓库交互（类似 git remote/pull/push）

这样的分离使得命令更加清晰，用户可以很容易地区分：
- 我是在管理自己的 APK 仓库？→ 使用 `apkhub repo ...`
- 我是在使用别人的 APK 仓库？→ 使用 `apkhub search/install ...`