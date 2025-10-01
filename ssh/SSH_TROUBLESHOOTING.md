# SSH Troubleshooting Guide

## Error: "no common algorithm for key exchange"

Ini terjadi karena server SSH tidak mendukung algoritma yang digunakan oleh client.

### Solutions:

1. **Server SSH Configuration** (di server target):
   - Edit `/etc/ssh/sshd_config`
   - Tambahkan:
     ```
     KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org,ecdh-sha2-nistp256,ecdh-sha2-nistp384,ecdh-sha2-nistp521,diffie-hellman-group16-sha512,diffie-hellman-group14-sha256,diffie-hellman-group14-sha1
     Ciphers aes128-ctr,aes192-ctr,aes256-ctr,aes128-gcm@openssh.com,aes256-gcm@openssh.com
     MACs hmac-sha2-256-etm@openssh.com,hmac-sha2-256,hmac-sha2-512-etm@openssh.com,hmac-sha2-512
     ```
   - Restart SSH: `sudo systemctl restart sshd`

2. **Update SSH Server**:
   ```bash
   sudo apt update
   sudo apt upgrade openssh-server
   ```

3. **Check SSH Server Version**:
   ```bash
   ssh -V
   ```

4. **Manual SSH Test**:
   ```bash
   ssh -v user@host
   ```

## Auto Fallback

Aplikasi sudah dikonfigurasi dengan:
- Primary config: Modern algorithms
- Fallback config: Legacy/default algorithms
- Extended timeout untuk compatibility

## Common Fixes

1. **Ubuntu 14.04 or older**: Update to newer version
2. **CentOS 6 or older**: Update OpenSSH
3. **Custom SSH configs**: Check server sshd_config
