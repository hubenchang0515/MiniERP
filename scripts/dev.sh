#!/usr/bin/env bash
set -euo pipefail
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root/frontend"
if [ ! -d node_modules ]; then
  echo "缺少前端依赖，请先运行 ./scripts/install-deps.sh。" >&2
  exit 1
fi
npm run build
cp -R dist/. "$root/backend/internal/web/static/"
cd "$root/backend"
if ! GOPROXY=off go build -mod=readonly -o dev-minioa ./cmd/minioa; then
  echo "后端构建失败；如缺少依赖，请先运行 ./scripts/install-deps.sh。" >&2
  exit 1
fi
cd "$root"
./backend/dev-minioa &
backend_pid=$!
frontend_pid=''
cleanup() { kill "$backend_pid" 2>/dev/null || true; if [ -n "$frontend_pid" ]; then kill "$frontend_pid" 2>/dev/null || true; fi; }
trap cleanup EXIT INT TERM
cd "$root/frontend"
npm run dev &
frontend_pid=$!
wait "$frontend_pid"
