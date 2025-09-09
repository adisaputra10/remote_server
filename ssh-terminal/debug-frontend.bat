@echo off
echo Testing Nginx configuration...

cd ..\frontend

echo.
echo Testing nginx config syntax...
docker run --rm -v "%cd%\nginx.conf:/etc/nginx/conf.d/default.conf" nginx:alpine nginx -t

if %ERRORLEVEL% NEQ 0 (
    echo Nginx configuration test failed!
    exit /b 1
)

echo.
echo Nginx configuration is valid.

echo.
echo Building frontend with nginx config...
docker build -t goteleport-frontend-debug .

if %ERRORLEVEL% NEQ 0 (
    echo Frontend build failed!
    exit /b 1
)

echo.
echo Running frontend container with debug logging...
docker run -d --name frontend-debug -p 8080:80 goteleport-frontend-debug

echo.
echo Waiting for container to start...
timeout /t 5

echo.
echo Testing various routes:
echo.

echo Testing root path:
curl -s -o nul -w "Root (/) - HTTP Status: %%{http_code}\n" http://localhost:8080/

echo Testing dashboard path:
curl -s -o nul -w "Dashboard (/dashboard) - HTTP Status: %%{http_code}\n" http://localhost:8080/dashboard

echo Testing command-logs path:
curl -s -o nul -w "Command Logs (/command-logs) - HTTP Status: %%{http_code}\n" http://localhost:8080/command-logs

echo Testing static assets:
curl -s -o nul -w "CSS files - HTTP Status: %%{http_code}\n" http://localhost:8080/assets/

echo.
echo Nginx access logs:
docker exec frontend-debug cat /var/log/nginx/access.log

echo.
echo Nginx error logs:
docker exec frontend-debug cat /var/log/nginx/error.log

echo.
echo Container logs:
docker logs frontend-debug

echo.
echo To cleanup test:
echo   docker stop frontend-debug
echo   docker rm frontend-debug
echo   docker rmi goteleport-frontend-debug
