-- Update database with project management tables
-- Run this script in your Docker MySQL container

USE tunnel;

-- Projects table already exists, skipping creation

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

-- Insert default projects (only if they don't exist)
INSERT IGNORE INTO projects (project_name, description, created_by) VALUES 
    ('Default Project', 'Default project for all users', '1'),
    ('Development', 'Development environment project', '1'),
    ('Production', 'Production environment project', '1');

-- Assign admin user (id=1) to all projects
INSERT IGNORE INTO project_users (project_id, user_id, role, assigned_by) 
SELECT p.id, 1, 'admin', 1 
FROM projects p 
WHERE EXISTS (SELECT 1 FROM users WHERE id = 1);

-- Assign other users to default project
INSERT IGNORE INTO project_users (project_id, user_id, role, assigned_by) 
SELECT 1, u.id, 'member', 1 
FROM users u 
WHERE u.id > 1 AND u.id <= 10;

-- Show created tables
SHOW TABLES LIKE '%project%';
SHOW TABLES LIKE 'agents';

-- Show sample data
SELECT 'Projects created:' as Info;
SELECT * FROM projects;

SELECT 'Project user assignments:' as Info;
SELECT 
    p.project_name, 
    u.username, 
    pu.role,
    pu.assigned_at
FROM project_users pu
JOIN projects p ON pu.project_id = p.id
JOIN users u ON pu.user_id = u.id;

FLUSH PRIVILEGES;
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

-- Show created tables
SHOW TABLES LIKE '%project%';
SHOW TABLES LIKE 'agents';

-- Show sample data
SELECT * FROM projects;
SELECT * FROM project_users;

FLUSH PRIVILEGES;