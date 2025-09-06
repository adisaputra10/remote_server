# GoTeleport Unified Client - MySQL & PostgreSQL Support

## Overview
Unified client sekarang mendukung port forward untuk MySQL dan PostgreSQL dengan command yang disederhanakan.

## Quick Start Commands

### 1. List Agents
```
agents
```

### 2. Connect to Agent
```
connect <agent_id>
```

### 3. MySQL Port Forward
```
mysql <local_port>
```
Example:
```
mysql 3308
```
This creates: `localhost:3308 -> agent:3306`

### 4. PostgreSQL Port Forward
```
postgresql <local_port>
```
Example:
```
postgresql 5433
```
This creates: `localhost:5433 -> agent:5432`

### 5. Custom Port Forward
```
forward <local_port> <target_host> <target_port>
```
Example:
```
forward 8080 localhost 80
```

### 6. List Active Port Forwards
```
list
```

### 7. Stop Port Forward
```
stop <local_port>
```
Example:
```
stop 3308
```

## Complete Usage Flow

1. **Start unified client:**
   ```bash
   .\unified-client.exe client-config-clean.json
   ```

2. **List available agents:**
   ```
   command> agents
   ```

3. **Connect to agent:**
   ```
   command> connect 1862343a04e880f4
   ```

4. **Create MySQL port forward:**
   ```
   command> mysql 3308
   ```

5. **Test MySQL connection:**
   ```bash
   go run test-mysql-real.go
   ```

6. **Create PostgreSQL port forward:**
   ```
   command> postgresql 5433
   ```

7. **Test PostgreSQL connection:**
   ```bash
   go run test-postgres-real.go
   ```

8. **View active forwards:**
   ```
   command> list
   ```

9. **Stop specific forward:**
   ```
   command> stop 3308
   ```

## Available Test Scripts

- `test-mysql-real.go` - Test MySQL connection through port 3308
- `test-postgres-real.go` - Test PostgreSQL connection through port 5433

## Features

- ✅ Quick MySQL port forward with `mysql <port>`
- ✅ Quick PostgreSQL port forward with `postgresql <port>`
- ✅ Custom port forwarding with `forward <local> <host> <port>`
- ✅ Interactive shell mode with `interactive`
- ✅ Database query logging with `logs`
- ✅ Real-time agent management

## Notes

- MySQL default target port: 3306
- PostgreSQL default target port: 5432
- Commands `postgres` and `postgresql` are both supported
- Port forwards automatically use the selected agent
- All SQL commands are logged and visible in frontend
