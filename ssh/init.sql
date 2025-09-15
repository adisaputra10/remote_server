-- Initialize database with required tables and users

-- Create additional user for relay server
CREATE USER IF NOT EXISTS 'relay_admin'@'%' IDENTIFIED BY 'relay_admin123';
GRANT ALL PRIVILEGES ON logs.* TO 'relay_admin'@'%';

-- Use the logs database
USE logs;

-- Create tables (these should match the tables created by the relay server)
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS connection_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    type VARCHAR(20) NOT NULL,
    agent_id VARCHAR(100),
    client_id VARCHAR(100),
    event VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    details TEXT
);

CREATE TABLE IF NOT EXISTS tunnel_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id VARCHAR(100),
    agent_id VARCHAR(100),
    client_id VARCHAR(100),
    direction VARCHAR(20),
    protocol VARCHAR(20),
    operation VARCHAR(100),
    table_name VARCHAR(100),
    query_text LONGTEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- SSH tunnel logs table
CREATE TABLE IF NOT EXISTS ssh_logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id VARCHAR(100),
    agent_id VARCHAR(100),
    client_id VARCHAR(100),
    direction VARCHAR(20),
    ssh_user VARCHAR(100),
    ssh_host VARCHAR(100),
    ssh_port VARCHAR(10),
    command TEXT,
    data_size INT DEFAULT 0,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default users
INSERT IGNORE INTO users (username, password, role) VALUES 
    ('admin', 'admin123', 'admin'),
    ('user', 'user123', 'user');

-- Flush privileges
FLUSH PRIVILEGES;