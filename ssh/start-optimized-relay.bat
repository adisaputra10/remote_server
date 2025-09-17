@echo off
echo Starting optimized SSH tunnel relay server...
echo.

:: Load performance environment variables
if exist .env.performance (
    echo Loading performance configuration...
    for /f "usebackq tokens=1,2 delims==" %%i in (".env.performance") do (
        if not "%%i"=="" if not "%%i"=="rem" if not "%%i"=="#" (
            set "%%i=%%j"
        )
    )
) else (
    echo Warning: .env.performance file not found, using defaults
    set LOG_LEVEL=INFO
    set PORT=8080
)

echo Configuration:
echo - Log Level: %LOG_LEVEL%
echo - Port: %PORT%
echo - DB Host: %DB_HOST%
echo.

:: Build and run the optimized relay server
echo Building relay server...
go build -o bin/relay-optimized.exe cmd/relay/main.go

if %ERRORLEVEL% NEQ 0 (
    echo Error: Failed to build relay server
    pause
    exit /b 1
)

echo Starting relay server on port %PORT%...
echo Press Ctrl+C to stop
echo.

bin\relay-optimized.exe --port %PORT%

pause