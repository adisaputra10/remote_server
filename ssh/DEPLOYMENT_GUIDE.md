# Database Migration Instructions for Linux Server

## Overview
This migration adds token authentication system to your existing relay server database.

## Files Included
- `database_migration_complete.sql` - For servers using `tunnel` database
- `database_migration_logs.sql` - For servers using `logs` database

## Changes Applied
1. **Users Table**:
   - Added `token` column (VARCHAR(255) UNIQUE)
   - Added default tokens for admin and user accounts

2. **Clients Table**:
   - Added `username` column (VARCHAR(50))
   - Enhanced to store user information from token validation

3. **Agents Table**:
   - Added `token` column for future agent authentication

4. **Indexes**:
   - Added performance indexes on token and username columns

## Default Tokens
- **Admin Token**: `admin_token_2025_secure`
- **User Token**: `user_token_2025_access`

## Installation Instructions

### Step 1: Backup Your Database
```bash
# For tunnel database
mysqldump -u root -p tunnel > backup_tunnel_$(date +%Y%m%d_%H%M%S).sql

# For logs database  
mysqldump -u root -p logs > backup_logs_$(date +%Y%m%d_%H%M%S).sql
```

### Step 2: Run Migration
```bash
# For tunnel database
mysql -u root -p tunnel < database_migration_complete.sql

# For logs database
mysql -u root -p logs < database_migration_logs.sql
```

### Step 3: Update Relay Server Binary
1. Copy the new `relay` binary to your Linux server
2. Stop the existing relay server
3. Replace the old binary with the new one
4. Update any environment variables if needed
5. Start the new relay server

### Step 4: Update Frontend (if applicable)
1. Copy the new frontend files to your web server
2. Restart web server if needed

## Verification

After running the migration, verify the changes:

```sql
-- Check users table structure
DESCRIBE users;

-- Check clients table structure  
DESCRIBE clients;

-- Check agents table structure
DESCRIBE agents;

-- Verify token data
SELECT username, token FROM users;
```

## Usage Examples

### Client Connection with Token
```bash
# Connect as admin user
./universal-client -T admin_token_2025_secure -L :3307 -t localhost:3306 -a test1

# Connect as regular user
./universal-client -T user_token_2025_access -L :3308 -t localhost:22 -a test1
```

### Expected Behavior
- Client authentication now uses user tokens from `users` table
- Username is automatically resolved from token
- Client history preserves username even after disconnect
- Dashboard shows "History Client" instead of "Active Clients"

## Troubleshooting

### Common Issues
1. **Token validation fails**: Ensure tokens exist in users table
2. **Username not showing**: Check if username column was added to clients table
3. **Connection rejected**: Verify token matches exactly (case-sensitive)

### Debug Commands
```sql
-- Check if token exists
SELECT * FROM users WHERE token = 'admin_token_2025_secure';

-- Check client records
SELECT client_id, username, token, status FROM clients;

-- Check recent connections
SELECT * FROM clients ORDER BY connected_at DESC LIMIT 10;
```

## Rollback (if needed)
If you need to rollback:

1. Stop the relay server
2. Restore from backup:
   ```bash
   mysql -u root -p tunnel < backup_tunnel_YYYYMMDD_HHMMSS.sql
   ```
3. Use the old relay binary

## Support
- Check logs for detailed error messages
- Verify database connection settings
- Ensure all columns were added successfully