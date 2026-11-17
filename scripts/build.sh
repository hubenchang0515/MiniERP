#!/usr/bin/env bash
set -euo pipefail
usage() {
  echo "用法: ./scripts/build.sh [--target <系统/架构>]"
  echo "默认仅构建当前主机平台。例如: --target linux/amd64 或 --target darwin/arm64"
  echo "请先运行 ./scripts/install-deps.sh 安装依赖；构建不会自动安装依赖。"
}
target=""
target_set=false
while [ "$#" -gt 0 ]; do
  case "$1" in
    --target|-t)
      if [ "$#" -lt 2 ]; then echo "--target 需要系统/架构参数" >&2; exit 2; fi
      target="$2"; target_set=true; shift 2 ;;
    --target=*) target="${1#*=}"; target_set=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "未知参数: $1" >&2; usage >&2; exit 2 ;;
  esac
done
if [ "$target_set" = false ]; then
  host_os="$(go env GOHOSTOS)"
  host_arch="$(go env GOHOSTARCH)"
  target="$host_os/$host_arch"
fi
supported_targets="$(go tool dist list)"
valid=false
while IFS= read -r candidate; do
  if [ "$candidate" = "$target" ]; then valid=true; break; fi
done <<< "$supported_targets"
if [[ ! "$target" =~ ^[a-z0-9]+/[a-z0-9]+$ ]] || [ "$valid" = false ]; then
  echo "不支持的构建目标: '$target'；可通过 go tool dist list 查看有效目标。" >&2
  exit 2
fi
os="${target%/*}"; arch="${target#*/}"; ext=""
if [ "$os" = windows ]; then ext=.exe; fi
echo "构建目标: $target"
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root/frontend"
if [ ! -d node_modules ]; then
  echo "缺少前端依赖，请先运行 ./scripts/install-deps.sh。" >&2
  exit 1
fi
npm run build
static="$root/backend/internal/web/static"
find "$static" -mindepth 1 -maxdepth 1 ! -name .keep ! -name placeholder.html -exec rm -rf -- {} +
cp -R dist/. "$static/"
mkdir -p "$root/release"
cd "$root/backend"
if ! GOPROXY=off CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -mod=readonly -trimpath -ldflags '-s -w' -o "$root/release/minierp-$os-$arch$ext" ./cmd/minierp; then
  echo "Go 构建失败；如缺少依赖，请先运行 ./scripts/install-deps.sh。" >&2
  exit 1
fi
echo "已构建 minierp-$os-$arch$ext"
