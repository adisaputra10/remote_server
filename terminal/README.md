# Interactive Terminal with Golang

Terminal interaktif yang dibangun dengan Golang dengan fitur-fitur:

## Fitur

- **Cross-platform**: Berjalan di Windows, Linux, dan macOS
- **Command History**: Menyimpan riwayat perintah
- **Command Aliases**: Mendukung alias untuk perintah
- **Logging**: Mencatat semua aktivitas terminal
- **Built-in Commands**: Perintah bawaan yang berguna
- **Working Directory Management**: Pengelolaan direktori kerja

## Built-in Commands

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

## Default Aliases

### Windows
- `ll` → `dir`
- `ls` → `dir`
- `cat` → `type`
- `grep` → `findstr`
- `which` → `where`
- `ps` → `tasklist`

### Linux/macOS
- `ll` → `ls -la`
- `la` → `ls -a`
- `dir` → `ls`
- `type` → `cat`

### Universal
- `..` → `cd ..`
- `...` → `cd ../..`
- `h` → `history`
- `c` → `clear`
- `q` → `quit`

## Cara Menjalankan

### Build dan Run
```bash
# Build terminal
cd terminal
go build -o terminal.exe main.go

# Jalankan terminal
./terminal.exe
```

### Atau gunakan script batch (Windows)
```batch
run-interactive-terminal.bat
```

## Contoh Penggunaan

```
GoTerm [remote]> help
===============================================
Interactive Terminal Help
===============================================
Built-in Commands:
  help         - Show this help message
  exit/quit    - Exit the terminal
  clear/cls    - Clear the screen
  history      - Show command history
  cd [dir]     - Change directory
  pwd          - Show current directory
  alias        - Show/manage aliases
  env          - Show environment info
  version      - Show terminal version
  time         - Show current time
  whoami       - Show current user

GoTerm [remote]> ls
# Akan menjalankan perintah sistem sesuai OS

GoTerm [remote]> alias mycommand = echo "Hello World"
Alias set: mycommand = 'echo "Hello World"'

GoTerm [remote]> mycommand
Hello World

GoTerm [remote]> cd ..
GoTerm [repo]> pwd
D:\repo

GoTerm [repo]> history
Command History:
  1: help
  2: ls
  3: alias mycommand = echo "Hello World"
  4: mycommand
  5: cd ..
  6: pwd

GoTerm [repo]> exit
Goodbye!
```

## Log File

Semua aktivitas terminal dicatat dalam file `terminal_session.log`:

```
[2025-08-18 13:45:30] CMD: SESSION_START
[2025-08-18 13:45:30] OUT: Terminal started at 2025-08-18 13:45:30
[2025-08-18 13:45:35] CMD: help
[2025-08-18 13:45:35] OUT: Success
[2025-08-18 13:45:40] CMD: ls
[2025-08-18 13:45:40] OUT: Success
```

## Kompilasi untuk Platform Lain

```bash
# Untuk Windows
GOOS=windows GOARCH=amd64 go build -o terminal.exe main.go

# Untuk Linux
GOOS=linux GOARCH=amd64 go build -o terminal main.go

# Untuk macOS
GOOS=darwin GOARCH=amd64 go build -o terminal main.go
```

## Fitur Tambahan

- **Signal Handling**: Menangani Ctrl+C dengan graceful shutdown
- **Error Handling**: Penanganan error yang baik
- **Memory Management**: Pembatasan history untuk menghemat memori
- **Cross-platform Compatibility**: Mendukung berbagai sistem operasi
