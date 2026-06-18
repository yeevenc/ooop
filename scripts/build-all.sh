#!/bin/sh
set -e

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

sh "$ROOT_DIR/scripts/build-api.sh"
sh "$ROOT_DIR/scripts/build-web-admin.sh"
