# SSH Tunnel System - Dokumentasi

## Deskripsi
Sistem ini adalah implementasi lengkap untuk manajemen tunnel SSH yang terdiri dari tiga komponen utama:
- **Universal Client** (`universal-client.go`) - Client yang dapat beroperasi dalam mode tunnel atau SSH terintegrasi
- **Relay Server** (`cmd/relay/main.go`) - Server pusat yang mengelola koneksi antara client dan agent
- **Agent** (`cmd/agent/main.go`) - Agent yang berjalan di server target untuk menerima koneksi

## Arsitektur Sistem

```
┌─────────────────┐    WebSocket    ┌─────────────────┐    WebSocket    ┌─────────────────┐
│  Universal      │ ───────────────▶│  Relay Server   │◀─────────────── │     Agent       │
│  Client         │                 │   (Port 8080)   │                 │  (Target Server)│
└─────────────────┘                 └─────────────────┘                 └─────────────────┘
         │                                   │                                   │
         │                                   │                                   │
         ▼                                   ▼                                   ▼
┌─────────────────┐                 ┌─────────────────┐                 ┌─────────────────┐
│  Local Service  │                 │   Web Dashboard │                 │  SSH/DB Service │
│  (SSH/Database) │                 │   HTTP API      │                 │   (localhost)   │
└─────────────────┘                 └─────────────────┘                 └─────────────────┘
```

## Komponen Utama

### 1. Universal Client (`universal-client.go`)

**Fungsi Utama:**
- Client serbaguna yang mendukung dua mode operasi
- Mengelola koneksi WebSocket ke Relay Server
- Melakukan logging aktivitas SSH dan database

**Mode Operasi:**

#### Mode Tunnel
- Menggunakan flag `-L` untuk membuat local port forwarding
- Contoh: `-L :2222` membuat tunnel dari port lokal 2222
- Cocok untuk aplikasi yang membutuhkan koneksi database atau SSH terpisah

#### Mode SSH Terintegrasi
- Tanpa flag `-L`, langsung membuka sesi SSH interaktif
- Otomatis membuat tunnel dan koneksi SSH
- Mendukung logging real-time command dan output

**Fitur Utama:**
- **Logging Komprehensif**: Mencatat semua aktivitas SSH (INPUT, OUTPUT, ERROR)
- **Heartbeat Monitoring**: Memantau status koneksi secara berkala
- **Session Management**: Mengelola multiple session dengan ID unik
- **Authentication**: Menggunakan token untuk autentikasi
- **Auto-reconnection**: Otomatis reconnect jika koneksi terputus

**Parameter Command Line:**
```bash
# Mode Tunnel
./universal-client -c client1 -a agent1 -T <token> -L :2222 -t localhost:22

# Mode SSH Terintegrasi
./universal-client -c client1 -a agent1 -T <token> -u username -H 127.0.0.1 -p 2222
```

### 2. Relay Server (`cmd/relay/main.go`)

**Fungsi Utama:**
- Server pusat yang menghubungkan client dan agent
- Mengelola routing pesan WebSocket
- Menyediakan web dashboard dan HTTP API
- Melakukan logging ke database MySQL/PostgreSQL

**Komponen Utama:**

#### WebSocket Management
- **Agent Registration**: Menerima registrasi agent dengan token
- **Client Registration**: Menerima registrasi client dengan token  
- **Message Routing**: Meneruskan pesan antara client dan agent
- **Session Tracking**: Melacak semua session aktif

#### Database Integration
- **MySQL/PostgreSQL Support**: Menyimpan log SSH dan query database
- **Batch Logging**: Optimasi performa dengan batch insert
- **User Management**: Mengelola user dan role access
- **Query Monitoring**: Mencatat semua query database

#### Web Dashboard
- **Real-time Monitoring**: Dashboard untuk monitoring koneksi
- **Log Viewer**: Interface untuk melihat log SSH dan database
- **User Management**: Panel admin untuk mengelola user
- **Agent Status**: Status real-time semua agent

**API Endpoints:**
```
GET  /api/agents          - List semua agent
GET  /api/clients         - List semua client  
GET  /api/sessions        - List session aktif
GET  /api/ssh-logs        - SSH logs dengan pagination
GET  /api/query-logs      - Database query logs
POST /api/ssh-logs        - Submit SSH log entry
POST /api/query-logs      - Submit database query log
```

**Parameter Command Line:**
```bash
./relay -p 8080  # Menjalankan relay server di port 8080
```

### 3. Agent (`cmd/agent/main.go`)

**Fungsi Utama:**
- Berjalan di server target (remote server)
- Menerima koneksi dari client melalui relay
- Meneruskan koneksi ke service lokal (SSH, database, dll)

**Fitur Utama:**

#### Connection Forwarding
- **TCP Forwarding**: Meneruskan koneksi TCP ke service lokal
- **SSH Forwarding**: Khusus untuk koneksi SSH dengan logging
- **Database Forwarding**: Support MySQL, PostgreSQL dengan query logging
- **Multi-session**: Mendukung multiple session bersamaan

#### Security Features
- **Token Authentication**: Autentikasi menggunakan token
- **Connection Validation**: Validasi setiap koneksi masuk
- **Access Control**: Kontrol akses berdasarkan client ID

#### Monitoring & Logging
- **Heartbeat**: Mengirim heartbeat ke relay server
- **Connection Status**: Melaporkan status koneksi ke relay
- **Error Handling**: Penanganan error dengan logging detail

**Parameter Command Line:**
```bash
./agent -a agent1 -t <token> -r ws://relay-server:8080/ws/agent
```

