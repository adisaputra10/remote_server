@echo off
echo ===============================================
echo Building SSH Terminal with Remote Agent
echo ===============================================

cd ssh-terminal

echo Building SSH Terminal...
go build -o ssh-terminal.exe main.go

if errorlevel 1 (
    echo Failed to build SSH terminal application
    pause
    exit /b 1
)

echo.
echo Build successful! Starting SSH Terminal...
echo ===============================================
echo.

.\ssh-terminal.exe

echo.
echo SSH Terminal session ended.
cd ..
pause
