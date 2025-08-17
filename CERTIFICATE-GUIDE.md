# Self-Signed Certificate Guide

## Quick Setup

For quick setup with self-signed certificates:

```bash
# Linux/Mac
./generate-certs.sh

# Windows  
generate-certs.bat
```

## What Gets Created

The script creates the following files in `certs/` directory:

- `server.crt` - SSL certificate (public)
- `server.key` - Private key (keep secure)
- `server.csr` - Certificate signing request
- `server.conf` - OpenSSL configuration

## Certificate Details

- **Validity**: 365 days
- **Algorithm**: RSA 2048-bit
- **Subject**: `/C=ID/ST=Jakarta/L=Jakarta/O=Remote Tunnel/OU=IT Department/CN=sh.adisaputra.online`
- **Subject Alternative Names**:
  - `sh.adisaputra.online`
  - `*.sh.adisaputra.online`
  - `localhost`
  - `127.0.0.1`

## Using Self-Signed Certificates

### Server Side (Relay)

The relay server automatically uses certificates from `certs/` directory:

```bash
./start-relay.sh
# Uses: -cert certs/server.crt -key certs/server.key
```

### Client Side

Clients need to accept self-signed certificates:

#### curl Commands
```bash
# Use -k flag to ignore certificate warnings
curl -k https://sh.adisaputra.online:8443/health
```

#### Agent connects normally (WebSocket handles self-signed)
```bash
# Agent connects with -insecure flag
./start-agent.sh

# Or manually:
./bin/agent -id laptop-agent -relay-url wss://sh.adisaputra.online:8443/ws/agent -allow 127.0.0.1:22 -token YOUR_TOKEN -insecure
```

#### Client connects normally
```bash
# Client connects with -insecure flag  
./bin/client -L :2222 -relay-url wss://sh.adisaputra.online:8443/ws/client -agent laptop-agent -target 127.0.0.1:22 -token YOUR_TOKEN -insecure
```

#### Browsers
- Firefox: Click "Advanced" → "Accept the Risk and Continue"
- Chrome: Click "Advanced" → "Proceed to sh.adisaputra.online (unsafe)"
- Safari: Click "Show Details" → "Visit this website"

## Security Notes

### For Development/Testing ✅
- Self-signed certificates are perfect for development
- Provide same TLS encryption as CA-signed certificates
- No cost and immediate setup

### For Production ⚠️
- Consider using Let's Encrypt for production
- Self-signed certificates show browser warnings
- May not be accepted by some enterprise firewalls

## Troubleshooting

### Certificate Warnings
```bash
# Test certificate details
openssl x509 -in certs/server.crt -text -noout

# Test connection
openssl s_client -connect sh.adisaputra.online:8443 -servername sh.adisaputra.online
```

### Regenerate Certificates
```bash
# Remove old certificates
rm -rf certs/

# Generate new ones
./generate-certs.sh
```

### Windows OpenSSL Issues
If OpenSSL is not available on Windows:

1. Install from: https://slproweb.com/products/Win32OpenSSL.html
2. Or use WSL: `wsl ./generate-certs.sh`
3. Or use Git Bash (includes OpenSSL)

## Validation

Test your setup:

```bash
# Test all connections
./test-domain.sh    # Linux/Mac
test-domain.bat     # Windows

# Monitor continuously  
./monitor-connection.sh    # Linux/Mac
monitor-connection.bat     # Windows
```

The test scripts automatically handle self-signed certificates using `-k` flags and appropriate certificate validation bypassing.
