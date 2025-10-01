#!/bin/bash

# Start SSH Web Server in background
echo "Starting SSH Web Server..."
cd ssh-web
node server.js &
SSH_WEB_PID=$!

# Start main application
echo "Starting main application..."
cd ../frontend
npm run dev &
FRONTEND_PID=$!

# Start relay server
echo "Starting relay server..."
cd ../
./bin/relay.exe &
RELAY_PID=$!

echo "All services started:"
echo "SSH Web Server PID: $SSH_WEB_PID"
echo "Frontend PID: $FRONTEND_PID"
echo "Relay Server PID: $RELAY_PID"

# Function to cleanup on exit
cleanup() {
    echo "Stopping all services..."
    kill $SSH_WEB_PID 2>/dev/null
    kill $FRONTEND_PID 2>/dev/null
    kill $RELAY_PID 2>/dev/null
    echo "All services stopped."
}

# Set trap to cleanup on script exit
trap cleanup EXIT

# Wait for user to press Ctrl+C
echo "Press Ctrl+C to stop all services"
wait