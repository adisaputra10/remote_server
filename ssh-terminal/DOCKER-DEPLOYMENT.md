# GoTeleport Docker Deployment

## Overview
This guide explains how to run the complete GoTeleport system using Docker containers.

## Components
- **MySQL Database**: Port 3306
- **PostgreSQL Database**: Port 5432  
- **GoTeleport Server**: Ports 8081, 8082
- **GoTeleport Agent**: Port 8080, 3308 (MySQL proxy), 5435 (PostgreSQL proxy)
- **Frontend**: Port 80

## Quick Start

### 1. Start All Services
```bash
start-docker.bat
```

### 2. View Logs
```bash
# View specific service logs
view-logs.bat server
view-logs.bat agent
view-logs.bat frontend

# View all logs
view-logs.bat all
```

### 3. Stop All Services
```bash
stop-docker.bat
```

## Manual Docker Commands

### Build Services
```bash
# Build all services
docker-compose build

# Build specific service
docker-compose build goteleport-server
docker-compose build goteleport-agent
docker-compose build frontend
```

### Start Services
```bash
# Start all services
docker-compose up -d

# Start specific service
docker-compose up -d mysql postgres
docker-compose up -d goteleport-server
docker-compose up -d goteleport-agent
```

### View Status
```bash
# Check running containers
docker-compose ps

# View logs
docker-compose logs -f goteleport-server
docker-compose logs -f goteleport-agent
```

## Configuration

### Environment Variables
Edit `.env` file to configure:
- Database connections
- Port mappings
- Log levels
- Agent tokens

### Volume Mounts
- `./goteleport.db` - Server database
- `./server.log` - Server logs
- `./agent-db.log` - Agent logs

## Testing Database Connections

### Using Unified Client
```bash
# Build unified client
go build -o unified-client.exe unified-client.go

# Run unified client
./unified-client.exe

# Select agent and create port forwards:
# 1. MySQL: localhost:3309 -> agent:3308
# 2. PostgreSQL: localhost:5436 -> agent:5435
```

### Test Scripts
```bash
# Test MySQL connection
go run test-mysql-real.go

# Test PostgreSQL connection  
go run test-postgres-real.go
```

## Troubleshooting

### Check Container Status
```bash
docker-compose ps
```

### View Container Logs
```bash
docker-compose logs goteleport-server
docker-compose logs goteleport-agent
```

### Restart Services
```bash
# Restart specific service
docker-compose restart goteleport-server
docker-compose restart goteleport-agent

# Restart all services
docker-compose restart
```

### Clean Reset
```bash
# Stop and remove all containers
docker-compose down

# Remove volumes (WARNING: This deletes data)
docker-compose down -v

# Rebuild from scratch
docker-compose build --no-cache
docker-compose up -d
```

## Network Architecture
All services run on the `goteleport-network` Docker network:
- Services can communicate using container names
- External access through exposed ports
- Database connections use internal Docker DNS

## Security Notes
- Default passwords in `.env` should be changed for production
- Agent tokens should be regenerated
- Consider using Docker secrets for sensitive data
- Ensure proper firewall rules for exposed ports
