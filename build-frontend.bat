@echo off
echo Building GoTeleport Frontend for Production...
echo.

REM Navigate to frontend directory
cd /d "%~dp0frontend"

REM Install dependencies if node_modules doesn't exist
if not exist "node_modules" (
    echo Installing dependencies...
    npm install
)

echo Building production files...
npm run build

if %errorlevel% equ 0 (
    echo.
    echo Build completed successfully!
    echo Built files are in: frontend/dist/
    echo.
    echo To serve the production build:
    echo npm run serve
) else (
    echo.
    echo Build failed!
)

pause
