# SSH Tunnel System

A comprehensive SSH tunnel system inspired by Teleport, built with Go. This system provides secure remote access through WebSocket-based tunneling with advanced logging, web dashboard, and database integration.

## 🏗️ Architecture

```
┌─────────────────┐    WebSocket     ┌─────────────────┐    SSH/TCP     ┌─────────────────┐
│                 │ ◄──────────────► │                 │ ◄────────────► │                 │
│     CLIENT      │                  │      RELAY      │                │      AGENT      │
│                 │                  │                 │                │                 │
│ - Connect to    │                  │ - WebSocket Hub │                │ - SSH Server    │
│   remote server │                  │ - Web Dashboard │                │ - Local Services│
│ - Port forward  │                  │ - Database Log  │                │ - Port Forward  │
│ - Authentication│                  │ - Authentication│                │ - Monitoring    │
└─────────────────┘                  └─────────────────┘                └─────────────────┘
                                              │
                                              ▼
                                     ┌─────────────────┐
                                     │   MySQL DB      │
                                     │                 │
                                     │ - Connection    │
                                     │   Logs          │
                                     │ - Query Logs    │
                                     │ - Agent Status  │
                                     │ - Client Status │
                                     └─────────────────┘
```

## 🚀 Features

### Core Features
- **WebSocket-based tunneling** - Real-time, bidirectional communication
- **Multi-component architecture** - Relay, Agent, and Client components
- **SSH forwarding** - Secure shell access through tunnels
- **Port forwarding** - Forward local ports through secure tunnels

### Advanced Features
- **Web Dashboard** - Real-time monitoring and management interface
- **Database Logging** - All connections and queries logged to MySQL
- **Authentication System** - Admin login for dashboard access
- **Real-time Monitoring** - Live status of agents, clients, and connections
- **File Logging** - Comprehensive logging to files for debugging
- **Environment Configuration** - Easy deployment with .env configuration
- **SSH Interactive Shell** - Full remote terminal experience like PuTTY
- **SSH Command Logging** - All SSH commands logged to database and dashboard

### Dashboard Features
- **Agent Status** - View all connected agents and their status
- **Client Status** - Monitor active client connections
- **Connection Logs** - Real-time view of all tunnel activities
- **Database Query Logs** - Monitor all database queries through tunnels
- **SSH Logs** - Monitor all SSH commands and sessions in real-time
- **Administrative Interface** - Secure login and management

## 📋 Prerequisites

