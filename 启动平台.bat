@echo off
chcp 65001 >nul
title 400C 平台控制台
cd /d "%~dp0"

echo ========================================================
echo       400C 化学物质智能监测综合态势平台
echo ========================================================
echo.

:: 1. 检查或启动 MongoDB
netstat -ano | findstr ":27017 " >nul 2>&1
if %errorlevel% equ 0 (
    echo [1/3] MongoDB 数据库已在运行.
) else (
    echo [1/3] 启动内置 MongoDB 数据库...
    if not exist "data\db" mkdir "data\db"
    start "400C-MongoDB" /min "%~dp0services\mongodb\mongod.exe" --dbpath "%~dp0data\db" --port 27017 --bind_ip 127.0.0.1 --quiet
)

:: 2. 检查或启动 Mosquitto MQTT
netstat -ano | findstr ":1883 " >nul 2>&1
if %errorlevel% equ 0 (
    echo [2/3] Mosquitto MQTT 已在运行.
) else (
    echo [2/3] 启动内置 Mosquitto MQTT...
    start "400C-MQTT" /min "%~dp0services\mosquitto\mosquitto.exe" -c "%~dp0services\mosquitto\mosquitto.conf"
)

echo.
echo 等待核心基础组件就绪...
ping -n 2 127.0.0.1 >nul

:: 3. 启动主程序 (内置自动托管离线地图服务)
echo [3/3] 启动 400C 平台服务 (自动托管离线瓦片引擎)...
set HTTPD_URI=:8080
set DATABASE_URI=mongodb://127.0.0.1:27017
set MQTT_BROKER=tcp://127.0.0.1:1883
set TILE_SERVER_URL=http://127.0.0.1:8081
set SETUP_TOKEN=400c-offline-delivery-admin-setup-token-2026

:: 打开默认浏览器
start "" "http://127.0.0.1:8080/map#setup=400c-offline-delivery-admin-setup-token-2026"

echo.
echo ========================================================
echo  平台已启动成功！
echo  访问地址: http://127.0.0.1:8080/map
echo.
echo  首次初始化凭据: 400c-offline-delivery-admin-setup-token-2026
echo  管理员密码要求: 至少 12 位 (例如: Admin@12345678)
echo.
echo  提示: 请保持此窗口开启。如需退出平台，可直接关闭此窗口。
echo ========================================================
echo.

"%~dp0chem-platform.exe" httpd

call "%~dp0停止平台.bat"
