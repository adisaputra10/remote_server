# Database Query Logging

Sistem SSH Tunnel sekarang dilengkapi dengan fitur **Database Query Logging** yang dapat mendeteksi dan mencatat query database yang melewati tunnel.

## Fitur yang Didukung

### 🗄️ **Database Protocols yang Didukung:**
- **MySQL** (Port 3306)
- **PostgreSQL** (Port 5432) 
- **Redis** (Port 6379)
- **MongoDB** (Port 27017)
- **SSH** (Port 22)
- **Generic SQL** Pattern detection

### 📊 **Informasi yang Dicatat:**
- SQL Query (SELECT, INSERT, UPDATE, DELETE, dll)
- Database commands (USE DATABASE, PREPARE, EXECUTE)
- Redis commands (GET, SET, HGET, dll)
- SSH command patterns
- Data flow direction (Client->Target, Target->Client)
- Session tracking
- Protocol detection

## Contoh Log Output

### MySQL Query Logging
```
2025/09/14 18:30:15 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL QUERY - Session: abc123 - SQL: SELECT * FROM users WHERE id = 1
2025/09/14 18:30:15 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL USE DATABASE - Session: abc123 - DB: myapp_production
2025/09/14 18:30:16 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL PREPARE - Session: abc123 - SQL: INSERT INTO logs (message, created_at) VALUES (?, ?)
2025/09/14 18:30:16 [AGENT-my-agent] INFO: [CLIENT->TARGET] MySQL EXECUTE - Session: abc123
```

### PostgreSQL Query Logging
```
2025/09/14 18:31:20 [AGENT-my-agent] INFO: [CLIENT->TARGET] PostgreSQL QUERY - Session: xyz789 - SQL: SELECT name, email FROM customers LIMIT 10
2025/09/14 18:31:21 [AGENT-my-agent] INFO: [CLIENT->TARGET] PostgreSQL PARSE - Session: xyz789 - Statement: get_user_by_id
2025/09/14 18:31:22 [AGENT-my-agent] INFO: [CLIENT->TARGET] PostgreSQL EXECUTE - Session: xyz789
```

### Redis Command Logging
```
2025/09/14 18:32:10 [AGENT-my-agent] INFO: [CLIENT->TARGET] Redis COMMAND - Session: def456 - CMD: GET
2025/09/14 18:32:10 [AGENT-my-agent] DEBUG: [CLIENT->TARGET] Redis KEY - Session: def456 - Key: user:session:12345
2025/09/14 18:32:11 [AGENT-my-agent] INFO: [CLIENT->TARGET] Redis COMMAND - Session: def456 - CMD: SET
```

### SSH Command Logging
```
2025/09/14 18:33:05 [AGENT-my-agent] INFO: [CLIENT->TARGET] SSH MySQL command detected - Session: ghi789
2025/09/14 18:33:06 [AGENT-my-agent] INFO: [CLIENT->TARGET] SSH SQL pattern detected - Session: ghi789
```

## Cara Penggunaan

### 1. Test Database Tunneling
```bash
# Jalankan test lengkap untuk semua database
.\test-db.bat
```

### 2. Manual Setup untuk MySQL
```bash
# Terminal 1: Relay
set DEBUG=1
.\bin\tunnel-relay.exe -p 8080

# Terminal 2: Agent
set DEBUG=1  
.\bin\tunnel-agent.exe -a my-agent -r ws://localhost:8080/ws/agent

# Terminal 3: MySQL Client tunnel
set DEBUG=1
.\bin\tunnel-client.exe -L :3307 -a my-agent -t 127.0.0.1:3306 -r ws://localhost:8080/ws/client

# Terminal 4: Connect dan test query
mysql -h localhost -P 3307 -u root -p
```

### 3. Test MySQL Queries
```sql
-- Queries ini akan dicatat di log
USE myapp_production;
SELECT * FROM users LIMIT 5;
INSERT INTO logs (message) VALUES ('Test log entry');
UPDATE users SET last_login = NOW() WHERE id = 1;
DELETE FROM temp_data WHERE created_at < '2025-01-01';
```

### 4. Test PostgreSQL
```bash
# Setup tunnel
.\bin\tunnel-client.exe -L :5433 -a my-agent -t 127.0.0.1:5432

# Connect dan test
psql -h localhost -p 5433 -U postgres
```

```sql
-- PostgreSQL queries yang akan dicatat
\l
SELECT version();
CREATE TABLE test_table (id SERIAL, name VARCHAR(50));
INSERT INTO test_table (name) VALUES ('test');
```