- **Go 1.21+** - [Download Go](https://golang.org/dl/)
- **MySQL 8.0+** - Database for logging (can use Docker)
- **Git** - For cloning the repository
- **PowerShell** - For running setup scripts (Windows)

## 🛠️ Quick Setup

### 1. Clone and Initialize
```powershell
git clone <repository-url>
cd ssh-tunnel-system
go mod tidy
```

### 2. Database Setup
```powershell
# Option A: Automated setup with Docker
.\setup-database.bat

# Option B: Manual MySQL setup
# Create database 'logs' and configure user access
```

### 3. Configuration
```powershell
# Edit .env file with your settings
notepad .env
```

Example `.env` configuration:
```env
# Database Configuration
DB_CONNECTION_STRING=root:root@tcp(localhost:3306)/logs?parseTime=true
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=logs

# Admin User for Dashboard
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123
```

### 4. Start System
```powershell
# Start all components
.\start-system.bat

# Or start relay only
.\start-relay.bat
```

## 🎯 Usage

### Starting Individual Components

#### Relay Server
```powershell
.\bin\relay.exe serve
```
- WebSocket endpoint: `ws://localhost:8080/ws`
- Dashboard: `http://localhost:8080`
- Default login: `admin` / `admin123`

#### Agent (on remote server)
```powershell
.\bin\agent.exe --server ws://relay-server:8080/ws --id agent1
```

#### Client (local machine)
```powershell
.\bin\client.exe --server ws://relay-server:8080/ws --target localhost:22
```

#### SSH Interactive Shell
```powershell
.\bin\interactive-shell.exe -relay ws://localhost:8080/ws -agent agent-01 -remote-user root -remote-host 192.168.1.100
```

### Dashboard Access

1. **Open Dashboard**: Navigate to `http://localhost:8080`
2. **Login**: Use credentials from `.env` (default: admin/admin123)
3. **Monitor**: View agents, clients, SSH logs, and database logs in real-time

### SSH Access Methods

#### Method 1: SSH Through Tunnel
Once client is connected:
```bash
ssh username@localhost -p 2222
```

#### Method 2: Interactive Shell (Recommended)
Direct interactive shell through relay:
```powershell
.\bin\interactive-shell.exe -relay ws://localhost:8080/ws -agent agent-01 -remote-user root -remote-host 192.168.1.100
```

**Features:**
- Dynamic prompt with working directory
- Real-time command execution
- Command logging to dashboard
- Full remote terminal experience
- Automatic session management

## 📁 Project Structure

```
ssh-tunnel-system/
├── cmd/
│   ├── relay/             # Relay server implementation
│   ├── agent/             # Agent implementation  
│   ├── client/            # Client implementation
│   ├── ssh-client/        # SSH tunnel client
│   └── interactive-shell/ # Interactive SSH shell
├── internal/
│   └── common/            # Shared utilities
│       ├── logger.go      # File logging
│       ├── message.go     # WebSocket messages
│       └── db_logger.go   # Database logging
├── frontend/              # Vue.js dashboard
│   ├── src/
│   │   ├── components/    # Vue components
│   │   ├── views/         # Dashboard views
│   │   └── router/        # Vue router
│   ├── Dockerfile         # Frontend container
│   └── nginx.conf         # Nginx configuration
├── docs/                  # Documentation
│   └── SSH_INTERACTIVE_SHELL.md # SSH shell guide
├── logs/                  # Log files (auto-created)
├── bin/                   # Compiled binaries (auto-created)
├── docker-compose.yml     # Docker deployment
├── Dockerfile             # Relay server container
├── .env                   # Environment configuration
├── init.sql               # Database schema
├── setup-database.bat     # Database setup script
├── start-system.bat       # System launcher
├── start-relay.bat        # Relay-only launcher
├── monitor.bat            # Monitoring and debug tool
├── load-env.bat           # Environment loader
└── README.md             # This file
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_CONNECTION_STRING` | Full MySQL connection string | `root:root@tcp(localhost:3306)/logs?parseTime=true` |
| `DB_HOST` | MySQL host | `localhost` |
| `DB_PORT` | MySQL port | `3306` |
| `DB_USER` | MySQL user | `root` |
| `DB_PASSWORD` | MySQL password | `root` |
| `DB_NAME` | MySQL database | `logs` |
| `ADMIN_USERNAME` | Dashboard admin username | `admin` |
| `ADMIN_PASSWORD` | Dashboard admin password | `admin123` |

### Component Arguments

#### Relay
```powershell
relay.exe serve [--port 8080] [--dashboard-port 8081]
```

#### Agent
```powershell
agent.exe --server ws://host:port/ws --id agent-name [--ssh-port 22]
```

#### Client  
```powershell
client.exe --server ws://host:port/ws --target host:port [--local-port 2222]
```

## 🔍 Monitoring & Debugging

### Monitor Tool
```powershell
.\monitor.bat
```

Features:
- System status overview
- Real-time log viewing
- Database connection testing
- Process management
- Network port checking

### Log Files
- **Relay**: `logs/relay.log`
- **Agent**: `logs/agent.log` 
- **Client**: `logs/client.log`

### Database Tables
- **agents** - Connected agent information
- **clients** - Connected client information  
- **connections** - Connection event logs
- **queries** - Database query logs through tunnels

## 🐳 Docker Support

### MySQL with Docker
```powershell
# Automated setup
.\setup-database.bat

# Manual setup
docker run -d --name mysql-tunnel \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=logs \
  -p 3306:3306 \
  mysql:8.0
```

### Full System with Docker Compose
```yaml
version: '3.8'
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: logs
    ports:
      - "3306:3306"
    
  relay:
    build: .
    depends_on:
      - mysql
    ports:
      - "8080:8080"
    environment:
      DB_CONNECTION_STRING: root:root@tcp(mysql:3306)/logs?parseTime=true
```

## 🔒 Security

### Dashboard Authentication
- Admin login required for dashboard access
- Configurable username/password via environment variables
- Session-based authentication

### Network Security
- WebSocket connections over secure protocols
- Connection logging and monitoring
- Agent/client authentication through relay

### Database Security
- All connections logged with timestamps
- Query logging for audit trails
- Configurable database credentials

## 🚀 Deployment

### Production Deployment

1. **Configure Environment**
   ```powershell
   # Create production .env
   cp .env .env.production
   # Edit with production settings
   ```

2. **Build for Production**
   ```powershell
   go build -ldflags="-s -w" -o bin/relay.exe ./cmd/relay
   go build -ldflags="-s -w" -o bin/agent.exe ./cmd/agent  
   go build -ldflags="-s -w" -o bin/client.exe ./cmd/client
   go build -ldflags="-s -w" -o bin/ssh-client.exe ./cmd/ssh-client
   go build -ldflags="-s -w" -o bin/interactive-shell.exe ./cmd/interactive-shell
   
   # Build frontend
   cd frontend && npm run build && cd ..
   ```

3. **Deploy Components**
   - **Relay**: Deploy on central server with public access
   - **Agent**: Deploy on target servers (behind firewalls)
   - **Client**: Run locally or on jump hosts

### High Availability

For production environments:
- Run multiple relay instances behind load balancer
- Use MySQL cluster for database reliability
- Monitor with external health checks
- Implement automatic restarts for components

## 🛠️ Development

### Building from Source
```powershell
# Install dependencies
go mod tidy

# Build all components
go build -o bin/relay.exe ./cmd/relay
go build -o bin/agent.exe ./cmd/agent
go build -o bin/client.exe ./cmd/client
go build -o bin/ssh-client.exe ./cmd/ssh-client
go build -o bin/interactive-shell.exe ./cmd/interactive-shell

# Build frontend
cd frontend
npm install
npm run build
cd ..

# Run tests
go test ./...
```

### Adding Features

1. **New WebSocket Messages**: Update `internal/common/message.go`
2. **Database Schema**: Modify database initialization in relay
3. **Dashboard Pages**: Add routes and templates to relay
4. **Logging**: Extend `internal/common/logger.go` and `db_logger.go`

## 📚 API Reference

### WebSocket API

#### Connection
```
ws://relay-host:8080/ws
```

#### Message Types
- `agent_register` - Agent registration
- `client_register` - Client registration  
- `tunnel_data` - Tunnel data transfer
- `tunnel_close` - Close tunnel connection
- `ssh_command` - SSH command execution
- `ssh_response` - SSH command response
- `shell_input` - Interactive shell input
- `shell_output` - Interactive shell output

### REST API

#### Dashboard Endpoints
- `GET /` - Dashboard home page
- `POST /login` - Admin authentication
- `GET /api/agents` - List connected agents (JSON)
- `GET /api/clients` - List connected clients (JSON)
- `GET /api/logs` - Connection logs (JSON)
- `GET /api/tunnel-logs` - Database query logs (JSON)
- `GET /api/ssh-logs` - SSH command logs (JSON)
- `POST /api/log-query` - Log database query
- `POST /api/log-ssh` - Log SSH command

## ❓ Troubleshooting

### Common Issues

**Database Connection Failed**
```powershell
# Check MySQL status
.\monitor.bat
# Select option 5 (Test Database Connection)

# Verify configuration
notepad .env
```

**WebSocket Connection Failed**
```powershell
# Check if relay is running
.\monitor.bat
# Select option 1 (Show System Status)

# Check firewall settings
netsh advfirewall firewall add rule name="SSH Tunnel" dir=in action=allow protocol=TCP localport=8080
```

**Dashboard Not Accessible**
```powershell
# Test dashboard
.\monitor.bat  
# Select option 6 (Test Web Dashboard)

# Check logs
.\monitor.bat
# Select option 2 (View Recent Logs - Relay)
```

### Debug Mode

Enable verbose logging by setting environment variable:
```powershell
$env:DEBUG = "true"
.\start-system.bat
```

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/your-repo/issues)
- **Documentation**: See this README and inline code comments
- **Debugging**: Use `monitor.bat` for system diagnostics

