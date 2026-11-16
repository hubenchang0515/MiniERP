#!/usr/bin/env bash
set -euo pipefail
root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root/frontend"
npm ci
npm run build
cp -R dist/. "$root/backend/internal/web/static/"
cd "$root/backend"
go build -o dev-minierp ./cmd/minierp
cd "$root"
./backend/dev-minierp &
backend_pid=$!
frontend_pid=''
cleanup() { kill "$backend_pid" 2>/dev/null || true; if [ -n "$frontend_pid" ]; then kill "$frontend_pid" 2>/dev/null || true; fi; }
trap cleanup EXIT INT TERM
cd "$root/frontend"
npm run dev &
frontend_pid=$!
wait "$frontend_pid"
