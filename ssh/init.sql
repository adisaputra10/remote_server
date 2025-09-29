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
    token VARCHAR(255) UNIQUE,
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

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_by INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    status ENUM('active', 'inactive', 'archived') DEFAULT 'active',
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Project users relationship table
CREATE TABLE IF NOT EXISTS project_users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    project_id INT NOT NULL,
    user_id INT NOT NULL,
    role ENUM('admin', 'member', 'viewer') DEFAULT 'member',
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    assigned_by INT,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE KEY unique_project_user (project_id, user_id)
);

-- Agents table (if not exists)
CREATE TABLE IF NOT EXISTS agents (
    id INT AUTO_INCREMENT PRIMARY KEY,
    agent_id VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(100),
    description TEXT,
    host VARCHAR(255),
    port INT,
    status ENUM('connected', 'disconnected', 'error') DEFAULT 'disconnected',
    last_ping TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    token VARCHAR(255) UNIQUE
);

-- Project agents relationship table
CREATE TABLE IF NOT EXISTS project_agents (
    id INT AUTO_INCREMENT PRIMARY KEY,
    project_id INT NOT NULL,
    agent_id VARCHAR(100) NOT NULL,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    assigned_by INT,
    access_type ENUM('ssh', 'database', 'both') DEFAULT 'both',
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE KEY unique_project_agent (project_id, agent_id)
);

-- Insert default users with unique tokens
INSERT IGNORE INTO users (username, password, role, token) VALUES 
    ('admin', 'admin123', 'admin', 'admin_token_2025_secure'),
    ('user', 'user123', 'user', 'user_token_2025_access');

-- Insert default projects
INSERT IGNORE INTO projects (name, description, created_by) VALUES 
    ('Default Project', 'Default project for all users', 1),
    ('Development', 'Development environment project', 1),
    ('Production', 'Production environment project', 1);

-- Assign admin to all projects
INSERT IGNORE INTO project_users (project_id, user_id, role, assigned_by) VALUES 
    (1, 1, 'admin', 1),
    (2, 1, 'admin', 1),
    (3, 1, 'admin', 1);

-- Assign regular user to default project
INSERT IGNORE INTO project_users (project_id, user_id, role, assigned_by) VALUES 
    (1, 2, 'member', 1);

-- Flush privileges
FLUSH PRIVILEGES;