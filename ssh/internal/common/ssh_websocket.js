const WebSocket = require('ws');
const { Client } = require('ssh2');
const path = require('path');
const fs = require('fs');

// Setup logging
const logFile = path.join(__dirname, '../logs/ssh_commands.log');

// Function to log commands to file
function logCommand(host, username, command) {
    const timestamp = new Date().toISOString();
    const logEntry = `[${timestamp}] [${username}@${host}] ${command}\n`;
    
    fs.appendFile(logFile, logEntry, (err) => {
        if (err) {
            console.error('Error writing to log file:', err);
        }
    });
}

// Create log file if it doesn't exist
if (!fs.existsSync(logFile)) {
    const logDir = path.dirname(logFile);
    if (!fs.existsSync(logDir)) {
        fs.mkdirSync(logDir, { recursive: true });
    }
    fs.writeFileSync(logFile, '# SSH Command Log\n# Format: [timestamp] [username@host] command\n\n');
}

function createSSHWebSocketServer(server) {
    // Create WebSocket server
    const wss = new WebSocket.Server({ 
        server,
        path: '/ws/ssh'
    });

    // Handle WebSocket connections
    wss.on('connection', (ws, req) => {
        let sshClient = null;
        let isConnected = false;
        let connectionInfo = { host: '', username: '' }; // Store connection info for logging
        let commandBuffer = ''; // Buffer to accumulate command characters
        
        console.log('New SSH WebSocket connection established from:', req.socket.remoteAddress);
        
        // Handle messages from client
        ws.on('message', (message) => {
            try {
                const data = JSON.parse(message);
                
                if (data.type === 'connect') {
                    // Connect to SSH server
                    if (isConnected) {
                        ws.send(JSON.stringify({ type: 'error', message: 'Already connected to an SSH server' }));
                        return;
                    }
                    
                    const { host, port, username, password } = data;
                    
                    // Store connection info for logging
                    connectionInfo = { host, username };
                    
                    sshClient = new Client();
                    
                    sshClient.on('ready', () => {
                        isConnected = true;
                        ws.send(JSON.stringify({ type: 'connected' }));
                        console.log(`SSH connection established to ${host}`);
                        
                        // Start an interactive shell session
                        sshClient.shell((err, stream) => {
                            if (err) {
                                isConnected = false;
                                ws.send(JSON.stringify({ type: 'error', message: err.message }));
                                console.error('Shell error:', err);
                                return;
                            }
                            
                            // Forward data from SSH to WebSocket
                            stream.on('data', (data) => {
                                ws.send(JSON.stringify({ type: 'data', data: data.toString('utf8') }));
                            });
                            
                            // Handle shell closure
                            stream.on('close', () => {
                                isConnected = false;
                                ws.send(JSON.stringify({ type: 'disconnected' }));
                                console.log('SSH shell closed');
                            });
                            
                            // Store the stream for writing data
                            sshClient.shellStream = stream;
                        });
                    })
                    .on('close', () => {
                        isConnected = false;
                        ws.send(JSON.stringify({ type: 'disconnected' }));
                        console.log('SSH connection closed');
                    })
                    .on('error', (err) => {
                        isConnected = false;
                        ws.send(JSON.stringify({ type: 'error', message: err.message }));
                        console.error('SSH error:', err);
                    });
                    
                    sshClient.connect({
                        host: host,
                        port: port,
                        username: username,
                        password: password,
                        tryKeyboard: true
                    });
                    
                } else if (data.type === 'data') {
                    // Forward data to SSH server
                    if (isConnected && sshClient && sshClient.shellStream) {
                        // Debug: Log the received data
                        console.log('Received data:', JSON.stringify(data.data));
                        
                        // Handle backspace (delete previous character)
                        if (data.data === '\b' || data.data === '\x7f') {
                            if (commandBuffer.length > 0) {
                                commandBuffer = commandBuffer.slice(0, -1);
                            }
                        }
                        // Handle carriage return or newline (end of command)
                        else if (data.data.includes('\n') || data.data.includes('\r')) {
                            console.log('End of command detected, buffer:', JSON.stringify(commandBuffer));
                            if (commandBuffer.trim()) {  // Only log non-empty commands
                                console.log('Logging command:', commandBuffer.trim());
                                logCommand(connectionInfo.host, connectionInfo.username, commandBuffer.trim());
                            }
                            commandBuffer = ''; // Clear buffer for next command
                        }
                        // Regular character, add to buffer
                        else {
                            commandBuffer += data.data;
                        }
                        
                        sshClient.shellStream.write(data.data);
                    }
                }
                
            } catch (error) {
                console.error('Error processing SSH WebSocket message:', error);
                ws.send(JSON.stringify({ type: 'error', message: 'Invalid message format' }));
            }
        });
        
        // Handle WebSocket close
        ws.on('close', () => {
            if (sshClient) {
                sshClient.end();
                sshClient = null;
            }
            isConnected = false;
            console.log('SSH WebSocket connection closed');
        });
        
        // Handle WebSocket error
        ws.on('error', (err) => {
            console.error('SSH WebSocket error:', err);
            if (sshClient) {
                sshClient.end();
                sshClient = null;
            }
            isConnected = false;
        });
    });

    console.log('SSH WebSocket server created on path /ws/ssh');
    return wss;
}

module.exports = { createSSHWebSocketServer };