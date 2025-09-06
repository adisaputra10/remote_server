package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("🧪 Testing PostgreSQL Database Proxy with Real PostgreSQL Driver...")

	// Connect through the unified client port forward 5439 → agent:5435
	// Test connection through unified client port forward
	dsns := []string{
		"postgres://postgres:postgres123@localhost:5439/sqleditor?sslmode=disable",
		"host=localhost port=5439 user=postgres password=postgres123 dbname=sqleditor sslmode=disable",
	}

	var db *sql.DB

	for _, dsn := range dsns {
		fmt.Printf("🔌 Trying connection: %s\n", dsn)

		tempDB, err := sql.Open("postgres", dsn)
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
		fmt.Println("❌ All connection attempts failed. Please check PostgreSQL server and credentials.")
		fmt.Println("💡 This test requires a running PostgreSQL server on localhost:5432")
		fmt.Println("💡 Make sure the agent's database proxy is running on port 5432")
		fmt.Println("💡 Using credentials: postgresql/postgresql123 and database: sqleditor")
		return
	}
	defer db.Close()

	// Test various SQL commands
	queries := []struct {
		name  string
		query string
	}{
		{"Query 1", "SELECT current_database()"},
		{"Query 2", "SELECT * FROM information_schema.tables WHERE table_schema = 'public' LIMIT 5"},
		{"Query 3", "SELECT version()"},
		{"Query 4", "SELECT datname FROM pg_database LIMIT 5"}, // List databases using SQL instead of \l
		{"Query 5", "SELECT current_user"},
		{"Query 6", "SHOW server_version"},
	}

	for _, q := range queries {
		fmt.Printf("📤 Executing %s: %s\n", q.name, q.query)

		rows, err := db.Query(q.query)
		if err != nil {
			fmt.Printf("❌ Query failed: %v\n", err)
			continue
		}

		columns, err := rows.Columns()
		if err != nil {
			fmt.Printf("❌ Failed to get columns: %v\n", err)
			rows.Close()
			continue
		}

		fmt.Printf("✅ Query executed, columns: %v\n", columns)

		// Get first row if available
		if rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				fmt.Printf("❌ Failed to scan row: %v\n", err)
			} else {
				fmt.Printf("📦 First row: %v\n", values)
			}
		}

		rows.Close()
		time.Sleep(100 * time.Millisecond) // Small delay between queries
	}

	fmt.Println("\n✅ All test queries completed!")
	fmt.Println("💡 Check the agent logs and unified client for SQL command logging")
}
