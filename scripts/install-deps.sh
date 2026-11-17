#!/usr/bin/env bash
set -euo pipefail
if [ "$#" -gt 0 ]; then
  case "$1" in
    --help|-h)
      echo "用法: ./scripts/install-deps.sh"
      echo "安装前端依赖（npm ci）并下载后端 Go 模块（go mod download）。首次使用或依赖变化后运行。"
      exit 0 ;;
    *) echo "未知参数: $1" >&2; exit 2 ;;
  esac
fi
for tool in npm go; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "未找到 $tool，请先安装 Node.js/npm 和 Go，并将它们加入 PATH。" >&2
    exit 1
  fi
done
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root/frontend"
npm ci
cd "$root/backend"
go mod download
echo "依赖安装完成。"
