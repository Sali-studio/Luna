@echo off
echo Restarting Luna Bot...

REM stop.batを呼び出して一度停止
call stop.bat

REM start.batを呼び出して再度起動
call start.bat

echo Luna Bot restarted.