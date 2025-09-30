# zconnect Dashboard Frontend

Vue.js 3 frontend untuk zconnect Dashboard dengan Vite build system.

## Features

- Dashboard untuk monitoring koneksi SSH tunnel
- Real-time log viewer untuk database queries
- Role-based authentication (admin/user)
- Responsive design dengan TailwindCSS
- RESTful API integration

## Development

### Prerequisites

- Node.js 18+
- npm atau yarn

### Install Dependencies

```bash
npm install
```

### Development Server

```bash
npm run dev
```

Server akan berjalan di `http://localhost:5173`

### Build for Production

```bash
npm run build
```

### Preview Production Build

```bash
npm run preview
```

## Docker

### Build Docker Image

```bash
docker build -t ssh-tunnel-frontend .
```

### Run Container

```bash
docker run -p 80:80 ssh-tunnel-frontend
```

### Docker Compose

Jalankan semua services (relay server, database, frontend):

```bash
docker-compose up -d
```

Akses dashboard di `http://localhost`

## Environment Variables

Copy `.env.example` ke `.env` dan sesuaikan konfigurasi:

```bash
# API Configuration
VITE_API_BASE_URL=http://localhost:8080

# App Configuration
VITE_APP_TITLE=SSH Tunnel Dashboard
VITE_APP_VERSION=1.0.0
```

## Project Structure

```
frontend/
├── src/
│   ├── components/     # Vue components
│   ├── views/         # Page components
│   ├── router/        # Vue Router config
│   ├── config/        # API configuration
│   └── App.vue        # Root component
├── public/            # Static assets
├── Dockerfile         # Docker configuration
├── nginx.conf         # Nginx configuration
└── package.json       # Dependencies
```

## API Endpoints

Frontend menggunakan API endpoints berikut:

- `GET /api/agents` - Daftar agents
- `GET /api/clients` - Daftar clients
- `GET /api/logs` - Connection logs
- `GET /api/tunnel-logs` - Database query logs
- `POST /login` - Authentication
- `POST /logout` - Logout

## Authentication

Login credentials (default):

- **Admin**: admin / admin123
- **User**: user / user123

## Technologies

- Vue.js 3
- Vite
- Vue Router
- Axios
- TailwindCSS
- Nginx (production)
- Docker