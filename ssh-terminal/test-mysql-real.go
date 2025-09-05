package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🧪 Testing MySQL Database Proxy with Real MySQL Driver...")

	// Connect through the proxy port 3307 (agent's database proxy)
	// Try different connection options
	dsns := []string{
		"root:rootpassword@tcp(localhost:3308)/mysql?timeout=5s",
	}

	var db *sql.DB

	for _, dsn := range dsns {
		fmt.Printf("🔌 Trying connection: %s\n", dsn)

		tempDB, err := sql.Open("mysql", dsn)
		if err != nil {
			fmt.Printf("❌ Failed to create connection: %v\n", err)
			continue
		}

		// Test connection
		err = tempDB.Ping()
		if err != nil {
			fmt.Printf("❌ Failed to ping database: %v\n", err)
			tempDB.Close()
			continue
		}

		fmt.Println("✅ Connected to database successfully!")
		db = tempDB
		break
	}

	if db == nil {
		fmt.Println("❌ All connection attempts failed. Please check MySQL server and credentials.")
		fmt.Println("💡 This test requires a running MySQL server on localhost:3306")
		fmt.Println("💡 Make sure the agent's database proxy is running on port 3307")
		return
	}
	defer db.Close()

	// Execute test queries that should be logged
	testQueries := []string{
		"use log",
		"select * from users",
		"SHOW DATABASES",
	}

	for i, query := range testQueries {
		fmt.Printf("📤 Executing Query %d: %s\n", i+1, query)

		rows, err := db.Query(query)
		if err != nil {
			fmt.Printf("❌ Query failed: %v\n", err)
			continue
		}

		// Process results
		columns, _ := rows.Columns()
		fmt.Printf("✅ Query executed, columns: %v\n", columns)

		// Read first row to trigger actual data transfer
		if rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for j := range values {
				valuePtrs[j] = &values[j]
			}

			rows.Scan(valuePtrs...)
			fmt.Printf("📦 First row: %v\n", values)
		}

		rows.Close()

		// Wait a bit between queries
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\n✅ All test queries completed!")
	fmt.Println("💡 Check the agent logs and unified client for SQL command logging")
}
