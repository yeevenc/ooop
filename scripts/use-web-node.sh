#!/bin/sh
# 本仓库 web-admin（Vite 8）需要 Node 20.19+ 或 22.12+。
# 机器上默认 PATH 被鸿蒙 DevEco 的 Node 18 占住，这里只在本仓库脚本里切到系统 Node。
# 不改全局 Node，避免影响其他项目。

OOOP_NODE_BIN="${OOOP_NODE_BIN:-/usr/local/bin}"

if [ ! -x "$OOOP_NODE_BIN/node" ]; then
  printf '%s\n' "未找到 Node：$OOOP_NODE_BIN/node" >&2
  printf '%s\n' "本项目需要 Node 20.19+ 或 22.12+，请安装后再启动。" >&2
  exit 1
fi

PATH="$OOOP_NODE_BIN:$PATH"
export PATH

NODE_VER="$("$OOOP_NODE_BIN/node" -p "process.versions.node")"
NODE_MAJOR="${NODE_VER%%.*}"
if [ "$NODE_MAJOR" -lt 20 ]; then
  printf '%s\n' "当前 Node v$NODE_VER 过低，本项目需要 20.19+ 或 22.12+" >&2
  exit 1
fi
