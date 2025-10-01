@echo off
echo Starting SSH Web Server...
cd ssh-web
start "SSH Web Server" node server.js

echo Starting Frontend...
cd ..\frontend
start "Frontend" npm run dev

echo Starting Relay Server...
cd ..\
start "Relay Server" .\bin\relay.exe

echo All services started in separate windows.
echo Close the respective windows to stop each service.
pause