package common

import (
    "encoding/hex"
    "regexp"
    "strings"
)

// DatabaseProtocol represents different database protocols
type DatabaseProtocol int

const (
    ProtocolUnknown DatabaseProtocol = iota
    ProtocolMySQL
    ProtocolPostgreSQL
    ProtocolMongoDB
    ProtocolRedis
    ProtocolSSH
)

// DatabaseQueryLogger handles logging of database queries
type DatabaseQueryLogger struct {
    logger   *Logger
    protocol DatabaseProtocol
    target   string
    callback func(sessionID, operation, tableName, query, protocol, direction string)
}

// NewDatabaseQueryLogger creates a new database query logger
func NewDatabaseQueryLogger(logger *Logger, target string) *DatabaseQueryLogger {
    protocol := detectProtocol(target)
    return &DatabaseQueryLogger{
        logger:   logger,
        protocol: protocol,
        target:   target,
        callback: nil,
    }
}

// SetCallback sets a callback function to be called when queries are detected
func (dql *DatabaseQueryLogger) SetCallback(callback func(sessionID, operation, tableName, query, protocol, direction string)) {
    dql.callback = callback
}

// detectProtocol determines the database protocol based on target address
func detectProtocol(target string) DatabaseProtocol {
    if strings.Contains(target, ":22") {
        return ProtocolSSH
    }
    if strings.Contains(target, ":3306") {
        return ProtocolMySQL
    }
    if strings.Contains(target, ":5432") {
        return ProtocolPostgreSQL
    }
    if strings.Contains(target, ":27017") {
        return ProtocolMongoDB
    }
    if strings.Contains(target, ":6379") {
        return ProtocolRedis
    }
    return ProtocolUnknown
}

// LogData logs and analyzes data packets for database queries
func (dql *DatabaseQueryLogger) LogData(data []byte, direction string, sessionID string) {
    if len(data) == 0 {
        return
    }

    // Log basic data transfer
    dql.logger.Debug("[%s] %s: %d bytes - Session: %s", direction, dql.getProtocolName(), len(data), sessionID)

    // Analyze based on protocol
    switch dql.protocol {
    case ProtocolMySQL:
        dql.analyzeMySQLPacket(data, direction, sessionID)
    case ProtocolPostgreSQL:
        dql.analyzePostgreSQLPacket(data, direction, sessionID)
    case ProtocolRedis:
        dql.analyzeRedisCommand(data, direction, sessionID)
    case ProtocolSSH:
        dql.analyzeSSHData(data, direction, sessionID)
    default:
        dql.analyzeGenericData(data, direction, sessionID)
    }
}

// analyzeMySQLPacket analyzes MySQL protocol packets
func (dql *DatabaseQueryLogger) analyzeMySQLPacket(data []byte, direction string, sessionID string) {
    if len(data) < 5 {
        return
    }

    // MySQL packet header: 3 bytes length + 1 byte sequence + 1 byte command
    packetLength := int(data[0]) | int(data[1])<<8 | int(data[2])<<16
    sequenceID := data[3]
    
    if len(data) < 5 {
        return
    }
    
    command := data[4]
    
    dql.logger.Debug("[%s] MySQL packet - Length: %d, Seq: %d, Cmd: 0x%02x, Session: %s", 
        direction, packetLength, sequenceID, command, sessionID)

    // MySQL command types
    switch command {
    case 0x03: // COM_QUERY
        if len(data) > 5 {
            query := string(data[5:])
            // Clean and limit query length for logging
            query = dql.cleanQuery(query)
            
            // Detect specific SQL operation types
            queryType := dql.detectSQLOperation(query)
            tableName := dql.extractTableName(query)
            
            if tableName != "" {
                dql.logger.Info("[%s] MySQL %s - Session: %s - Table: %s - SQL: %s", 
                    direction, queryType, sessionID, tableName, query)
            } else {
                dql.logger.Info("[%s] MySQL %s - Session: %s - SQL: %s", 
                    direction, queryType, sessionID, query)
            }
            
            // Call callback if set
            if dql.callback != nil {
                dql.callback(sessionID, queryType, tableName, query, "mysql", direction)
            }
        }
    case 0x01: // COM_QUIT
        dql.logger.Info("[%s] MySQL QUIT - Session: %s", direction, sessionID)
        // Call callback if set
        if dql.callback != nil {
            dql.callback(sessionID, "QUIT", "", "", "mysql", direction)
        }
    case 0x02: // COM_INIT_DB
        if len(data) > 5 {
            database := string(data[5:])
            dql.logger.Info("[%s] MySQL USE DATABASE - Session: %s - DB: %s", direction, sessionID, database)
            // Call callback if set
            if dql.callback != nil {
                dql.callback(sessionID, "USE_DATABASE", database, database, "mysql", direction)
            }
        }
    case 0x16: // COM_STMT_PREPARE
        if len(data) > 5 {
            query := string(data[5:])
            query = dql.cleanQuery(query)
            dql.logger.Info("[%s] MySQL PREPARE - Session: %s - SQL: %s", direction, sessionID, query)
            
            // Call callback if set
            if dql.callback != nil {
                _ = dql.detectSQLOperation(query) // We already detected the operation type above
                tableName := dql.extractTableName(query)
                dql.callback(sessionID, "PREPARE", tableName, query, "mysql", direction)
            }
        }
    case 0x17: // COM_STMT_EXECUTE
        dql.logger.Info("[%s] MySQL EXECUTE - Session: %s", direction, sessionID)
        // EXECUTE operations are not logged to database to keep table clean
    }
}

