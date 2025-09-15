package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🚀 Testing Database Operations via SSH Tunnel using Golang")
	fmt.Println("📊 This will test all database operations through the tunnel")
	fmt.Println("")

	// Database connection parameters
	// Using tunnel port 3307 instead of direct MySQL port 3306
	dsn := "root:rootpassword@tcp(localhost:3307)/db?parseTime=true"

	fmt.Printf("Connecting to MySQL via tunnel: %s\n", dsn)

	// Connect to database
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("❌ Error opening database: %v", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("❌ Error connecting to database: %v", err)
	}

	fmt.Println("✅ Connected to database successfully!")
	fmt.Println("")

	// 1. DDL Operations (Data Definition Language)
	fmt.Println("📋 Testing DDL Operations...")
	testDDLOperations(db)

	// 2. DML Operations (Data Manipulation Language)
	fmt.Println("📋 Testing DML Operations...")
	testDMLOperations(db)

	// 3. Transaction Operations
	fmt.Println("📋 Testing Transaction Operations...")
	testTransactionOperations(db)

	// 4. Administrative Operations
	fmt.Println("📋 Testing Administrative Operations...")
	testAdministrativeOperations(db)

	// 5. Cleanup
	fmt.Println("📋 Cleaning up...")
	testCleanupOperations(db)

	fmt.Println("")
	fmt.Println("✅ All database operations completed!")
	fmt.Println("📊 Check logs/AGENT-*.log for comprehensive query logging")
}

