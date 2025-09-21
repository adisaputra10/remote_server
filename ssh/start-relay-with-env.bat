@echo off
REM Start relay server with environment variables loaded

echo 🚀 Starting Relay Server with Environment Variables...
echo.

REM Load environment variables
call load-env.bat

echo 🔧 Starting relay server...
echo.

REM Start relay server with loaded environment
bin\relay.exe

pause