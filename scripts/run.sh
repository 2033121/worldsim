#!/bin/sh
# WorldSim 一键启动脚本（Android / Linux 通用）
# 用法：sh run.sh   （或 ./run.sh）
# 说明：sdcard 无执行权限位，二进制自动部署到 /tmp/worldsim_run/ 执行；
#       数据目录 wsdata/ 留在插件包内（持久、可备份）。
DIR="$(cd "$(dirname "$0")" && pwd)"
PORT=48091
DATA="$DIR/wsdata"
BIN_DIR="/tmp/worldsim_run"

# 已在运行？
if curl -s --max-time 2 "http://127.0.0.1:$PORT/api/worlds" >/dev/null 2>&1; then
  echo "✅ WorldSim 已在运行 → http://127.0.0.1:$PORT （浏览器打开即控制台）"
  exit 0
fi

mkdir -p "$DATA/worldbooks" "$BIN_DIR"

# 首次运行：生成 api.json（LLM 配置模板）
if [ ! -f "$DATA/api.json" ]; then
  if [ -f "$DATA/api.json.example" ]; then
    cp "$DATA/api.json.example" "$DATA/api.json"
  fi
  echo "⚙️  已生成 api.json 模板 → 打开 WebUI，在「操作台 → LLM 模式」填入 API Key 后点「应用」"
fi

# 二进制部署到可执行位置（sdcard 无 x 权限位；二进制有更新时自动覆盖）
if [ ! -x "$BIN_DIR/worldsim" ] || ! cmp -s "$DIR/worldsim" "$BIN_DIR/worldsim"; then
  cp -f "$DIR/worldsim" "$BIN_DIR/worldsim" 2>/dev/null || cp -f "$DIR/worldsim" "$BIN_DIR/" 
  chmod +x "$BIN_DIR/worldsim"
fi

# 启动（后台守护；数据目录用插件包内的 wsdata，持久保存）
cd "$BIN_DIR" || exit 1
setsid nohup "$BIN_DIR/worldsim" "$DATA" > "$DATA/run.log" 2>&1 &
sleep 3

if curl -s --max-time 3 "http://127.0.0.1:$PORT/api/worlds" >/dev/null 2>&1; then
  echo "🚀 WorldSim 已启动 → http://127.0.0.1:$PORT"
  echo "   · WebUI 控制台：世界状态 / 决策改选 / 循环开关 / 就绪度 / 小说阅读"
  echo "   · 日志：$DATA/run.log"
else
  echo "⚠️  启动可能失败，看日志：$DATA/run.log"
fi