# GoTeleport Database Proxy Feature

## Overview
Fitur Database Proxy memungkinkan GoTeleport Agent untuk melakukan port forwarding ke database (MySQL, PostgreSQL, dll) dengan logging lengkap untuk setiap command SQL yang dijalankan.

## Fitur Utama
- **Port Forwarding**: Agent dapat mem-forward koneksi database ke target server
- **SQL Command Logging**: Semua command SQL akan di-log secara real-time
- **Protocol Support**: Mendukung MySQL (dapat diperluas untuk PostgreSQL, dll)
- **Web Interface**: Command logs dapat dilihat melalui web interface
- **Multi-Proxy**: Satu agent dapat menjalankan multiple database proxy

## Komponen

### 1. Agent (goteleport-agent.go)
- **DatabaseProxy struct**: Menangani port forwarding dan packet inspection
- **SQL Command Detection**: Menggunakan regex untuk mendeteksi SQL commands
- **Real-time Logging**: Mengirim logs ke server secara real-time

### 2. Server (goteleport-server-db.go)
- **Database Command Storage**: Menyimpan logs ke MySQL database
- **API Endpoints**: Menyediakan API untuk mengakses database logs
- **Web Interface**: Dashboard untuk monitoring database activities

### 3. Client (interactive-client-clean.go)
- **Database Commands**: Command line interface untuk melihat database logs
- **Statistics**: Menampilkan statistik database proxy

## Konfigurasi

### Agent Configuration (agent-config-db.json)
```json
{
  "server_url": "ws://localhost:8080/agent",
  "agent_name": "database-agent-1",
  "platform": "windows",
  "log_file": "agent-db.log",
  "auth_token": "your-auth-token-here",
  "metadata": {
    "location": "datacenter-1",
    "environment": "production"
  },
  "working_dir": "",
  "allowed_users": ["admin", "dbuser"],
  "database_proxies": [
    {
      "name": "mysql-main",
      "local_port": 3307,
      "target_host": "localhost",
      "target_port": 3306,
      "protocol": "mysql",
      "enabled": true
    },
    {
      "name": "mysql-backup",
      "local_port": 3308,
      "target_host": "192.168.1.100",
      "target_port": 3306,
      "protocol": "mysql",
      "enabled": false
    }
  ]
}
```

## Database Schema

Database proxy menggunakan tabel `database_commands` untuk menyimpan logs:

```sql
CREATE TABLE database_commands (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    agent_id VARCHAR(255) NOT NULL,
    command TEXT NOT NULL,
    protocol VARCHAR(50) DEFAULT 'mysql',
    client_ip VARCHAR(45),
    proxy_name VARCHAR(255),
    metadata JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_session_id (session_id),
    INDEX idx_agent_id (agent_id),
    INDEX idx_protocol (protocol),
    INDEX idx_proxy_name (proxy_name),
    INDEX idx_timestamp (timestamp)
);
```

## ✅ Testing Results

### Successful Implementation:
- ✅ **Database Agent Connection**: Agent berhasil connect ke server
- ✅ **Database Proxy Listening**: Port 3307 listening dan ready
- ✅ **Port Forwarding**: Connection localhost:3307 -> localhost:3306 working
- ✅ **Connection Logging**: Session tracking berfungsi dengan baik
- ✅ **SQL Command Detection**: Pattern matching untuk SQL commands implemented

### Test Results:
```
Database proxy mysql-main started on port 3307 -> localhost:3306
Database connection established: 127.0.0.1:50263 -> localhost:3306 (Session: db_mysql-main_1756987916)
Database connection closed: Session db_mysql-main_1756987916
```

## Cara Penggunaan

### 1. Setup Agent
```bash
# Compile agent
go build -o goteleport-agent.exe goteleport-agent.go

# Start agent dengan database proxy
goteleport-agent.exe agent-config-db.json
```