## Protokol Komunikasi

### Message Types
```go
// Tipe pesan WebSocket
MsgTypeRegister    = "register"    // Registrasi client/agent
MsgTypeConnect     = "connect"     // Request koneksi baru
MsgTypeData        = "data"        // Transfer data
MsgTypeDisconnect  = "disconnect"  // Pemutusan koneksi
MsgTypeHeartbeat   = "heartbeat"   // Monitoring koneksi
MsgTypeError       = "error"       // Error handling
```

### Data Flow
1. **Agent Registration**: Agent mendaftar ke relay dengan token
2. **Client Connection**: Client terhubung ke relay dan request agent
3. **Session Creation**: Relay membuat session antara client-agent
4. **Data Transfer**: Data diteruskan bidirectional melalui relay
5. **Logging**: Semua aktivitas dicatat ke database
6. **Session Cleanup**: Session dibersihkan saat koneksi terputus

## Database Schema

### Tabel SSH Logs
```sql
CREATE TABLE ssh_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id VARCHAR(255),
    agent_id VARCHAR(255),
    client_id VARCHAR(255),
    username VARCHAR(255),
    direction ENUM('INPUT', 'OUTPUT', 'ERROR'),
    command TEXT,
    user VARCHAR(255),
    host VARCHAR(255),
    port VARCHAR(10),
    data TEXT,
    data_size INT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Tabel Query Logs
```sql
CREATE TABLE query_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id VARCHAR(255),
    agent_id VARCHAR(255),
    client_id VARCHAR(255),
    direction VARCHAR(50),
    protocol VARCHAR(50),
    operation VARCHAR(100),
    table_name VARCHAR(255),
    query_text TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Konfigurasi

### Environment Variables
```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=3306
DB_USER=username
DB_PASSWORD=password
DB_NAME=ssh_tunnel

# Relay Configuration
RELAY_URL=ws://localhost:8080/ws/client
LOG_LEVEL=INFO
```

### Config File (`config.json`)
```json
{
    "relay_url": "ws://localhost:8080/ws/client",
    "default_user": "admin",
    "local_port": "2222"
}
```

## Use Cases

### 1. Database Access melalui SSH Tunnel
```bash
# Terminal 1: Jalankan Agent di database server
./agent -a db-agent -t agent-token-123

# Terminal 2: Buat tunnel untuk database
./universal-client -c db-client -a db-agent -T client-token-123 -L :3306 -t localhost:3306

# Terminal 3: Koneksi ke database via tunnel
mysql -h 127.0.0.1 -P 3306 -u dbuser -p
```

### 2. SSH Access dengan Logging
```bash
# Terminal 1: Jalankan Agent di SSH server
./agent -a ssh-agent -t agent-token-456

# Terminal 2: SSH dengan logging otomatis
./universal-client -c ssh-client -a ssh-agent -T client-token-456 -u root -H 127.0.0.1
```

### 3. Multiple Agent Management
```bash
# Relay Server
./relay -p 8080

# Multiple Agents
./agent -a web-server -t token1 -r ws://relay:8080/ws/agent
./agent -a db-server -t token2 -r ws://relay:8080/ws/agent
./agent -a app-server -t token3 -r ws://relay:8080/ws/agent

# Clients connecting to different agents
./universal-client -a web-server -T client-token1 -L :80 -t localhost:80
./universal-client -a db-server -T client-token2 -L :3306 -t localhost:3306
```

## Security Considerations

### Token-based Authentication
- Setiap agent dan client memerlukan token unik
- Token divalidasi di relay server
- Implementasi role-based access control

### Data Encryption
- Semua komunikasi menggunakan WebSocket Secure (WSS) dalam production
- Data sensitif (password, command) di-encode base64
- Logging data aman disimpan di database

### Access Control
- Session isolation berdasarkan client-agent pair
- User authentication untuk web dashboard
- IP-based filtering (dapat dikonfigurasi)

## Monitoring & Troubleshooting

### Log Files
- `logs/client.log` - Log aktivitas client
- `logs/trace.log` - Trace debugging
- `logs/trace-requests.log` - HTTP request trace

### Dashboard Metrics
- Active connections count
- Data transfer statistics
- Error rate monitoring
- Session duration tracking

### Common Issues
1. **Connection Timeout**: Periksa network dan firewall
2. **Authentication Failed**: Verify token validity
3. **Agent Unreachable**: Check agent status dan connectivity
4. **Database Connection**: Verify database credentials dan accessibility

## Development

### Build Commands
```bash
# Build semua komponen
go build -o bin/universal-client.exe universal-client.go
go build -o bin/relay.exe cmd/relay/main.go  
go build -o bin/agent.exe cmd/agent/main.go
```

### Testing
```bash
# Test database connection
go run test-db-golang.go

# Test SSH connection
go run simple-ssh-client.go
```

### Dependencies
- `github.com/gorilla/websocket` - WebSocket support
- `github.com/spf13/cobra` - CLI framework
- `golang.org/x/crypto/ssh` - SSH client library
- `github.com/go-sql-driver/mysql` - MySQL driver

## Kesimpulan

Sistem ini menyediakan solusi lengkap untuk:
- **Secure SSH Tunneling** dengan logging komprehensif
- **Centralized Management** melalui relay server
- **Real-time Monitoring** dengan web dashboard
- **Database Integration** untuk audit trail
- **Scalable Architecture** untuk multiple agents/clients

Sistem ini cocok untuk environment enterprise yang membutuhkan kontrol akses ketat, monitoring aktivitas, dan audit trail untuk compliance requirements.