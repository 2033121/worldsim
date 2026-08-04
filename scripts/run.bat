@echo off
chcp 65001 >nul
REM WorldSim 一键启动（Windows）
REM 用法：双击 run.bat 或命令行运行
setlocal
set "PORT=48091"
set "DIR=%~dp0"
set "DATA=%DIR%wsdata"

REM 已在运行？
curl -s --max-time 2 "http://127.0.0.1:%PORT%/api/worlds" >nul 2>&1
if %errorlevel%==0 (
  echo [OK] WorldSim 已在运行 → http://127.0.0.1:%PORT%
  exit /b 0
)

if not exist "%DATA%" mkdir "%DATA%"
if not exist "%DATA%\worldbooks" mkdir "%DATA%\worldbooks"

REM 首次运行：生成 api.json（LLM 配置模板）
if not exist "%DATA%\api.json" (
  if exist "%DATA%\api.json.example" copy /y "%DATA%\api.json.example" "%DATA%\api.json" >nul
  echo [配置] 已生成 api.json 模板 → 打开 WebUI 在「操作台-LLM 模式」填 API Key 后点「应用」
)

REM 启动（后台）
start "WorldSim" /min "%DIR%worldsim.exe" "%DATA%"
timeout /t 3 /nobreak >nul

curl -s --max-time 3 "http://127.0.0.1:%PORT%/api/worlds" >nul 2>&1
if %errorlevel%==0 (
  echo [OK] WorldSim 已启动 → http://127.0.0.1:%PORT%
  echo     浏览器打开即控制台：世界状态 / 决策改选 / 循环开关 / 时间回退 / 小说阅读
) else (
  echo [错误] 启动可能失败，查看 %DATA%\run.log
)
pause