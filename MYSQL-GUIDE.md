# MySQL/MariaDB Remote Access Guide

## Quick Start

### 1. Setup Agent (on server with MySQL/MariaDB)

Ensure your MySQL/MariaDB is running and accessible locally. First, run the agent:

```bash
# Linux
./start-agent.sh

# Windows
start-agent.bat
```

### 2. Setup Client (on your local machine)

Run the MySQL-specific tunnel script:

```bash
# Linux
./start-mysql-tunnel.sh

# Windows
start-mysql-tunnel.bat
```

### 3. Connect to MySQL/MariaDB

After the tunnel is established, you can connect using:

#### Command Line
```bash
mysql -h localhost -P 3306 -u your_username -p
```

#### MySQL Workbench
- Host: `localhost`
- Port: `3306` (or custom port you specified)
- Username: your MySQL username
- Password: your MySQL password

#### phpMyAdmin
If using phpMyAdmin, configure it to connect to `localhost:3306`

#### Programming Languages

**PHP:**
```php
$pdo = new PDO('mysql:host=localhost;port=3306;dbname=your_db', $username, $password);
```

**Python:**
```python
import mysql.connector
conn = mysql.connector.connect(
    host='localhost',
    port=3306,
    user='username',
    password='password',
    database='your_db'
)
```

**Node.js:**
```javascript
const mysql = require('mysql2');
const connection = mysql.createConnection({
    host: 'localhost',
    port: 3306,
    user: 'username',
    password: 'password',
    database: 'your_db'
});
```

## Troubleshooting

### Port Already in Use
If port 3306 is already used locally, the script will ask for a different local port. Use that port for connections.

### MySQL Access Denied
Ensure your MySQL user has proper permissions:
```sql
-- Allow connections from localhost
GRANT ALL PRIVILEGES ON *.* TO 'username'@'localhost' IDENTIFIED BY 'password';
FLUSH PRIVILEGES;
```

### Connection Refused
1. Ensure MySQL/MariaDB is running on the remote server
2. Check if MySQL is binding to localhost (127.0.0.1) or all interfaces (0.0.0.0)
3. Verify firewall settings on the remote server

### MySQL Configuration
Make sure MySQL is configured to accept local connections in `/etc/mysql/mysql.conf.d/mysqld.cnf`:
```ini
bind-address = 127.0.0.1
# or
bind-address = 0.0.0.0
```

## Security Notes

- The tunnel encrypts data between client and relay
- MySQL credentials are still required for database access
- Consider using MySQL SSL for additional security
- Use strong MySQL passwords and limit user privileges

## Multiple Database Servers

You can tunnel to different MySQL servers by specifying custom addresses:
- Local MySQL: `127.0.0.1:3306`
- Docker MySQL: `172.17.0.2:3306`
- Remote MySQL: `192.168.1.100:3306`

## Docker MySQL Example

If your MySQL runs in Docker:
```bash
# Find Docker container IP
docker inspect mysql_container | grep IPAddress

# Use that IP in the tunnel script, e.g., 172.17.0.2:3306
```
