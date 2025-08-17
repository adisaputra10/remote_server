@echo off
REM Generate self-signed certificates for sh.adisaputra.online

echo ========================================
echo Self-Signed Certificate Generator
echo ========================================

set DOMAIN=sh.adisaputra.online
set CERT_DIR=certs
set CERT_FILE=%CERT_DIR%\server.crt
set KEY_FILE=%CERT_DIR%\server.key

echo Domain: %DOMAIN%
echo Certificate Directory: %CERT_DIR%
echo ========================================

REM Create certificates directory
if not exist "%CERT_DIR%" mkdir "%CERT_DIR%"

REM Check if OpenSSL is available
where openssl >nul 2>&1
if errorlevel 1 (
    echo ❌ OpenSSL not found!
    echo Please install OpenSSL from:
    echo https://slproweb.com/products/Win32OpenSSL.html
    echo.
    echo Or use Windows Subsystem for Linux (WSL):
    echo wsl ./generate-certs.sh
    pause
    exit /b 1
)

echo Generating private key...
openssl genrsa -out "%CERT_DIR%\server.key" 2048

echo Creating certificate configuration...
(
echo [req]
echo distinguished_name = req_distinguished_name
echo req_extensions = v3_req
echo prompt = no
echo.
echo [req_distinguished_name]
echo C=ID
echo ST=Jakarta
echo L=Jakarta
echo O=Remote Tunnel
echo OU=IT Department
echo CN=%DOMAIN%
echo.
echo [v3_req]
echo keyUsage = nonRepudiation, digitalSignature, keyEncipherment
echo subjectAltName = @alt_names
echo.
echo [alt_names]
echo DNS.1 = %DOMAIN%
echo DNS.2 = *.%DOMAIN%
echo DNS.3 = localhost
echo IP.1 = 127.0.0.1
) > "%CERT_DIR%\server.conf"

echo Generating certificate signing request...
openssl req -new -key "%CERT_DIR%\server.key" -out "%CERT_DIR%\server.csr" -config "%CERT_DIR%\server.conf"

echo Generating self-signed certificate (valid for 365 days)...
openssl x509 -req -in "%CERT_DIR%\server.csr" -signkey "%CERT_DIR%\server.key" -out "%CERT_DIR%\server.crt" -days 365 -extensions v3_req -extfile "%CERT_DIR%\server.conf"

echo.
echo ✅ Self-signed certificate generated successfully!
echo.
echo Files created:
echo - Private Key: %CD%\%CERT_DIR%\server.key
echo - Certificate: %CD%\%CERT_DIR%\server.crt
echo - CSR: %CD%\%CERT_DIR%\server.csr
echo - Config: %CD%\%CERT_DIR%\server.conf
echo.
echo Certificate Details:
openssl x509 -in "%CERT_DIR%\server.crt" -text -noout | findstr "Subject:"
openssl x509 -in "%CERT_DIR%\server.crt" -text -noout | findstr "DNS:"

echo.
echo To use these certificates:
echo 1. Certificates are already configured in .env.production
echo.
echo 2. Start relay server:
echo    start-relay.bat
echo.
echo 3. Test certificate:
echo    test-domain.bat
echo.
echo ⚠️  Note: Clients will need to accept self-signed certificate
echo    or use -k flag with curl commands
echo.
pause
