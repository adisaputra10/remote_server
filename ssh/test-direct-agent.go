package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🧪 Testing Direct Connection to Agent Tunnel on port 3307...")

	// Connect directly to agent tunnel on port 3307
	dsn := "root:rootpassword@tcp(localhost:3307)/mysql?timeout=10s"
	
	fmt.Printf("🔌 Trying direct connection to agent: %s\n", dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to create connection: %v\n", err)
		return
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		fmt.Printf("❌ Failed to ping database through agent: %v\n", err)
		return
	}

	fmt.Println("✅ Successfully connected to MySQL through agent tunnel!")

	// Try a simple query
	rows, err := db.Query("SELECT VERSION()")
	if err != nil {
		fmt.Printf("❌ Failed to execute query: %v\n", err)
		return
	}
	defer rows.Close()

	if rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			fmt.Printf("❌ Failed to scan result: %v\n", err)
			return
		}
		fmt.Printf("✅ MySQL Version: %s\n", version)
	}

	fmt.Println("🎉 Direct agent tunnel test completed successfully!")
}
