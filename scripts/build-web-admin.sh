#!/bin/sh
set -e

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
. "$ROOT_DIR/scripts/use-web-node.sh"

cd "$ROOT_DIR/web-admin"
npm run build

printf '%s\n' "后台管理打包完成: $ROOT_DIR/web-admin/dist"
