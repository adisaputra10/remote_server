# 📋 Ready-to-Use Database Tunnel Commands

## Copy & Paste Commands

### 🗄️ MySQL Database (Port 3308 → Local 3307)
```powershell
bin\universal-client.exe -L ":3307" -a "agent-linux" -t "103.41.206.153:3308" -c "mysql-db-tunnel" -n "MySQL Production Database"
```

### 🐘 PostgreSQL Database (Port 5432 → Local 5433)  
```powershell
bin\universal-client.exe -L ":5433" -a "agent-linux" -t "103.41.206.153:5432" -c "postgres-db-tunnel" -n "PostgreSQL Production Database"
```

### 🌐 Web Application (Port 80 → Local 8080)
```powershell
bin\universal-client.exe -L ":8080" -a "agent-linux" -t "103.41.206.153:80" -c "web-app-tunnel" -n "Production Web Application"
```

### 🔗 SSH Access (Port 22 → Local 2222)
```powershell
bin\universal-client.exe -L ":2222" -a "agent-linux" -t "103.41.206.153:22" -c "ssh-remote-tunnel" -n "Remote SSH Access"
```

### 📊 Admin Panel (Port 8000 → Local 8001)
```powershell
bin\universal-client.exe -L ":8001" -a "agent-linux" -t "103.41.206.153:8000" -c "admin-panel-tunnel" -n "Admin Dashboard Access"
```

## After Running
1. ✅ Check **SSH Tunnel Dashboard**
2. ✅ **CLIENT ID** column will show your specified client ID
3. ✅ **AGENT ID** will show "agent-linux"
4. ✅ You can now track which tunnel is which!

## Usage After Tunnel is Active
- **MySQL**: Connect to `localhost:3307`
- **PostgreSQL**: Connect to `localhost:5433`  
- **Web App**: Browse to `http://localhost:8080`
- **SSH**: `ssh user@localhost -p 2222`
- **Admin**: Browse to `http://localhost:8001`