-- Complete Database Migration for Token Authentication System
-- Run this script on your Linux server to update the database with all changes
-- Date: 2025-09-21

-- Use tunnel database (change to 'logs' if your setup uses logs database)
USE tunnel;

-- =====================================================
-- 1. Add token column to users table (if not exists)
-- =====================================================

-- Check if token column exists in users table
SET @exist_users_token := (SELECT count(*) FROM information_schema.COLUMNS 
    WHERE TABLE_SCHEMA='tunnel' AND TABLE_NAME='users' AND COLUMN_NAME='token');

SET @sqlstmt_users := IF(@exist_users_token > 0, 
    'SELECT ''Column token already exists in users table''', 
    'ALTER TABLE users ADD COLUMN token VARCHAR(255) UNIQUE');
PREPARE stmt FROM @sqlstmt_users;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Update existing users with default tokens (only if token is NULL or empty)
UPDATE users SET token = 'admin_token_2025_secure' 
WHERE username = 'admin' AND (token IS NULL OR token = '');

UPDATE users SET token = 'user_token_2025_access' 
WHERE username = 'user' AND (token IS NULL OR token = '');

-- =====================================================
-- 2. Add token column to clients table (if not exists)
-- =====================================================

-- Check if token column exists in clients table
SET @exist_clients_token := (SELECT count(*) FROM information_schema.COLUMNS 
    WHERE TABLE_SCHEMA='tunnel' AND TABLE_NAME='clients' AND COLUMN_NAME='token');

SET @sqlstmt_clients_token := IF(@exist_clients_token > 0, 
    'SELECT ''Column token already exists in clients table''', 
    'ALTER TABLE clients ADD COLUMN token VARCHAR(255) AFTER agent_id');
PREPARE stmt FROM @sqlstmt_clients_token;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- =====================================================
-- 3. Add username column to clients table (if not exists)
-- =====================================================

-- Check if username column exists in clients table
SET @exist_clients_username := (SELECT count(*) FROM information_schema.COLUMNS 
    WHERE TABLE_SCHEMA='tunnel' AND TABLE_NAME='clients' AND COLUMN_NAME='username');

SET @sqlstmt_clients_username := IF(@exist_clients_username > 0, 
    'SELECT ''Column username already exists in clients table''', 
    'ALTER TABLE clients ADD COLUMN username VARCHAR(50) AFTER token');
PREPARE stmt FROM @sqlstmt_clients_username;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- =====================================================
-- 4. Create users table if it doesn't exist
-- =====================================================

CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    token VARCHAR(255) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default users if they don't exist
INSERT IGNORE INTO users (username, password, role, token) VALUES 
    ('admin', 'admin123', 'admin', 'admin_token_2025_secure'),
    ('user', 'user123', 'user', 'user_token_2025_access');

-- =====================================================
-- 5. Update clients table structure if needed
-- =====================================================

-- Ensure clients table has all required columns
CREATE TABLE IF NOT EXISTS clients (
    id INT AUTO_INCREMENT PRIMARY KEY,
    client_id VARCHAR(100) UNIQUE NOT NULL,
    client_name VARCHAR(255),
    agent_id VARCHAR(100),
    token VARCHAR(255),
    username VARCHAR(50),
    status VARCHAR(20) DEFAULT 'connected',
    connected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_ping TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- =====================================================
-- 6. Update agents table structure if needed
-- =====================================================

-- Ensure agents table has token column
SET @exist_agents_token := (SELECT count(*) FROM information_schema.COLUMNS 
    WHERE TABLE_SCHEMA='tunnel' AND TABLE_NAME='agents' AND COLUMN_NAME='token');

SET @sqlstmt_agents := IF(@exist_agents_token > 0, 
    'SELECT ''Column token already exists in agents table''', 
    'ALTER TABLE agents ADD COLUMN token VARCHAR(255) AFTER agent_id');
PREPARE stmt FROM @sqlstmt_agents;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- =====================================================
-- 7. Create indexes for performance
-- =====================================================



-- =====================================================
-- 8. Clean up and optimize
-- =====================================================

-- Update any NULL usernames in clients table based on token
UPDATE clients c 
JOIN users u ON c.token = u.token 
SET c.username = u.username 
WHERE c.username IS NULL OR c.username = '';

-- Flush privileges
FLUSH PRIVILEGES;

-- Show final table structures for verification
SELECT 'Users table structure:' AS info;
DESCRIBE users;

SELECT 'Clients table structure:' AS info;
DESCRIBE clients;

SELECT 'Agents table structure:' AS info;
DESCRIBE agents;

-- Show sample data for verification
SELECT 'Sample users data:' AS info;
SELECT id, username, role, token, created_at FROM users LIMIT 5;

SELECT 'Sample clients data:' AS info;
SELECT id, client_id, client_name, agent_id, token, username, status FROM clients LIMIT 5;

SELECT 'Sample agents data:' AS info;
SELECT id, agent_id, status, token, connected_at FROM agents LIMIT 5;

-- Migration completed successfully
SELECT 'Database migration completed successfully!' AS result;