func testDDLOperations(db *sql.DB) {
	fmt.Println("  🔧 Creating test table...")

	// CREATE TABLE operation
	createTableSQL := `CREATE TABLE IF NOT EXISTS test_users (
		id INT PRIMARY KEY AUTO_INCREMENT,
		username VARCHAR(50) NOT NULL UNIQUE,
		email VARCHAR(100) NOT NULL,
		age INT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Printf("⚠️  Error creating table: %v", err)
	} else {
		fmt.Println("    ✅ Table 'test_users' created")
	}

	// ALTER TABLE operation
	fmt.Println("  🔧 Altering table structure...")
	alterTableSQL := "ALTER TABLE test_users ADD COLUMN last_login TIMESTAMP NULL"
	_, err = db.Exec(alterTableSQL)
	if err != nil {
		log.Printf("⚠️  Error altering table: %v", err)
	} else {
		fmt.Println("    ✅ Column 'last_login' added")
	}

	// CREATE INDEX operation
	fmt.Println("  🔧 Creating index...")
	createIndexSQL := "CREATE INDEX idx_username ON test_users(username)"
	_, err = db.Exec(createIndexSQL)
	if err != nil {
		log.Printf("⚠️  Error creating index: %v", err)
	} else {
		fmt.Println("    ✅ Index 'idx_username' created")
	}

	time.Sleep(1 * time.Second)
}

func testDMLOperations(db *sql.DB) {
	// INSERT operations
	fmt.Println("  📝 Inserting test data...")

	users := []map[string]interface{}{
		{"username": "john_doe", "email": "john@example.com", "age": 30},
		{"username": "jane_smith", "email": "jane@example.com", "age": 25},
		{"username": "bob_wilson", "email": "bob@example.com", "age": 35},
	}

	for _, user := range users {
		insertSQL := "INSERT INTO test_users (username, email, age) VALUES (?, ?, ?)"
		result, err := db.Exec(insertSQL, user["username"], user["email"], user["age"])
		if err != nil {
			log.Printf("⚠️  Error inserting user %s: %v", user["username"], err)
		} else {
			id, _ := result.LastInsertId()
			fmt.Printf("    ✅ User '%s' inserted with ID: %d\n", user["username"], id)
		}
	}

	// SELECT operations
	fmt.Println("  🔍 Selecting data...")

	// Simple SELECT
	selectSQL := "SELECT id, username, email, age FROM test_users WHERE age > ?"
	rows, err := db.Query(selectSQL, 25)
	if err != nil {
		log.Printf("⚠️  Error selecting data: %v", err)
	} else {
		fmt.Println("    📊 Users with age > 25:")
		for rows.Next() {
			var id int
			var username, email string
			var age int
			err := rows.Scan(&id, &username, &email, &age)
			if err != nil {
				log.Printf("⚠️  Error scanning row: %v", err)
				continue
			}
			fmt.Printf("      ID: %d, Username: %s, Email: %s, Age: %d\n", id, username, email, age)
		}
		rows.Close()
	}

	// UPDATE operations
	fmt.Println("  ✏️  Updating data...")
	updateSQL := "UPDATE test_users SET last_login = NOW(), age = ? WHERE username = ?"
	result, err := db.Exec(updateSQL, 31, "john_doe")
	if err != nil {
		log.Printf("⚠️  Error updating user: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("    ✅ Updated user 'john_doe', rows affected: %d\n", rowsAffected)
	}

	// DELETE operations
	fmt.Println("  🗑️  Deleting data...")
	deleteSQL := "DELETE FROM test_users WHERE username = ?"
	result, err = db.Exec(deleteSQL, "bob_wilson")
	if err != nil {
		log.Printf("⚠️  Error deleting user: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("    ✅ Deleted user 'bob_wilson', rows affected: %d\n", rowsAffected)
	}

	time.Sleep(1 * time.Second)
}

func testTransactionOperations(db *sql.DB) {
	fmt.Println("  🔄 Testing transaction with COMMIT...")

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("⚠️  Error beginning transaction: %v", err)
		return
	}

	// Insert in transaction
	insertSQL := "INSERT INTO test_users (username, email, age) VALUES (?, ?, ?)"
	_, err = tx.Exec(insertSQL, "alice_cooper", "alice@example.com", 28)
	if err != nil {
		log.Printf("⚠️  Error inserting in transaction: %v", err)
		tx.Rollback()
		return
	}

	// Update in transaction
	updateSQL := "UPDATE test_users SET age = ? WHERE username = ?"
	_, err = tx.Exec(updateSQL, 26, "jane_smith")
	if err != nil {
		log.Printf("⚠️  Error updating in transaction: %v", err)
		tx.Rollback()
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		log.Printf("⚠️  Error committing transaction: %v", err)
	} else {
		fmt.Println("    ✅ Transaction committed successfully")
	}

	// Test transaction with ROLLBACK
	fmt.Println("  🔄 Testing transaction with ROLLBACK...")

	tx2, err := db.Begin()
	if err != nil {
		log.Printf("⚠️  Error beginning rollback transaction: %v", err)
		return
	}

	// Insert in transaction
	_, err = tx2.Exec(insertSQL, "temp_user", "temp@example.com", 99)
	if err != nil {
		log.Printf("⚠️  Error inserting temp user: %v", err)
		tx2.Rollback()
		return
	}

	// Rollback transaction
	err = tx2.Rollback()
	if err != nil {
		log.Printf("⚠️  Error rolling back transaction: %v", err)
	} else {
		fmt.Println("    ✅ Transaction rolled back successfully")
	}

	time.Sleep(1 * time.Second)
}

func testAdministrativeOperations(db *sql.DB) {
	// SHOW operations
	fmt.Println("  📊 Testing SHOW operations...")

	// Show tables
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		log.Printf("⚠️  Error showing tables: %v", err)
	} else {
		fmt.Println("    📋 Tables in database:")
		for rows.Next() {
			var tableName string
			rows.Scan(&tableName)
			fmt.Printf("      - %s\n", tableName)
		}
		rows.Close()
	}

	// DESCRIBE operation
	fmt.Println("  📊 Testing DESCRIBE operation...")
	rows, err = db.Query("DESCRIBE test_users")
	if err != nil {
		log.Printf("⚠️  Error describing table: %v", err)
	} else {
		fmt.Println("    📋 Structure of test_users table:")
		for rows.Next() {
			var field, fieldType, null, key, defaultVal, extra string
			rows.Scan(&field, &fieldType, &null, &key, &defaultVal, &extra)
			fmt.Printf("      %s: %s\n", field, fieldType)
		}
		rows.Close()
	}

	// EXPLAIN operation
	fmt.Println("  📊 Testing EXPLAIN operation...")
	rows, err = db.Query("EXPLAIN SELECT * FROM test_users WHERE username = 'john_doe'")
	if err != nil {
		log.Printf("⚠️  Error explaining query: %v", err)
	} else {
		fmt.Println("    📋 Query execution plan generated")
		rows.Close()
	}

	time.Sleep(1 * time.Second)
}

func testCleanupOperations(db *sql.DB) {
	// DROP INDEX
	fmt.Println("  🧹 Dropping index...")
	_, err := db.Exec("DROP INDEX idx_username ON test_users")
	if err != nil {
		log.Printf("⚠️  Error dropping index: %v", err)
	} else {
		fmt.Println("    ✅ Index dropped")
	}

	// TRUNCATE TABLE
	fmt.Println("  🧹 Truncating table...")
	_, err = db.Exec("TRUNCATE TABLE test_users")
	if err != nil {
		log.Printf("⚠️  Error truncating table: %v", err)
	} else {
		fmt.Println("    ✅ Table truncated")
	}

	// DROP TABLE
	/* 	fmt.Println("  🧹 Dropping table...")
	   	_, err = db.Exec("DROP TABLE IF EXISTS test_users")
	   	if err != nil {
	   		log.Printf("⚠️  Error dropping table: %v", err)
	   	} else {
	   		fmt.Println("    ✅ Table dropped")
	   	} */

	time.Sleep(1 * time.Second)
}
