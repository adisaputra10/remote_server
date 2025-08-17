@echo off
REM Generate secure token for tunnel authentication

echo ========================================
echo Remote Tunnel - Token Generator
echo ========================================

echo Generating secure authentication token...
echo.

REM Try different methods to generate random token
powershell -Command "try { Add-Type -AssemblyName System.Web; [System.Web.Security.Membership]::GeneratePassword(32, 8) } catch { -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 32 | ForEach {[char]$_}) }" > temp_token.txt

if exist temp_token.txt (
    set /p NEW_TOKEN=<temp_token.txt
    del temp_token.txt
    
    echo Generated Token: %NEW_TOKEN%
    echo.
    echo ========================================
    echo IMPORTANT: Save this token securely!
    echo ========================================
    echo.
    echo 1. Copy this token to your .env.production file:
    echo    TUNNEL_TOKEN=%NEW_TOKEN%
    echo.
    echo 2. Use the SAME token on:
    echo    - Relay server (103.195.169.32)
    echo    - Agent (your laptop)
    echo    - Client (remote connections)
    echo.
    echo 3. Keep this token secret and secure!
    echo.
    
    echo Do you want to automatically update .env.production? (y/n)
    set /p update_env=
    
    if /i "%update_env%"=="y" (
        if exist .env.production (
            REM Backup existing file
            copy .env.production .env.production.backup >nul
            echo Backed up existing .env.production to .env.production.backup
        )
        
        REM Update or create .env.production
        powershell -Command "(Get-Content .env.production -ErrorAction SilentlyContinue) -replace 'TUNNEL_TOKEN=.*', 'TUNNEL_TOKEN=%NEW_TOKEN%' | Out-File -encoding ASCII .env.production"
        
        REM If token line doesn't exist, add it
        findstr /C:"TUNNEL_TOKEN" .env.production >nul || echo TUNNEL_TOKEN=%NEW_TOKEN% >> .env.production
        
        echo.
        echo ✅ Token updated in .env.production
    )
    
) else (
    echo ❌ Failed to generate token automatically
    echo.
    echo Please manually create a secure token:
    echo - Use a password generator
    echo - Minimum 32 characters
    echo - Mix of letters, numbers, and symbols
)

echo.
echo Remember to:
echo - Copy this token to your relay server
echo - Keep it secure and private
echo - Change it regularly for security
echo.
pause
