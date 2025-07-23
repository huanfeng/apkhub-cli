#!/bin/bash

# 发布脚本 - 使用 GoReleaser 进行发布

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查是否安装了 goreleaser
check_goreleaser() {
    if ! command -v goreleaser &> /dev/null; then
        echo -e "${RED}错误: 未找到 goreleaser${NC}"
        echo "请先安装 GoReleaser:"
        echo "  brew install goreleaser  # macOS"
        echo "  go install github.com/goreleaser/goreleaser@latest  # 其他平台"
        exit 1
    fi
}

# 检查是否有未提交的更改
check_git_status() {
    if [[ -n $(git status -s) ]]; then
        echo -e "${YELLOW}警告: 有未提交的更改${NC}"
        echo "请先提交所有更改再发布"
        exit 1
    fi
}

# 获取下一个版本号
get_next_version() {
    # 获取最新的标签
    LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    echo -e "${GREEN}最新标签: $LATEST_TAG${NC}"
    
    # 提示输入新版本号
    echo -n "请输入新版本号 (例如: v0.2.0): "
    read NEW_VERSION
    
    if [[ ! "$NEW_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo -e "${RED}错误: 版本号格式不正确，应为 vX.Y.Z 格式${NC}"
        exit 1
    fi
}

# 主流程
main() {
    echo -e "${GREEN}ApkHub CLI 发布工具${NC}"
    echo "======================="
    
    # 检查依赖
    check_goreleaser
    check_git_status
    
    # 选择操作
    echo "请选择操作:"
    echo "1. 测试构建 (不发布)"
    echo "2. 正式发布到 GitHub"
    echo "3. 退出"
    
    read -p "选择 (1-3): " choice
    
    case $choice in
        1)
            echo -e "${GREEN}开始测试构建...${NC}"
            goreleaser build --snapshot --clean
            echo -e "${GREEN}构建完成！文件位于 ./dist/ 目录${NC}"
            ;;
        2)
            get_next_version
            
            # 确认发布
            echo -e "${YELLOW}即将发布版本: $NEW_VERSION${NC}"
            read -p "确认发布? (y/N): " confirm
            
            if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
                echo "取消发布"
                exit 0
            fi
            
            # 创建标签
            echo -e "${GREEN}创建标签 $NEW_VERSION...${NC}"
            git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"
            
            # 推送标签
            echo -e "${GREEN}推送标签到远程仓库...${NC}"
            git push origin "$NEW_VERSION"
            
            echo -e "${GREEN}标签已推送！GitHub Actions 将自动完成发布。${NC}"
            echo "请访问 https://github.com/apkhub/apkhub-cli/actions 查看进度"
            ;;
        3)
            echo "退出"
            exit 0
            ;;
        *)
            echo -e "${RED}无效选择${NC}"
            exit 1
            ;;
    esac
}

main