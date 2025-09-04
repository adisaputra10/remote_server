@echo off
echo ============================================
echo      GoTeleport Vue.js Frontend Demo
echo ============================================
echo.

REM Check if Node.js is installed
node --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Error: Node.js is not installed
    echo Please install Node.js from https://nodejs.org/
    echo Minimum version required: Node.js 16+
    pause
    exit /b 1
)

echo ✅ Node.js is installed
node --version

REM Navigate to frontend directory
cd /d "%~dp0frontend"

REM Check if package.json exists
if not exist "package.json" (
    echo ❌ Error: package.json not found
    echo Make sure you're running this from the correct directory
    pause
    exit /b 1
)

echo ✅ Frontend directory found

REM Install dependencies if node_modules doesn't exist
if not exist "node_modules" (
    echo.
    echo 📦 Installing dependencies...
    echo This may take a few minutes on first run...
    npm install
    if %errorlevel% neq 0 (
        echo ❌ Error: Failed to install dependencies
        pause
        exit /b 1
    )
    echo ✅ Dependencies installed successfully
) else (
    echo ✅ Dependencies already installed
)

echo.
echo ============================================
echo           Starting Frontend Server
echo ============================================
echo.
echo 🚀 Frontend URL: http://localhost:3000
echo 🔗 API Backend: http://localhost:8080  
echo.
echo 📝 Make sure GoTeleport backend server is running on port 8080
echo.
echo ⏸️  Press Ctrl+C to stop the development server
echo.

REM Start development server
npm run dev

echo.
echo Frontend server stopped.
pause