### 5. Test Redis
```bash
# Setup tunnel
.\bin\tunnel-client.exe -L :6380 -a my-agent -t 127.0.0.1:6379

# Connect dan test
redis-cli -h localhost -p 6380
```

```redis
SET user:1 "John Doe"
GET user:1
HSET user:profile:1 name "John" email "john@example.com"
HGET user:profile:1 name
```

## Monitoring Logs

### Real-time Log Monitoring
```bash
# Monitor semua logs
Get-Content logs\*.log -Wait -Tail 50

# Monitor hanya Agent logs (yang mencatat database queries)
Get-Content logs\AGENT-*.log -Wait -Tail 50

# Monitor dengan filter
Get-Content logs\AGENT-*.log -Wait -Tail 50 | Select-String "QUERY\|COMMAND"
```

### Log Analysis
```bash
# Find semua MySQL queries
Select-String "MySQL QUERY" logs\AGENT-*.log

# Find semua Redis commands  
Select-String "Redis COMMAND" logs\AGENT-*.log

# Find specific SQL pattern
Select-String "SELECT.*FROM" logs\AGENT-*.log
```

## Konfigurasi Advanced

### Environment Variables untuk Database Logging
```bash
# Enable debug untuk melihat query details
set DEBUG=1
set TUNNEL_DEBUG=1

# Untuk production, matikan debug tapi tetap catat queries
set TUNNEL_DB_LOG=1
```

### Protocol Detection Override
Database protocol dideteksi otomatis berdasarkan target port:
- `:3306` → MySQL
- `:5432` → PostgreSQL  
- `:6379` → Redis
- `:27017` → MongoDB
- `:22` → SSH

## Security Considerations

### ⚠️ **Important Security Notes:**

1. **Sensitive Data**: Query logs mungkin mengandung data sensitif
2. **Password Logging**: Password dalam query bisa tercatat
3. **Production Use**: Pertimbangkan untuk memfilter data sensitif
4. **Log Rotation**: Setup log rotation untuk production
5. **Access Control**: Batasi akses ke log files

### Recommended Production Settings
```bash
# Hanya log query structure, bukan data
set TUNNEL_LOG_STRUCTURE_ONLY=1

# Filter sensitive keywords
set TUNNEL_FILTER_PASSWORDS=1

# Limit query length in logs
set TUNNEL_MAX_QUERY_LENGTH=100
```

## Troubleshooting

### Query Tidak Tercatat
1. **Pastikan Debug Mode aktif**: `set DEBUG=1`
2. **Cek protocol detection**: Apakah port target sesuai?
3. **Monitor data flow**: Lihat debug logs untuk data transfer
4. **Binary protocol**: Beberapa client menggunakan binary protocol

### Performance Impact
- Query logging menambah minimal overhead
- Regex matching untuk SQL detection
- String manipulation untuk cleaning
- File I/O untuk logging

### Custom Protocol Support
Untuk menambah support protocol lain, edit `internal/common/db_logger.go`:

```go
// Tambah protocol baru
const (
    ProtocolCustomDB DatabaseProtocol = iota + 10
)

// Tambah detection logic
func DetectProtocol(target string) DatabaseProtocol {
    if strings.Contains(target, ":1433") {
        return ProtocolSQLServer  // Custom
    }
    // ... existing logic
}
```

## Example Use Cases

### 1. **Development Debugging**
Monitor semua query yang dibuat aplikasi untuk debugging performance issues.

### 2. **Security Auditing** 
Track semua database access melalui tunnel untuk audit security.

### 3. **Performance Monitoring**
Identifikasi slow queries dan pattern usage.

### 4. **Compliance Logging**
Catat semua database operations untuk compliance requirements.

### 5. **Learning & Analysis**
Understand bagaimana aplikasi berinteraksi dengan database.

## Log File Locations

```
logs/
├── RELAY_20250914_183000.log           # Relay server logs
├── AGENT-my-agent_20250914_183001.log  # Agent logs (DATABASE QUERIES HERE)
└── CLIENT-abc123_20250914_183002.log   # Client logs
```

**Note**: Database queries primarily dicatat di **Agent logs** karena agent yang langsung berkomunikasi dengan database server.

Fitur Database Query Logging ini memberikan visibility tinggi terhadap aktivitas database yang melewati tunnel, sangat berguna untuk debugging, monitoring, dan security auditing.