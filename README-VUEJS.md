# GoTeleport dengan Vue.js Frontend

Sistem manajemen server remote yang telah diupgrade dengan frontend Vue.js yang modern dan clean.

## 🚀 Quick Start

### Opsi 1: Startup Lengkap (Recommended)
```cmd
start-complete-system.bat
```
Script ini akan:
- Build backend Go server
- Start backend server di window terpisah  
- Install dependencies frontend Vue.js
- Start frontend development server
- Membuka kedua sistem secara bersamaan

### Opsi 2: Manual Startup

**Backend Server:**
```cmd
cd ssh-terminal
go build -o goteleport-server-db.exe goteleport-server-db.go
goteleport-server-db.exe server-config-db.json
```

**Frontend Vue.js:**
```cmd
start-vue-frontend.bat
```

## 📋 Prerequisites

- **Go 1.19+** - untuk backend server
- **Node.js 16+** - untuk frontend Vue.js
- **MySQL** - untuk database (configuration: root/rootpassword@localhost:3306/log)

## 🎯 Fitur Utama

### Backend (Go Server)
- ✅ WebSocket connections untuk agents dan clients
- ✅ MySQL database logging
- ✅ RESTful API endpoints
- ✅ CORS support untuk frontend terpisah
- ✅ Command dan access logging
- ✅ User authentication

### Frontend (Vue.js)
- 🎨 **Modern UI** dengan Element Plus
- 📊 **Dashboard** - Real-time statistics
- 📝 **Command Logs** - Filter dan export functionality  
- 📋 **Access Logs** - Monitor user activities
- 🔌 **Sessions** - Manage active connections
- 📱 **Responsive Design** - Mobile friendly
- 🔄 **Auto-refresh** - Live data updates

## 🌐 URLs

| Service | URL | Description |
|---------|-----|-------------|
| Backend API | http://localhost:8080 | Go server dengan API endpoints |
| Frontend Vue.js | http://localhost:3000 | Modern web interface |
| Original Web UI | http://localhost:8080/admin | Built-in Go HTML interface |

## 📡 API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/stats` | GET | System statistics |
| `/api/logs` | GET | Command logs dengan filtering |
| `/api/access-logs` | GET | Access logs dengan filtering |
| `/api/sessions` | GET | Active sessions |
| `/ws/client` | WebSocket | Client connections |
| `/ws/agent` | WebSocket | Agent connections |

## 🔧 Development

### Frontend Development
```cmd
cd frontend
npm install
npm run dev
```

Frontend menggunakan Vite proxy untuk API calls ke backend.

### Backend Development
```cmd
cd ssh-terminal
go build -o goteleport-server-db.exe goteleport-server-db.go
```

### Production Build
```cmd
cd frontend
npm run build
```

Built files akan ada di `frontend/dist/` yang bisa di-serve oleh web server apapun.

## 🏗️ Arsitektur

```
┌─────────────────┐    HTTP/WebSocket    ┌──────────────────┐
│   Vue.js        │ ←──────────────────→ │   Go Server      │
│   Frontend      │                      │   (Backend)      │
│   Port 3000     │                      │   Port 8080      │
└─────────────────┘                      └──────────────────┘
                                                    │
                                                    ▼
                                         ┌──────────────────┐
                                         │   MySQL          │
                                         │   Database       │
                                         │   Port 3306      │
                                         └──────────────────┘
```

## 📂 Struktur Project

```
remote_server/
├── ssh-terminal/              # Backend Go server
│   ├── goteleport-server-db.go
│   ├── server-config-db.json
│   └── ...
├── frontend/                  # Frontend Vue.js
│   ├── src/
│   │   ├── views/            # Page components
│   │   ├── services/         # API services
│   │   ├── router/           # Routing
│   │   └── App.vue
│   ├── package.json
│   └── vite.config.js
├── start-complete-system.bat  # Complete startup
├── start-vue-frontend.bat     # Frontend only
└── README.md                  # This file
```

## 🎨 UI Screenshots & Features

### Dashboard
- Real-time connection statistics
- Recent activity overview
- Quick navigation ke halaman detail

### Command Logs
- Filter berdasarkan session, client, agent, status
- Export ke CSV
- Pagination dan sorting
- Real-time updates

### Access Logs  
- Monitor login/logout activities
- Filter berdasarkan user, action, IP
- Export functionality

### Sessions
- View active connections
- Session management
- Terminate sessions

## 🔐 Authentication

Backend mendukung user authentication:
- **Default admin**: username=admin, password=admin123
- **Default user**: username=user, password=user123
- Database storage untuk users
- Role-based access (admin/user)

## 🚨 Troubleshooting

### Port Already in Use
```cmd
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

### Frontend tidak bisa connect ke backend
1. Pastikan backend server running di port 8080
2. Check CORS settings di server
3. Verify proxy configuration di vite.config.js

### Database Connection Error
1. Pastikan MySQL running
2. Check database credentials di server-config-db.json
3. Create database `log` jika belum ada

### Dependencies Issues
```cmd
# Frontend
cd frontend
rm -rf node_modules package-lock.json
npm install

# Backend  
cd ssh-terminal
go mod tidy
go build
```

## 📝 Changelog

### v2.0 - Vue.js Frontend
- ✅ Terpisah frontend Vue.js dari Go server
- ✅ Modern UI dengan Element Plus
- ✅ CORS support di backend
- ✅ Improved API responses
- ✅ Real-time updates
- ✅ Export functionality
- ✅ Responsive design

### v1.0 - Original Go Server
- ✅ Basic Go HTML interface
- ✅ WebSocket communications
- ✅ MySQL logging
- ✅ User authentication

## 🤝 Contributing

1. Frontend changes: Edit files di `frontend/src/`
2. Backend changes: Edit `ssh-terminal/goteleport-server-db.go`
3. Test both systems dengan `start-complete-system.bat`
4. Ensure CORS dan API compatibility

## 📄 License

Mengikuti license terms yang sama dengan GoTeleport project.
