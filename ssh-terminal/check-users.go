package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Connect to database
	db, err := sql.Open("mysql", "root:rootpassword@tcp(localhost:3306)/log?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Connected to database successfully!")

	// Check if users table exists
	var tableName string
	err = db.QueryRow("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = 'log' AND TABLE_NAME = 'users'").Scan(&tableName)
	if err != nil {
		fmt.Println("Users table does not exist:", err)
		return
	}

	fmt.Println("Users table exists:", tableName)

	// Count users
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		fmt.Println("Failed to count users:", err)
		return
	}

	fmt.Printf("Number of users in database: %d\n", count)

	// List all users
	if count > 0 {
		// First, let's see what columns exist
		rows, err := db.Query("SHOW COLUMNS FROM users")
		if err != nil {
			fmt.Println("Failed to show columns:", err)
			return
		}

		fmt.Println("\nUsers table columns:")
		var columns []string
		for rows.Next() {
			var field, fieldType, null, key, defaultVal, extra string
			if err := rows.Scan(&field, &fieldType, &null, &key, &defaultVal, &extra); err != nil {
				// Handle NULL values
				var nullDefault *string
				if err := rows.Scan(&field, &fieldType, &null, &key, &nullDefault, &extra); err != nil {
					fmt.Printf("Failed to scan column info\n")
					continue
				}
			}
			fmt.Printf("- %s (%s)\n", field, fieldType)
			columns = append(columns, field)
		}
		rows.Close()

		// Try to get some data
		fmt.Println("\nAttempting to query available data...")
		rows, err = db.Query("SELECT * FROM users LIMIT 3")
		if err != nil {
			fmt.Println("Failed to query users:", err)
		} else {
			// Get column count
			cols, _ := rows.Columns()
			fmt.Printf("Found %d columns: %v\n", len(cols), cols)

			for rows.Next() {
				// Create slice for values
				values := make([]interface{}, len(cols))
				valuePtrs := make([]interface{}, len(cols))
				for i := range values {
					valuePtrs[i] = &values[i]
				}

				if err := rows.Scan(valuePtrs...); err != nil {
					fmt.Println("Failed to scan row:", err)
					continue
				}

				fmt.Print("Row: ")
				for i, val := range values {
					if val != nil {
						fmt.Printf("%s=%v ", cols[i], val)
					} else {
						fmt.Printf("%s=NULL ", cols[i])
					}
				}
				fmt.Println()
			}
			rows.Close()
		}
	} else {
		fmt.Println("No users found in database")
	}
}
