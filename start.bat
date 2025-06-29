@echo off
echo Starting Luna Bot...
REM 'go build'で実行ファイル(luna.exe)を作成
go build -o luna.exe .

REM luna.exeが存在するかチェック
if not exist luna.exe (
    echo Build failed. Exiting.
    exit /b
)

REM 新しいウィンドウを立てずにバックグラウンドで実行
start "LunaBot" /B .\luna.exe
echo Luna Bot started successfully.