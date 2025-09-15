# SSH Tunnel Relay Server - Docker Setup

## Prerequisites
- Docker
- Docker Compose

## Quick Start

1. **Build and start services**:
```bash
docker-compose up -d
```

2. **Check service status**:
```bash
docker-compose ps
```

3. **View logs**:
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f relay
docker-compose logs -f mysql
```

4. **Stop services**:
```bash
docker-compose down
```

## Services

### Relay Server
- **Container**: `ssh-tunnel-relay`
- **Port**: `8080`
- **Health Check**: `http://localhost:8080/health`
- **Web Dashboard**: `http://localhost:8080`
- **Default Credentials**:
  - Admin: `admin/admin123`
  - User: `user/user123`

### MySQL Database
- **Container**: `ssh-tunnel-mysql`
- **Port**: `3306`
- **Database**: `logs`
- **Credentials**:
  - Root: `root/root123`
  - App User: `tunnel_user/tunnel_pass`

### PostgreSQL Database (for testing)
- **Container**: `ssh-tunnel-postgres`
- **Port**: `5432`
- **Database**: `testdb`
- **Credentials**: `testuser/testpass`

## Environment Variables

You can customize the configuration by creating a `.env` file:

```env
# Database Configuration
DB_HOST=mysql
DB_PORT=3306
DB_USER=tunnel_user
DB_PASSWORD=tunnel_pass
DB_NAME=logs

# Admin User
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123

# Regular User
USER_USERNAME=user
USER_PASSWORD=user123

# MySQL Root Password
MYSQL_ROOT_PASSWORD=root123
```

## Development

### Build only the relay server:
```bash
docker build -t ssh-tunnel-relay .
```

### Run with custom environment:
```bash
docker-compose --env-file .env up -d
```

### Access container shell:
```bash
# Relay server
docker exec -it ssh-tunnel-relay sh

# MySQL
docker exec -it ssh-tunnel-mysql mysql -u root -p

# PostgreSQL
docker exec -it ssh-tunnel-postgres psql -U testuser -d testdb
```

## Volumes

- `mysql_data`: MySQL database files
- `postgres_data`: PostgreSQL database files  
- `relay_logs`: Relay server log files

## Network

All services run on the `tunnel-network` bridge network, allowing them to communicate using container names as hostnames.

## Troubleshooting

### Check service health:
```bash
docker-compose ps
```

### View detailed logs:
```bash
docker-compose logs --tail=100 relay
```

### Restart specific service:
```bash
docker-compose restart relay
```

### Reset all data:
```bash
docker-compose down -v
docker-compose up -d
```