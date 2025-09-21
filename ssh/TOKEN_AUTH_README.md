# Token Authentication System

## Overview
Sistem autentikasi token telah ditambahkan ke relay server untuk mengamankan koneksi client. 

## Database Changes
Kolom `token` telah ditambahkan ke tabel `users` di database `tunnel`:

```sql
ALTER TABLE users ADD COLUMN token VARCHAR(255) UNIQUE;
```

## Default Tokens
- **Admin Token**: `admin_token_2025_secure`
- **User Token**: `user_token_2025_access`

## Usage Examples

### 1. Connect as Admin User
```bash
.\bin\universal-client.exe -T admin_token_2025_secure -c client-admin -n "Admin Client"
```

### 2. Connect as Regular User
```bash
.\bin\universal-client.exe -T user_token_2025_access -c client-user -n "User Client"
```

### 3. SSH Mode with Token
```bash
.\bin\universal-client.exe -T admin_token_2025_secure -u username -H target-server
```

### 4. Tunnel Mode with Token
```bash
.\bin\universal-client.exe -T admin_token_2025_secure -L :2222 -t localhost:22 -a agent-id
```

## Token Validation
Relay server akan memvalidasi token terhadap database sebelum memperbolehkan client terhubung:

1. Client mengirim token melalui parameter `-T` atau `--token`
2. Relay server memvalidasi token di database `tunnel.users`
3. Jika valid, client diizinkan terhubung
4. Jika tidak valid, koneksi ditolak dengan error "Invalid user token"

## Security Features
- Token disimpan di database dengan constraint UNIQUE
- Validasi token dilakukan sebelum registrasi client
- Log autentikasi dicatat di server
- Koneksi tanpa token valid akan ditolak

## Migration
Untuk database yang sudah ada, jalankan:
```bash
Get-Content migration_add_token_tunnel.sql | docker exec -i mysql8 mysql -u root -prootpassword tunnel
```

## Configuration
Relay server sekarang menggunakan database `tunnel` sebagai default (bukan `logs`).