// analyzePostgreSQLPacket analyzes PostgreSQL protocol packets
func (dql *DatabaseQueryLogger) analyzePostgreSQLPacket(data []byte, direction string, sessionID string) {
    if len(data) < 5 {
        return
    }

    // PostgreSQL message format: 1 byte type + 4 bytes length + payload
    msgType := data[0]
    
    dql.logger.Debug("[%s] PostgreSQL packet - Type: '%c' (0x%02x), Session: %s", 
        direction, msgType, msgType, sessionID)

    switch msgType {
    case 'Q': // Simple query
        if len(data) > 5 {
            query := string(data[5:])
            query = dql.cleanQuery(query)
            
            // Detect PostgreSQL operation type and table
            queryType := dql.detectSQLOperation(query)
            tableName := dql.extractTableName(query)
            
            if tableName != "" {
                dql.logger.Info("[%s] PostgreSQL %s - Session: %s - Table: %s - SQL: %s", 
                    direction, queryType, sessionID, tableName, query)
            } else {
                dql.logger.Info("[%s] PostgreSQL %s - Session: %s - SQL: %s", 
                    direction, queryType, sessionID, query)
            }
            
            // Call callback if set
            if dql.callback != nil {
                dql.callback(sessionID, queryType, tableName, query, "postgresql", direction)
            }
        }
    case 'P': // Parse (prepared statement)
        if len(data) > 5 {
            payload := string(data[5:])
            query := dql.cleanQuery(payload)
            dql.logger.Info("[%s] PostgreSQL PARSE - Session: %s - Statement: %s", direction, sessionID, query)
            
            // Call callback if set
            if dql.callback != nil {
                _ = dql.detectSQLOperation(query) // We already detected the operation type above
                tableName := dql.extractTableName(query)
                dql.callback(sessionID, "PREPARE", tableName, query, "postgresql", direction)
            }
        }
    case 'E': // Execute
        dql.logger.Info("[%s] PostgreSQL EXECUTE - Session: %s", direction, sessionID)
        // EXECUTE operations are not logged to database to keep table clean
    case 'X': // Terminate
        dql.logger.Info("[%s] PostgreSQL TERMINATE - Session: %s", direction, sessionID)
        // Call callback if set
        if dql.callback != nil {
            dql.callback(sessionID, "TERMINATE", "", "", "postgresql", direction)
        }
    }
}

