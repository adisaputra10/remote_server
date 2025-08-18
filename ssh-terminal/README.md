# SSH Terminal with Remote Agent Support

Terminal interaktif yang dibangun dengan Golang dengan kemampuan untuk mengelola SSH remote agent dan melakukan koneksi SSH melalui relay server.

## Fitur Utama

### 1. Terminal Interaktif
- **Cross-platform**: Berjalan di Windows, Linux, dan macOS
- **Command History**: Menyimpan riwayat perintah
- **Command Aliases**: Mendukung alias untuk perintah
- **Logging**: Mencatat semua aktivitas terminal
- **Built-in Commands**: Perintah bawaan yang berguna

### 2. SSH Remote Management
- **SSH Relay Server**: Mengelola relay server untuk koneksi remote
- **SSH Agent**: Mengelola SSH agent untuk tunneling
- **PTY Sessions**: Mendukung pseudo-terminal untuk session interaktif
- **Remote Connections**: Koneksi SSH melalui tunnel

## Perintah SSH

### Setup dan Status
| Command | Description |
|---------|-------------|
| `ssh-setup` | Memeriksa dan setup lingkungan SSH |
| `ssh-status` | Menampilkan status semua layanan SSH |
| `ssh-test` | Melakukan test koneksi SSH |

### Management Layanan
| Command | Description |
|---------|-------------|
| `ssh-quick` | Quick start - menjalankan semua layanan SSH |
| `ssh-start-relay` | Memulai SSH relay server |
| `ssh-start-agent` | Memulai SSH agent |
| `ssh-stop-relay` | Menghentikan SSH relay server |
| `ssh-stop-agent` | Menghentikan SSH agent |
| `ssh-stop-all` | Menghentikan semua layanan SSH |

### Koneksi SSH
| Command | Description |
|---------|-------------|
| `ssh-connect` | Melakukan koneksi ke SSH agent |
| `ssh-pty` | Memulai SSH PTY session (sama dengan ssh-connect) |

## Built-in Commands

### Basic Commands
| Command | Description |
|---------|-------------|
| `help` | Menampilkan bantuan |
| `exit`/`quit` | Keluar dari terminal |
| `clear`/`cls` | Membersihkan layar |
| `history` | Menampilkan riwayat perintah |
| `cd [dir]` | Mengubah direktori |
| `pwd` | Menampilkan direktori saat ini |
| `alias` | Mengelola alias perintah |
| `env` | Menampilkan informasi lingkungan |
| `version` | Menampilkan versi terminal |
| `time` | Menampilkan waktu saat ini |
| `whoami` | Menampilkan informasi user |

## Quick Start Guide

### 1. Setup Environment
```
SSH-Term [remote]> ssh-setup
Setting up SSH environment...
===============================================
✅ relay.exe - FOUND
✅ agent.exe - FOUND
✅ ssh-pty.exe - FOUND
✅ server.crt - FOUND
✅ server.key - FOUND

Windows SSH Service:
  ✅ OpenSSH Server - RUNNING

✅ SSH environment setup complete!

Next steps:
  1. ssh-quick     - Start all services
  2. ssh-connect   - Connect to SSH
```

### 2. Start Services
```
SSH-Term [remote]> ssh-quick
Quick Start - Starting all SSH services...
===============================================
Starting SSH relay server...
✅ Relay server started (PID: 1234)
   Listening on: https://localhost:8080

Starting SSH agent...
✅ SSH agent started (PID: 5678)
   Connected to relay at: wss://localhost:8080/ws/agent
   Allowing connections to: 127.0.0.1:22

✅ Quick start complete!
All SSH services are running.
Use 'ssh-connect' to start SSH session.
```

### 3. Check Status
```
SSH-Term [remote]> ssh-status
SSH Services Status:
===============================================
Running Processes:
  ✅ relay.exe - RUNNING
  ✅ agent.exe - RUNNING

SSH Client: INACTIVE

Summary:
  ✅ Ready for SSH connections
```

### 4. Connect to SSH
```
SSH-Term [remote]> ssh-connect
Connecting to SSH agent...
Enter your SSH credentials when prompted
Press Ctrl+C to disconnect
===============================================

Remote Tunnel SSH Client
SSH Username: john
Password: [enter password]

Microsoft Windows [Version 10.0.19045.5131]
(c) Microsoft Corporation. All rights reserved.

john@DESKTOP-12345 C:\Users\john>
```

## Arsitektur Sistem

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   SSH Terminal  │────│   Relay Server  │────│   SSH Agent     │
│                 │    │   :8080/ws      │    │                 │
│ - ssh-connect   │    │                 │    │ - 127.0.0.1:22  │
│ - ssh-pty       │    │ - WebSocket     │    │ - Windows SSH   │
│ - Management    │    │ - TLS/SSL       │    │ - User: john    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Konfigurasi Default

- **Relay URL**: `wss://localhost:8080`
- **Token**: `demo-token`
- **Agent ID**: `demo-agent`
- **SSH Target**: `127.0.0.1:22`
- **SSH User**: `john`
- **Log File**: `ssh_terminal.log`

## Cara Menjalankan

### Menggunakan Script Batch (Windows)
```batch
.\run-ssh-terminal.bat
```

### Manual Build dan Run
```bash
cd ssh-terminal
go build -o ssh-terminal.exe main.go
.\ssh-terminal.exe
```

## Troubleshooting

### 1. Services Not Starting
```
SSH-Term [remote]> ssh-setup
```
Pastikan semua file binary dan certificate ada.

### 2. SSH Connection Failed
```
SSH-Term [remote]> ssh-test
```
Periksa status Windows SSH service dan koneksi network.

### 3. Certificate Issues
Pastikan `server.crt` dan `server.key` ada di directory yang sama dengan binary.

### 4. Port Already in Use
Jika port 8080 sudah digunakan, hentikan proses yang menggunakan port tersebut:
```bash
netstat -ano | findstr :8080
taskkill /PID <process_id> /F
```

## Log Files

Semua aktivitas dicatat dalam `ssh_terminal.log`:

```
[2025-08-18 14:30:15] CMD: SESSION_START
[2025-08-18 14:30:15] OUT: SSH Terminal started at 2025-08-18 14:30:15
[2025-08-18 14:30:20] CMD: ssh-setup
[2025-08-18 14:30:20] OUT: Success
[2025-08-18 14:30:25] CMD: ssh-quick
[2025-08-18 14:30:25] OUT: Success
[2025-08-18 14:30:30] CMD: ssh-connect
[2025-08-18 14:30:30] OUT: Success
```

## Security Notes

- Gunakan sertifikat SSL yang valid untuk production
- Ganti default token untuk keamanan
- Batasi akses network sesuai kebutuhan
- Monitor log files untuk aktivitas mencurigakan

## Development

Untuk mengembangkan lebih lanjut:

1. **Add New Commands**: Implementasikan interface `func([]string) error`
2. **Custom Aliases**: Modify `getDefaultAliases()` function
3. **Extended Logging**: Enhance `logCommand()` method
4. **Authentication**: Add custom authentication mechanisms