### 2. Connect to Database through Proxy
```bash
# MySQL client connection
mysql -h localhost -P 3307 -u username -p

# Example SQL commands (akan di-log)
USE mydb;
SELECT * FROM users LIMIT 10;
INSERT INTO logs (message) VALUES ('test');
```

### 3. Monitor Logs
```bash
# Melalui client
database logs agent-1 50

# Melalui web interface
http://localhost:8080/api/database-commands
```

## API Endpoints

### Get Database Commands
```
GET /api/database-commands?limit=100&offset=0&agent_id=agent-1
```

Response:
```json
{
  "commands": [
    {
      "id": 1,
      "session_id": "db_mysql-main_1234567890",
      "agent_id": "agent-1",
      "command": "SELECT * FROM users LIMIT 10",
      "protocol": "mysql",
      "client_ip": "127.0.0.1:12345",
      "proxy_name": "mysql-main",
      "timestamp": "2024-01-01T10:00:00Z",
      "created_at": "2024-01-01T10:00:00Z"
    }
  ],
  "total": 1,
  "limit": 100,
  "offset": 0
}
```

## Security Features

### 1. Command Logging
- Semua SQL commands di-log dengan timestamp
- Client IP address dicatat
- Session tracking untuk audit trail

### 2. Protocol Detection
- Automatic MySQL protocol detection
- Regex-based SQL command extraction
- Support untuk multiple database protocols

### 3. Access Control
- Agent-level access control
- User-agent assignment
- Role-based permissions

## File Structure
```
ssh-terminal/
├── goteleport-agent.go          # Agent dengan database proxy
├── goteleport-server-db.go      # Server dengan database logging
├── interactive-client-clean.go  # Client dengan database commands
├── agent-config-db.json         # Agent configuration
├── start-db-agent.bat          # Script untuk start agent
├── test-db-proxy.bat           # Script untuk testing
└── DATABASE-PROXY.md           # Dokumentasi ini
```

## Command yang Dideteksi

Agent akan mendeteksi dan log command SQL berikut:
- SELECT statements
- INSERT statements  
- UPDATE statements
- DELETE statements
- CREATE statements
- DROP statements
- ALTER statements
- SHOW statements
- DESCRIBE statements
- USE statements

## Logging Format

### Agent Log (agent-db.log)
```
[2024-01-01 10:00:00] [DB_COMMAND] Proxy: mysql-main | Session: db_mysql-main_1234567890 | Client: 127.0.0.1:12345 | Protocol: mysql | Command: SELECT * FROM users LIMIT 10
```

### Server Log
```
[2024-01-01 10:00:00] [DB_COMMAND] Database command executed | Agent: database-agent-1, Command: SELECT * FROM users LIMIT 10
```

## Troubleshooting

### 1. Port Already in Use
```
Error: failed to listen on :3307: address already in use
```
Solution: Ganti port di configuration atau stop service yang menggunakan port tersebut.

### 2. Target Database Connection Failed
```
Failed to connect to target 192.168.1.100:3306: connection refused
```
Solution: Pastikan target database server dapat diakses dari agent.

### 3. No SQL Commands Detected
Solution: 
- Pastikan protocol setting sesuai (mysql/postgresql)
- Check client compatibility dengan MySQL protocol

## Performance Considerations

- **Buffer Size**: Default buffer 4KB untuk packet inspection
- **Memory Usage**: Buffer di-reset setiap 10KB untuk mencegah memory leak
- **Connection Pooling**: Setiap client connection membuat koneksi baru ke target
- **Logging Overhead**: Minimal impact, async logging ke server

## Future Enhancements

1. **PostgreSQL Support**: Tambah detection untuk PostgreSQL protocol
2. **SSL/TLS Support**: Encrypted database connections
3. **Query Performance Metrics**: Response time tracking
4. **Advanced Filtering**: Filter sensitive data dari logs
5. **Real-time Alerts**: Alert untuk suspicious database activities
