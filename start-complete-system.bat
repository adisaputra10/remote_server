@echo off
echo ============================================
echo     GoTeleport Complete System Startup
echo ============================================
echo.

REM Check if Go is installed
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Error: Go is not installed
    echo Please install Go from https://golang.org/
    pause
    exit /b 1
)

REM Check if Node.js is installed
node --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Error: Node.js is not installed
    echo Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

echo ✅ Go and Node.js are installed
echo.

REM Build and start backend server
echo 🔧 Building backend server...
cd /d "%~dp0ssh-terminal"
go build -o goteleport-server-db.exe goteleport-server-db.go
if %errorlevel% neq 0 (
    echo ❌ Error: Failed to build backend server
    pause
    exit /b 1
)

echo ✅ Backend server built successfully
echo.

REM Start backend server in background
echo 🚀 Starting backend server on port 8080...
start "GoTeleport Backend" cmd /c "goteleport-server-db.exe server-config-db.json"

REM Wait a bit for backend to start
timeout /t 3 /nobreak >nul

REM Setup frontend
echo 🔧 Setting up frontend...
cd /d "%~dp0frontend"

if not exist "node_modules" (
    echo 📦 Installing frontend dependencies...
    npm install
    if %errorlevel% neq 0 (
        echo ❌ Error: Failed to install frontend dependencies
        pause
        exit /b 1
    )
)

echo ✅ Frontend setup complete
echo.

echo ============================================
echo            System Started!
echo ============================================
echo.
echo 🖥️  Backend Server: http://localhost:8080
echo 🎨 Frontend (Vue.js): http://localhost:3000
echo.
echo 📝 Backend logs are in separate window
echo 🔗 Frontend will proxy API calls to backend
echo.
echo ⏸️  Press Ctrl+C to stop frontend (backend runs separately)
echo.

REM Start frontend development server
npm run dev

echo.
echo Frontend stopped. Backend may still be running.
echo Check the separate backend window or task manager.
pause
