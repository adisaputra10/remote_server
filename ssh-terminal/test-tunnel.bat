@echo off
echo connect 1863a4245efb8ac3
echo forward 3308 localhost 3307
timeout /t 2 >nul
echo list
timeout /t 1 >nul
echo Starting MySQL test...
start /B go run test-mysql-real.go
timeout /t 5 >nul
echo stop 3308
echo exit