---

Built with ❤️ using Go, WebSockets, and MySQL
- **Session Management** dengan unique tracking
- **File-based Logging** untuk debugging dan monitoring
- **WebSocket Communication** untuk low-latency

### ✅ **Logging Capabilities** 
- **Query Extraction**: SQL queries, Redis commands, SSH patterns
- **Direction Tracking**: Client→Target vs Target→Client flow
- **Protocol Detection**: Automatic berdasarkan port target
- **Session Correlation**: Track queries per session
- **Debug Mode**: Detailed packet analysis 

## 🏗️ Arsitektur

```
[Client] ←→ [Relay Server] ←→ [Agent] ←→ [Target Database]
             ↓
         [Query Logger]
             ↓
         [Log Files]
```

### Komponen:
1. **Relay Server** - WebSocket hub untuk komunikasi
2. **Agent** - Forwarding + Database Query Logging
3. **Client** - Local port forwarding + Optional query logging

## 🚀 Quick Start

### 1. Build semua komponen
```bash
.\build.bat
```

### 2. Start sistem lengkap
```bash
# Terminal 1: Start Relay
.\bin\tunnel-relay.exe -p 8080

# Terminal 2: Start Agent dengan debug
set DEBUG=1
.\bin\tunnel-agent.exe -a my-agent -r ws://localhost:8080/ws/agent

# Terminal 3: Test database tunneling
.\quick-test-db.bat
```

