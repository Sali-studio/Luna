@echo off
echo Stopping Luna Bot...
REM 'taskkill'コマンドでプロセス名で強制終了
taskkill /IM luna.exe /F 2>nul
echo Luna Bot stopped.