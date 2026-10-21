@echo off
chcp 65001 >nul
title 400C 平台停止
cd /d "%~dp0"

echo 正在停止 400C 平台进程...

taskkill /f /im chem-platform.exe >nul 2>&1
taskkill /f /im martin.exe >nul 2>&1

:: 如果用户使用内置独立进程，可一并退出（若是系统服务则不会被意外终止）
taskkill /f /fi "WINDOWTITLE eq 400C-MongoDB*" /im mongod.exe >nul 2>&1
taskkill /f /fi "WINDOWTITLE eq 400C-MQTT*" /im mosquitto.exe >nul 2>&1

echo 所有 400C 服务已安全退出。
ping -n 2 127.0.0.1 >nul
