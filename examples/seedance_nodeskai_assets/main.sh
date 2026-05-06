#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
GROUP_ID="${1:-}"
NAME="${2:-测试素材}"
START_TS="$(date '+%Y-%m-%d %H:%M:%S')"

if [ -z "$GROUP_ID" ]; then
  echo "Usage: ./main.sh <group_id> [name]" >&2
  exit 1
fi

echo "[$START_TS] [main.sh] start dir=$DIR group_id=$GROUP_ID name=$NAME"
echo "[$(date '+%Y-%m-%d %H:%M:%S')] [main.sh] upload test.png"
UPLOAD_OUT="$(go run "$DIR/main.go" upload "$DIR/test.png" "$GROUP_ID" "$NAME")"
echo "$UPLOAD_OUT"

ASSET_ID="$(
  printf '%s\n' "$UPLOAD_OUT" | awk -F': ' '/^asset_id:/ {print $2; exit}'
)"

if [ -z "$ASSET_ID" ]; then
  echo "upload output missing asset_id" >&2
  exit 1
fi

echo
echo "[$(date '+%Y-%m-%d %H:%M:%S')] [main.sh] get asset_id=$ASSET_ID"
go run "$DIR/main.go" get "$ASSET_ID"
echo "[$(date '+%Y-%m-%d %H:%M:%S')] [main.sh] done"
