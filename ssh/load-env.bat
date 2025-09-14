@echo off
REM Load environment variables from .env file

echo 🔧 Loading environment variables from .env...

if not exist .env (
    echo ❌ Error: .env file not found
    echo Please create .env file from env.template
    exit /b 1
)

REM Read .env file and set environment variables
for /f "usebackq tokens=1,2 delims==" %%a in (".env") do (
    if not "%%a"=="" if not "%%b"=="" (
        REM Skip comments and empty lines
        echo %%a | findstr /r "^#" >nul
        if errorlevel 1 (
            set %%a=%%b
            echo   ✅ %%a=%%b
        )
    )
)

echo ✅ Environment variables loaded successfully
echo.