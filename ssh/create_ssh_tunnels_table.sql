-- Create SSH Tunnels Table
-- Run this in your MySQL database

USE tunnel;

CREATE TABLE IF NOT EXISTS ssh_tunnels (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INT DEFAULT 22,
    username VARCHAR(100) NOT NULL,
    password VARCHAR(255) NOT NULL,
    description TEXT,
    status ENUM('CONNECTED', 'DISCONNECTED') DEFAULT 'DISCONNECTED',
    ssh_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    
    INDEX idx_name (name),
    INDEX idx_host (host),
    INDEX idx_status (status)
);

-- Insert sample data
INSERT INTO ssh_tunnels (id, name, host, port, username, password, description, status, created_by) VALUES
('tunnel1', 'Production Server', '192.168.1.100', 22, 'root', 'admin123', 'Production server SSH access', 'CONNECTED', 'admin'),
('tunnel2', 'Development Server', '192.168.1.101', 22, 'ubuntu', 'dev123', 'Development server SSH access', 'DISCONNECTED', 'admin');

-- Verify the table was created
SELECT * FROM ssh_tunnels;