// analyzeRedisCommand analyzes Redis protocol commands
func (dql *DatabaseQueryLogger) analyzeRedisCommand(data []byte, direction string, sessionID string) {
    dataStr := string(data)
    
    // Redis RESP protocol
    if strings.HasPrefix(dataStr, "*") {
        // Array command
        lines := strings.Split(dataStr, "\r\n")
        if len(lines) >= 4 {
            command := strings.ToUpper(lines[2])
            commandType := dql.getRedisCommandType(command)
            
            dql.logger.Info("[%s] Redis %s - Session: %s - CMD: %s", direction, commandType, sessionID, command)
            
            // Log key if available
            if len(lines) >= 6 && lines[4] != "" {
                key := lines[4]
                if len(key) > 50 {
                    key = key[:50] + "..."
                }
                dql.logger.Debug("[%s] Redis KEY - Session: %s - Key: %s", direction, sessionID, key)
            }
            
            // Log value for SET commands
            if command == "SET" && len(lines) >= 8 && lines[6] != "" {
                value := lines[6]
                if len(value) > 100 {
                    value = value[:100] + "..."
                }
                dql.logger.Debug("[%s] Redis VALUE - Session: %s - Value: %s", direction, sessionID, value)
            }
        }
    } else if strings.HasPrefix(dataStr, "+") || strings.HasPrefix(dataStr, "-") || strings.HasPrefix(dataStr, ":") {
        // Response
        response := strings.TrimSpace(dataStr)
        if len(response) > 100 {
            response = response[:100] + "..."
        }
        dql.logger.Debug("[%s] Redis RESPONSE - Session: %s - Data: %s", direction, sessionID, response)
    }
}

// analyzeSSHData analyzes SSH protocol data
func (dql *DatabaseQueryLogger) analyzeSSHData(data []byte, direction string, sessionID string) {
    // Look for common SSH patterns
    dataStr := string(data)
    
    // SSH command patterns
    if strings.Contains(dataStr, "mysql") {
        dql.logger.Info("[%s] SSH MySQL command detected - Session: %s", direction, sessionID)
    } else if strings.Contains(dataStr, "psql") {
        dql.logger.Info("[%s] SSH PostgreSQL command detected - Session: %s", direction, sessionID)
    } else if strings.Contains(dataStr, "redis-cli") {
        dql.logger.Info("[%s] SSH Redis command detected - Session: %s", direction, sessionID)
    } else if containsSQL(dataStr) {
        dql.logger.Info("[%s] SSH SQL pattern detected - Session: %s", direction, sessionID)
    }
}

// analyzeGenericData analyzes generic data for common patterns
func (dql *DatabaseQueryLogger) analyzeGenericData(data []byte, direction string, sessionID string) {
    dataStr := string(data)
    
    // Look for SQL patterns in any protocol
    if containsSQL(dataStr) {
        query := dql.cleanQuery(dataStr)
        dql.logger.Info("[%s] SQL PATTERN - Session: %s - Query: %s", direction, sessionID, query)
    }
    
    // Log hex dump for unknown protocols if data is small
    if len(data) <= 64 {
        hexDump := hex.EncodeToString(data)
        dql.logger.Debug("[%s] HEX DUMP - Session: %s - Data: %s", direction, sessionID, hexDump)
    }
}

// containsSQL checks if data contains SQL patterns
func containsSQL(data string) bool {
    sqlPatterns := []string{
        "SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "TRUNCATE",
        "select", "insert", "update", "delete", "create", "drop", "alter", "truncate",
    }
    
    upperData := strings.ToUpper(data)
    for _, pattern := range sqlPatterns {
        if strings.Contains(upperData, strings.ToUpper(pattern)) {
            return true
        }
    }
    return false
}

// cleanQuery cleans and limits query string for logging
func (dql *DatabaseQueryLogger) cleanQuery(query string) string {
    // Remove null bytes and control characters
    query = strings.ReplaceAll(query, "\x00", "")
    query = strings.TrimSpace(query)
    
    // Limit length
    if len(query) > 200 {
        query = query[:200] + "..."
    }
    
    // Remove multiple spaces
    re := regexp.MustCompile(`\s+`)
    query = re.ReplaceAllString(query, " ")
    
    return query
}