### 3. Test MySQL dengan Query Logging
```bash
# Setup tunnel ke MySQL
.\bin\tunnel-client.exe -L :3307 -a my-agent -t 127.0.0.1:3306

# Connect dan jalankan queries
mysql -h localhost -P 3307 -u root -p
```

```sql
-- Queries ini akan dicatat di logs/AGENT-*.log
USE myapp;
SELECT * FROM users WHERE active = 1;
INSERT INTO logs (message, level) VALUES ('Test entry', 'INFO');
UPDATE users SET last_login = NOW() WHERE id = 1;
```

### 4. Monitor Database Queries Real-time
```bash
# Monitor semua query logs
Get-Content logs\AGENT-*.log -Wait -Tail 50 | Select-String "QUERY|COMMAND"

# Filter specific database
Get-Content logs\AGENT-*.log -Wait | Select-String "MySQL QUERY"
```

## 📊 Contoh Log Output

### MySQL Query Logging
```
2025/09/14 18:30:15 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL QUERY - Session: abc123 - SQL: SELECT * FROM users WHERE id = 1
2025/09/14 18:30:16 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL USE DATABASE - Session: abc123 - DB: myapp_production  
2025/09/14 18:30:17 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL INSERT - Session: abc123 - SQL: INSERT INTO logs (message) VALUES ('test')
```

### Redis Command Logging
```
2025/09/14 18:32:10 [AGENT-my-agent] INFO: [CLIENT->TARGET] Redis COMMAND - Session: def456 - CMD: GET user:session:12345
2025/09/14 18:32:11 [AGENT-my-agent] INFO: [CLIENT->TARGET] Redis COMMAND - Session: def456 - CMD: SET cache:key "value"
```

