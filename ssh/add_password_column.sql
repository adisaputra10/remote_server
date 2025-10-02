-- Add password column to existing ssh_tunnels table
-- Run this if you already have ssh_tunnels table without password column

USE tunnel;

-- Check if password column exists, if not add it
SELECT COLUMN_NAME 
FROM INFORMATION_SCHEMA.COLUMNS 
WHERE TABLE_SCHEMA = 'tunnel' 
AND TABLE_NAME = 'ssh_tunnels' 
AND COLUMN_NAME = 'password';

-- If password column doesn't exist, run this:
ALTER TABLE ssh_tunnels 
ADD COLUMN password VARCHAR(255) NOT NULL DEFAULT '' 
AFTER username;

-- Update existing records with empty password (you can set default passwords later)
UPDATE ssh_tunnels 
SET password = '' 
WHERE password IS NULL OR password = '';

-- Show table structure to verify
DESCRIBE ssh_tunnels;