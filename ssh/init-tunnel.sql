-- Create tunnel database and setup user permissions
CREATE DATABASE IF NOT EXISTS tunnel;
USE tunnel;

-- Grant permissions to root user from any host (including Docker containers)
GRANT ALL PRIVILEGES ON tunnel.* TO 'root'@'%' IDENTIFIED BY 'rootpassword';
GRANT ALL PRIVILEGES ON tunnel.* TO 'root'@'localhost' IDENTIFIED BY 'rootpassword';
GRANT ALL PRIVILEGES ON tunnel.* TO 'root'@'172.17.0.1' IDENTIFIED BY 'rootpassword';

-- Create relay admin user with full permissions
CREATE USER IF NOT EXISTS 'relay_admin'@'%' IDENTIFIED BY 'relay_admin123';
GRANT ALL PRIVILEGES ON tunnel.* TO 'relay_admin'@'%';

-- Create tables for tunnel database
CREATE TABLE IF NOT EXISTS agents (
    id INT AUTO_INCREMENT PRIMARY KEY,
    agent_id VARCHAR(255) UNIQUE NOT NULL,
    status ENUM('connected', 'disconnected') DEFAULT 'disconnected',
    token VARCHAR(255),
    connected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_ping TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS clients (
    id INT AUTO_INCREMENT PRIMARY KEY,
    client_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    status ENUM('connected', 'disconnected') DEFAULT 'disconnected',
    connected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_ping TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS connection_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    agent_id VARCHAR(255),
    client_id VARCHAR(255),
    event_type VARCHAR(100),
    details TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_agent_id (agent_id),
    INDEX idx_client_id (client_id),
    INDEX idx_timestamp (timestamp)
);

CREATE TABLE IF NOT EXISTS tunnel_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    agent_id VARCHAR(255),
    client_id VARCHAR(255),
    operation VARCHAR(100),
    query_text TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_agent_id (agent_id),
    INDEX idx_client_id (client_id),
    INDEX idx_timestamp (timestamp)
);

CREATE TABLE IF NOT EXISTS ssh_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    agent_id VARCHAR(255),
    client_id VARCHAR(255),
    command TEXT,
    result TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_agent_id (agent_id),
    INDEX idx_client_id (client_id),
    INDEX idx_timestamp (timestamp)
);

-- Flush privileges to ensure changes take effect
FLUSH PRIVILEGES;

-- Show created tables
SHOW TABLES;