### PostgreSQL Query Logging
```
2025/09/14 18:31:20 [AGENT-my-agent] INFO: [CLIENT->TARGET] PostgreSQL QUERY - Session: xyz789 - SQL: SELECT name FROM customers LIMIT 10
2025/09/14 18:31:21 [AGENT-my-agent] INFO: [CLIENT->TARGET] PostgreSQL PARSE - Session: xyz789 - Statement: get_user_by_id
```

## 🔧 Konfigurasi

### Environment Variables
```bash
# Enable detailed debugging
set DEBUG=1

# Enable database query logging
set TUNNEL_DB_LOG=1

# Custom log directory
set TUNNEL_LOG_DIR=C:\tunnel-logs
```

### Target Port Mapping
- `:3306` → MySQL protocol detection
- `:5432` → PostgreSQL protocol detection  
- `:6379` → Redis protocol detection
- `:27017` → MongoDB protocol detection
- `:22` → SSH protocol detection

## 📁 File Structure

```
ssh/
├── cmd/
│   ├── relay/main.go          # WebSocket relay server
│   ├── agent/main.go          # Agent + DB Query Logger
│   └── client/main.go         # Client + Optional logging
├── internal/common/
│   ├── utils.go               # Logger utilities
│   ├── message.go             # Message structures
│   └── db_logger.go           # Database Query Logger
├── bin/                       # Compiled binaries
├── logs/                      # Log files per session
├── build.bat                  # Build script
├── test-db.bat               # Database testing script
├── quick-test-db.bat         # Quick database test
└── DATABASE_LOGGING.md       # Detailed logging docs
```

## 🎯 Use Cases

### 1. **Development & Debugging**
- Monitor semua database queries aplikasi
- Debug performance issues
- Understand query patterns

### 2. **Security Auditing**
- Track all database access
- Detect suspicious queries
- Compliance logging

### 3. **Performance Monitoring**
- Identify slow queries
- Monitor query frequency
- Database usage analysis

### 4. **Production Monitoring**
- Real-time query monitoring
- Error detection
- Capacity planning

## ⚠️ Security Notes

### Database Query Logging Security
- **Sensitive Data**: Logs mungkin berisi data sensitif
- **Password Exposure**: SQL queries bisa mengandung passwords
- **Access Control**: Proteksi log files dengan proper permissions
- **Production**: Pertimbangkan filtering untuk data sensitif

### Recommended Production Setup
```bash
# Hanya log structure, bukan data detail
set TUNNEL_LOG_STRUCTURE_ONLY=1

# Filter keywords sensitif
set TUNNEL_FILTER_PASSWORDS=1

# Limit query length
set TUNNEL_MAX_QUERY_LENGTH=100
```

## 📖 Dokumentasi Lengkap

- **[DATABASE_LOGGING.md](DATABASE_LOGGING.md)** - Detailed database logging guide
- **[build.bat](build.bat)** - Build all components
- **[test-db.bat](test-db.bat)** - Full database testing
- **[quick-test-db.bat](quick-test-db.bat)** - Quick database test

## 🚦 Testing

### Full Test Suite
```bash
# Test semua komponen
.\test.bat

# Test database logging specifically  
.\test-db.bat

# Quick database test
.\quick-test-db.bat
```

### Manual Testing Workflow
1. Start Relay → Agent → Client
2. Setup database tunnel (MySQL/Redis/PostgreSQL)
3. Execute database commands/queries
4. Monitor logs for captured queries
5. Verify session tracking dan protocol detection

## 🔄 Advanced Features

### Multiple Database Tunnels
```bash
# MySQL tunnel
.\bin\tunnel-client.exe -L :3307 -a my-agent -t db1.example.com:3306

# PostgreSQL tunnel  
.\bin\tunnel-client.exe -L :5433 -a my-agent -t db2.example.com:5432

# Redis tunnel
.\bin\tunnel-client.exe -L :6380 -a my-agent -t cache.example.com:6379
```

