#!/usr/bin/env bash
# WorldSim 容器入口：首次启动时把源码自带的 worldbooks / material 复制到代码期望的位置
set -e

DATA_DIR="/app/wsdata"
SEED_DIR="/app/seed"
# 代码用 filepath.Join(baseDir=wsdata/worlds, "..", "worldbooks") → /app/wsdata/worldbooks
WB_DIR="/app/wsdata/worldbooks"

mkdir -p "$DATA_DIR" "$WB_DIR"

# 首次启动：复制世界书池（含 themes/）到 $WB_DIR（代码期望路径）
if [ ! -d "$WB_DIR/themes" ]; then
  echo "[entrypoint] 首次启动：复制 worldbooks 模板到 $WB_DIR"
  cp -r "$SEED_DIR/worldbooks/." "$WB_DIR/"
else
  # 已存在则只补 _template.md（避免覆盖用户自建世界书）
  if [ ! -f "$WB_DIR/_template.md" ]; then
    cp "$SEED_DIR/worldbooks/_template.md" "$WB_DIR/_template.md"
  fi
fi

# 首次启动：复制素材库到 wsdata/material（novel.go 按相对路径从 storys 解析到 data-dir/material）
if [ ! -d "$DATA_DIR/material" ]; then
  echo "[entrypoint] 首次启动：复制 material 素材库到 $DATA_DIR/material"
  cp -r "$SEED_DIR/material" "$DATA_DIR/material"
fi

# api.json 检查
if [ ! -f "$DATA_DIR/api.json" ]; then
  echo "[entrypoint] 警告：$DATA_DIR/api.json 不存在，服务启动后无法调用 LLM"
  echo "[entrypoint] 请在宿主数据目录($DATA_DIR)的 api.json 填入 LLM 配置后重启容器"
fi

echo "[entrypoint] 启动 WorldSim：$@"
exec "$@"
