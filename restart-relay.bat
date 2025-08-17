@echo off
REM Restart relay server with port 8443

echo ========================================
echo Restarting Relay Server (Port 8443)
echo ========================================

echo [1/3] Stopping existing relay...
call stop-relay.bat

echo.
echo [2/3] Verifying configuration...

REM Check .env.production
if exist ".env.production" (
    echo ✅ .env.production found
    
    REM Check binary
    if exist "bin\relay.exe" (
        echo ✅ Relay binary found
    ) else (
        echo ❌ Relay binary not found. Building...
        call build.bat
        if errorlevel 1 (
            echo ❌ Build failed
            pause
            exit /b 1
        )
        echo ✅ Build successful
    )
) else (
    echo ❌ .env.production not found
    pause
    exit /b 1
)

echo.
echo [3/3] Starting relay on port 8443...

REM Start relay server in background
start "Relay Server" /min cmd /c "start-relay.bat"

REM Wait and check
timeout /t 5 /nobreak >nul

echo Verifying port 8443...
netstat -an | find ":8443" | find "LISTENING" >nul
if %ERRORLEVEL% EQU 0 (
    echo ✅ Relay server is listening on port 8443
    netstat -an | find ":8443"
) else (
    echo ❌ Relay server is not listening on port 8443
    echo Check for errors
)

echo.
echo ========================================
echo Restart Complete
echo ========================================
echo.
echo Commands:
echo - Check status: netstat -an ^| find ":8443"
echo - Stop server: stop-relay.bat
echo - Test health: curl -k https://sh.adisaputra.online:8443/health
echo.
pause
