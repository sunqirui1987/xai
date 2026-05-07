#!/usr/bin/env bash

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
GROUP_NAME="${1:-${NODESKAI_DEFAULT_GROUP_NAME:-默认素材组}}"
NAME="${2:-测试素材}"
DEFAULT_GROUP_DESCRIPTION="${NODESKAI_DEFAULT_GROUP_DESCRIPTION:-Seedance 2.0 默认素材组}"
START_TS="$(date '+%Y-%m-%d %H:%M:%S')"

echo "[$START_TS] [main.sh] start dir=$DIR group_name=$GROUP_NAME name=$NAME"
echo "[$(date '+%Y-%m-%d %H:%M:%S')] [main.sh] create group"
GROUP_OUT="$(go run "$DIR/main.go" create-group "$GROUP_NAME" "$DEFAULT_GROUP_DESCRIPTION")"
echo "$GROUP_OUT"
GROUP_ID="$(
  printf '%s\n' "$GROUP_OUT" | awk -F': ' '/^group_id:/ {print $2; exit}'
)"
if [ -z "$GROUP_ID" ]; then
  echo "create-group output missing group_id" >&2
  exit 1
fi

echo
echo "[$(date '+%Y-%m-%d %H:%M:%S')] [main.sh] upload z11.jpg group_id=$GROUP_ID"
UPLOAD_OUT="$(go run "$DIR/main.go" upload "$DIR/z11.jpg" "$GROUP_ID" "$NAME")"
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