### Session Tracking
Setiap tunnel memiliki unique session ID untuk tracking queries:
```
Session: abc123-mysql-tunnel
Session: def456-redis-tunnel  
Session: xyz789-postgres-tunnel
```

### Protocol Detection Override
Untuk custom ports atau protocols:
```bash
# Force MySQL detection untuk custom port
.\bin\tunnel-client.exe -L :3307 -a my-agent -t db.example.com:3333 -protocol mysql
```

## 📞 Support

Sistem ini menyediakan SSH tunneling dengan comprehensive database query logging, cocok untuk development, debugging, security auditing, dan production monitoring.

## Dependencies

- `github.com/gorilla/websocket` - WebSocket communication
- `github.com/spf13/cobra` - CLI framework
- `golang.org/x/crypto` - SSH utilities

# Build semua komponen
./build.sh    # Linux/macOS
build.bat     # Windows

# Atau build manual
go build -o bin/tunnel-relay ./cmd/relay
go build -o bin/tunnel-agent ./cmd/agent
go build -o bin/tunnel-client ./cmd/client
```

## Penggunaan

### 1. Menjalankan Relay Server

```bash
# Jalankan relay server di port default (8080)
./bin/tunnel-relay

# Atau dengan port custom
./bin/tunnel-relay -p 9090

# Lihat help
./bin/tunnel-relay -h
```

### 2. Menjalankan Agent di Server Target

```bash
# Jalankan agent dengan koneksi ke relay server
./bin/tunnel-agent -r ws://RELAY_SERVER_IP:8080/ws/agent

# Dengan agent ID custom
./bin/tunnel-agent -a my-server-agent -r ws://192.168.1.100:8080/ws/agent

# Lihat help
./bin/tunnel-agent -h
```

### 3. Menjalankan Client

#### Mode Single Tunnel
```bash
# Buat tunnel dari port lokal 2222 ke SSH server melalui agent
./bin/tunnel-client -L :2222 -agent my-server-agent -target localhost:22 -relay-url ws://RELAY_SERVER_IP:8080/ws/client

# Contoh lengkap
./bin/tunnel-client -L :2222 -agent my-server-agent -target 127.0.0.1:22 -relay-url ws://192.168.1.100:8080/ws/client
```

#### Mode Interactive
```bash
# Jalankan dalam mode interactive
./bin/tunnel-client -i -relay-url ws://RELAY_SERVER_IP:8080/ws/client

# Dalam mode interactive, gunakan command:
> tunnel    # Buat tunnel baru
> list      # Lihat session aktif
> help      # Bantuan
> quit      # Keluar
```

### 4. Menggunakan SSH melalui Tunnel

Setelah tunnel dibuat, Anda dapat menggunakan SSH seperti biasa:

```bash
# SSH ke server melalui tunnel lokal
ssh user@localhost -p 2222