// detectSQLOperation detects the type of SQL operation
func (dql *DatabaseQueryLogger) detectSQLOperation(query string) string {
    query = strings.ToUpper(strings.TrimSpace(query))
    
    // Data Manipulation Language (DML)
    if strings.HasPrefix(query, "SELECT") {
        return "SELECT"
    } else if strings.HasPrefix(query, "INSERT") {
        return "INSERT"
    } else if strings.HasPrefix(query, "UPDATE") {
        return "UPDATE"
    } else if strings.HasPrefix(query, "DELETE") {
        return "DELETE"
    } else if strings.HasPrefix(query, "REPLACE") {
        return "REPLACE"
    } else if strings.HasPrefix(query, "MERGE") {
        return "MERGE"
    }
    
    // Data Definition Language (DDL)
    if strings.HasPrefix(query, "CREATE") {
        if strings.Contains(query, "TABLE") {
            return "CREATE_TABLE"
        } else if strings.Contains(query, "INDEX") {
            return "CREATE_INDEX"
        } else if strings.Contains(query, "DATABASE") {
            return "CREATE_DATABASE"
        } else if strings.Contains(query, "VIEW") {
            return "CREATE_VIEW"
        }
        return "CREATE"
    } else if strings.HasPrefix(query, "ALTER") {
        if strings.Contains(query, "TABLE") {
            return "ALTER_TABLE"
        } else if strings.Contains(query, "DATABASE") {
            return "ALTER_DATABASE"
        }
        return "ALTER"
    } else if strings.HasPrefix(query, "DROP") {
        if strings.Contains(query, "TABLE") {
            return "DROP_TABLE"
        } else if strings.Contains(query, "INDEX") {
            return "DROP_INDEX"
        } else if strings.Contains(query, "DATABASE") {
            return "DROP_DATABASE"
        } else if strings.Contains(query, "VIEW") {
            return "DROP_VIEW"
        }
        return "DROP"
    } else if strings.HasPrefix(query, "TRUNCATE") {
        return "TRUNCATE"
    }
    
    // Data Control Language (DCL)
    if strings.HasPrefix(query, "GRANT") {
        return "GRANT"
    } else if strings.HasPrefix(query, "REVOKE") {
        return "REVOKE"
    }
    
    // Transaction Control Language (TCL)
    if strings.HasPrefix(query, "BEGIN") || strings.HasPrefix(query, "START TRANSACTION") {
        return "BEGIN_TRANSACTION"
    } else if strings.HasPrefix(query, "COMMIT") {
        return "COMMIT"
    } else if strings.HasPrefix(query, "ROLLBACK") {
        return "ROLLBACK"
    } else if strings.HasPrefix(query, "SAVEPOINT") {
        return "SAVEPOINT"
    }
    
    // Database administration
    if strings.HasPrefix(query, "USE") {
        return "USE_DATABASE"
    } else if strings.HasPrefix(query, "SHOW") {
        return "SHOW"
    } else if strings.HasPrefix(query, "DESCRIBE") || strings.HasPrefix(query, "DESC") {
        return "DESCRIBE"
    } else if strings.HasPrefix(query, "EXPLAIN") {
        return "EXPLAIN"
    } else if strings.HasPrefix(query, "ANALYZE") {
        return "ANALYZE"
    } else if strings.HasPrefix(query, "OPTIMIZE") {
        return "OPTIMIZE"
    } else if strings.HasPrefix(query, "REPAIR") {
        return "REPAIR"
    } else if strings.HasPrefix(query, "CHECK") {
        return "CHECK"
    }
    
    // Stored procedures and functions
    if strings.HasPrefix(query, "CALL") {
        return "CALL_PROCEDURE"
    } else if strings.HasPrefix(query, "EXEC") || strings.HasPrefix(query, "EXECUTE") {
        return "EXECUTE"
    }
    
    // Default fallback
    return "QUERY"
}

// extractTableName extracts table name from SQL query
func (dql *DatabaseQueryLogger) extractTableName(query string) string {
    query = strings.ToUpper(strings.TrimSpace(query))
    
    // Patterns for different SQL operations
    patterns := []string{
        `SELECT.*?FROM\s+([^\s,;()]+)`,           // SELECT FROM table
        `INSERT\s+INTO\s+([^\s,;()]+)`,           // INSERT INTO table
        `UPDATE\s+([^\s,;()]+)\s+SET`,            // UPDATE table SET
        `DELETE\s+FROM\s+([^\s,;()]+)`,           // DELETE FROM table
        `REPLACE\s+INTO\s+([^\s,;()]+)`,          // REPLACE INTO table
        `CREATE\s+TABLE\s+([^\s,;()]+)`,          // CREATE TABLE table
        `ALTER\s+TABLE\s+([^\s,;()]+)`,           // ALTER TABLE table
        `DROP\s+TABLE\s+([^\s,;()]+)`,            // DROP TABLE table
        `TRUNCATE\s+TABLE\s+([^\s,;()]+)`,        // TRUNCATE TABLE table
        `DESCRIBE\s+([^\s,;()]+)`,                // DESCRIBE table
        `EXPLAIN\s+SELECT.*?FROM\s+([^\s,;()]+)`, // EXPLAIN SELECT FROM table
    }
    
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindStringSubmatch(query)
        if len(matches) > 1 {
            tableName := strings.TrimSpace(matches[1])
            // Remove backticks, quotes, and brackets
            tableName = strings.Trim(tableName, "`'\"[]")
            return tableName
        }
    }
    
    return ""
}

