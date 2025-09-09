package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Printf("🧪 Testing MySQL Connection with Improved Timeouts...\n")
	
	// Wait a bit for tunnel to be ready
	fmt.Printf("⏳ Waiting 3 seconds for tunnel to stabilize...\n")
	time.Sleep(3 * time.Second)
	
	// Test connection with very long timeout
	dsn := "root:rootpassword@tcp(localhost:3308)/mysql?timeout=60s&readTimeout=60s&writeTimeout=60s&parseTime=true"
	fmt.Printf("🔌 Connecting with DSN: %s\n", dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to create connection: %v\n", err)
		return
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Minute * 3)

	fmt.Printf("📡 Attempting to ping database...\n")
	
	// Try ping with retry
	for attempt := 1; attempt <= 3; attempt++ {
		fmt.Printf("🔄 Ping attempt %d/3...\n", attempt)
		
		err = db.Ping()
		if err == nil {
			fmt.Printf("✅ Ping successful on attempt %d!\n", attempt)
			break
		}
		
		fmt.Printf("❌ Ping attempt %d failed: %v\n", attempt, err)
		if attempt < 3 {
			fmt.Printf("⏳ Waiting 5 seconds before retry...\n")
			time.Sleep(5 * time.Second)
		}
	}

	if err != nil {
		fmt.Printf("💥 All ping attempts failed. Last error: %v\n", err)
		return
	}

	// Test simple query
	fmt.Printf("🔍 Testing simple query...\n")
	var version string
	err = db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		return
	}

	fmt.Printf("✅ MySQL version: %s\n", version)
	fmt.Printf("🎉 All tests passed! MySQL tunnel is working!\n")
}
