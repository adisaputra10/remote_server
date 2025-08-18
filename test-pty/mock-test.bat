@echo off
timeout 3 test-ssh-pty.exe -relay-url ws://localhost:9999/ws/client -agent mock -token test123 -user test
