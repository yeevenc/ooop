#!/bin/sh
set -e

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

cleanup() {
  if [ -n "$API_PID" ]; then
    kill "$API_PID" 2>/dev/null || true
  fi
  if [ -n "$WEB_PID" ]; then
    kill "$WEB_PID" 2>/dev/null || true
  fi
}

trap cleanup INT TERM EXIT

printf '%s\n' '启动 API 服务: http://127.0.0.1:8080'
(
  cd "$ROOT_DIR/api"
  go run ./cmd/api
) &
API_PID=$!

printf '%s\n' '启动后台管理: http://127.0.0.1:5173'
(
  cd "$ROOT_DIR/web-admin"
  npm run dev -- --host 0.0.0.0
) &
WEB_PID=$!

wait
