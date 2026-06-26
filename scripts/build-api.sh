#!/bin/sh
set -e

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUTPUT_DIR="$ROOT_DIR/dist/api"

mkdir -p "$OUTPUT_DIR"

cd "$ROOT_DIR/api"
go build -buildvcs=false -o "$OUTPUT_DIR/ooop-api" ./cmd/api
go build -buildvcs=false -o "$OUTPUT_DIR/ooop-migrate" ./cmd/migrate

printf '%s\n' "API 打包完成: $OUTPUT_DIR"