# Atau dengan key file
ssh -i ~/.ssh/id_rsa user@localhost -p 2222
```

## Contoh Skenario Penggunaan

### Skenario 1: SSH ke Server Remote melalui Relay

1. **Setup Relay Server** (di server relay publik):
```bash
# Server: 192.168.1.100
./bin/tunnel-relay -p 8080
```

2. **Setup Agent** (di server target):
```bash
# Server target dengan SSH di port 22
./bin/tunnel-agent -a prod-server -r ws://192.168.1.100:8080/ws/agent
```

3. **Setup Client** (di komputer lokal):
```bash
# Buat tunnel lokal port 2222 ke server target
./bin/tunnel-client -L :2222 -agent prod-server -target localhost:22 -relay-url ws://192.168.1.100:8080/ws/client
```

4. **Gunakan SSH**:
```bash
ssh user@localhost -p 2222
```

### Skenario 2: Multiple Agents dan Targets

1. **Agent di Server 1**:
```bash
./bin/tunnel-agent -a server1 -r ws://relay.example.com:8080/ws/agent
```

2. **Agent di Server 2**:
```bash
./bin/tunnel-agent -a server2 -r ws://relay.example.com:8080/ws/agent
```

3. **Client untuk Server 1**:
```bash
./bin/tunnel-client -L :2221 -agent server1 -target localhost:22 -relay-url ws://relay.example.com:8080/ws/client
```

4. **Client untuk Server 2**:
```bash
./bin/tunnel-client -L :2222 -agent server2 -target localhost:22 -relay-url ws://relay.example.com:8080/ws/client
```

### Skenario 3: Port Forwarding Aplikasi

Selain SSH, sistem ini juga dapat digunakan untuk port forwarding aplikasi lain:

```bash
# Forward database MySQL
./bin/tunnel-client -L :3307 -agent db-server -target localhost:3306 -relay-url ws://relay.example.com:8080/ws/client

# Forward web server
./bin/tunnel-client -L :8081 -agent web-server -target localhost:80 -relay-url ws://relay.example.com:8080/ws/client
```

## Konfigurasi

### Environment Variables

- `TUNNEL_RELAY_URL`: Default URL relay server
- `TUNNEL_AGENT_ID`: Default agent ID
- `TUNNEL_LOG_LEVEL`: Level logging (debug, info, error)

### File Konfigurasi

Sistem juga mendukung file konfigurasi JSON:

```json
{
  "relay_url": "ws://relay.example.com:8080",
  "agent_id": "my-agent",
  "heartbeat_interval": 30,
  "buffer_size": 32768
}
```

## Monitoring dan Status

### Health Check
```bash
# Cek status relay server
curl http://RELAY_SERVER_IP:8080/health
```

Response:
```json
{
  "agents": 2,
  "clients": 1,
  "sessions": 3
}
```

### Logs
Semua komponen menghasilkan log dengan format:
```
[COMPONENT-ID] LEVEL: message
```

Contoh:
```
[RELAY] INFO: Starting relay server on :8080
[AGENT-server1] INFO: Connected to relay server
[CLIENT-abc123] INFO: New tunnel created: :2222 -> server1:localhost:22
```

## Keamanan

### TLS/SSL Support
Untuk produksi, gunakan WebSocket Secure (WSS):

```bash
# Relay dengan TLS
./bin/tunnel-relay -p 443 -tls -cert server.crt -key server.key

# Client dengan WSS
./bin/tunnel-client -relay-url wss://relay.example.com/ws/client
```

### Authentication
Sistem mendukung token-based authentication:

```bash
# Agent dengan token
./bin/tunnel-agent -token your-secret-token

# Client dengan token
./bin/tunnel-client -token your-secret-token
```

## Troubleshooting

### Connection Issues
1. Pastikan relay server berjalan dan dapat diakses
2. Cek firewall settings
3. Verify URL format (ws:// untuk HTTP, wss:// untuk HTTPS)

### Agent Not Found
1. Pastikan agent terhubung ke relay server
2. Cek agent ID yang benar
3. Monitor logs untuk error koneksi

### Performance Issues
1. Adjust buffer size dengan flag `-buffer-size`
2. Monitor network latency
3. Cek resource usage di relay server

## Development

### Project Structure
```
ssh-tunnel/
├── cmd/
│   ├── relay/     # Relay server
│   ├── agent/     # SSH agent
│   └── client/    # Tunnel client
├── internal/
│   └── common/    # Shared utilities
├── bin/           # Built executables
├── build.sh       # Build script (Linux/macOS)
├── build.bat      # Build script (Windows)
└── README.md
```

### Contributing
1. Fork repository
2. Create feature branch
3. Add tests
4. Submit pull request

## License

MIT License - lihat file LICENSE untuk detail lengkap.