// getRedisCommandType categorizes Redis commands by type
func (dql *DatabaseQueryLogger) getRedisCommandType(command string) string {
    command = strings.ToUpper(command)
    
    // String operations
    if command == "GET" || command == "SET" || command == "MGET" || command == "MSET" || 
       command == "INCR" || command == "DECR" || command == "APPEND" || command == "STRLEN" ||
       command == "GETSET" || command == "SETEX" || command == "SETNX" {
        return "STRING_OP"
    }
    
    // Hash operations
    if command == "HGET" || command == "HSET" || command == "HMGET" || command == "HMSET" ||
       command == "HGETALL" || command == "HDEL" || command == "HEXISTS" || command == "HKEYS" ||
       command == "HVALS" || command == "HLEN" || command == "HINCRBY" {
        return "HASH_OP"
    }
    
    // List operations
    if command == "LPUSH" || command == "RPUSH" || command == "LPOP" || command == "RPOP" ||
       command == "LRANGE" || command == "LLEN" || command == "LINDEX" || command == "LSET" ||
       command == "LTRIM" || command == "LREM" {
        return "LIST_OP"
    }
    
    // Set operations
    if command == "SADD" || command == "SREM" || command == "SMEMBERS" || command == "SCARD" ||
       command == "SISMEMBER" || command == "SPOP" || command == "SRANDMEMBER" ||
       command == "SUNION" || command == "SINTER" || command == "SDIFF" {
        return "SET_OP"
    }
    
    // Sorted set operations
    if command == "ZADD" || command == "ZREM" || command == "ZRANGE" || command == "ZCARD" ||
       command == "ZSCORE" || command == "ZRANK" || command == "ZREVRANK" || command == "ZINCRBY" ||
       command == "ZREMRANGEBYRANK" || command == "ZREMRANGEBYSCORE" {
        return "SORTED_SET_OP"
    }
    
    // Key operations
    if command == "DEL" || command == "EXISTS" || command == "EXPIRE" || command == "TTL" ||
       command == "PERSIST" || command == "RENAME" || command == "TYPE" || command == "KEYS" ||
       command == "RANDOMKEY" || command == "DUMP" || command == "RESTORE" {
        return "KEY_OP"
    }
    
    // Transaction operations
    if command == "MULTI" || command == "EXEC" || command == "DISCARD" || command == "WATCH" ||
       command == "UNWATCH" {
        return "TRANSACTION"
    }
    
    // Connection operations
    if command == "AUTH" || command == "PING" || command == "ECHO" || command == "SELECT" ||
       command == "QUIT" {
        return "CONNECTION"
    }
    
    // Server operations
    if command == "FLUSHDB" || command == "FLUSHALL" || command == "SAVE" || command == "BGSAVE" ||
       command == "LASTSAVE" || command == "SHUTDOWN" || command == "INFO" || command == "CONFIG" ||
       command == "DBSIZE" || command == "DEBUG" || command == "MONITOR" {
        return "SERVER_OP"
    }
    
    // Pub/Sub operations
    if command == "PUBLISH" || command == "SUBSCRIBE" || command == "UNSUBSCRIBE" ||
       command == "PSUBSCRIBE" || command == "PUNSUBSCRIBE" {
        return "PUBSUB_OP"
    }
    
    return "COMMAND"
}

// getProtocolName returns the protocol name as string
func (dql *DatabaseQueryLogger) getProtocolName() string {
    switch dql.protocol {
    case ProtocolMySQL:
        return "MySQL"
    case ProtocolPostgreSQL:
        return "PostgreSQL"
    case ProtocolMongoDB:
        return "MongoDB"
    case ProtocolRedis:
        return "Redis"
    case ProtocolSSH:
        return "SSH"
    default:
        return "Unknown"
    }
}