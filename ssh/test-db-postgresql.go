package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test-db-postgresql.go <relay_host:port>")
		fmt.Println("Example: go run test-db-postgresql.go localhost:3307")
		os.Exit(1)
	}

	relayAddr := os.Args[1]

	// Connect to PostgreSQL through the relay
	connStr := fmt.Sprintf("host=%s port=5433 user=postgres password=postgres dbname=postgres sslmode=disable",
		getHost(relayAddr))

	fmt.Printf("Connecting to PostgreSQL through relay at %s...\n", relayAddr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Connected successfully! Running test queries...")

	// Run various test queries that should be logged
	testQueries := []struct {
		name  string
		query string
	}{
		{"CREATE TABLE", "CREATE TABLE test_users (id SERIAL PRIMARY KEY, name VARCHAR(100), email VARCHAR(100))"},
		{"INSERT", "INSERT INTO test_users (name, email) VALUES ('John Doe', 'john@example.com')"},
		{"SELECT", "SELECT * FROM test_users"},
		{"UPDATE", "UPDATE test_users SET email = 'john.doe@example.com' WHERE id = 1"},
		{"DELETE", "DELETE FROM test_users WHERE id = 1"},
		{"DESCRIBE (PostgreSQL \\d)", "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'test_users'"},
		{"EXPLAIN", "EXPLAIN SELECT * FROM test_users"},
		{"BEGIN TRANSACTION", "BEGIN"},
		{"COMMIT", "COMMIT"},
		{"ALTER TABLE", "ALTER TABLE test_users ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"},
		{"DROP TABLE", "DROP TABLE IF EXISTS test_users"},
		{"PREPARE", "PREPARE test_stmt AS SELECT $1"},
		{"EXECUTE (should not be logged)", "EXECUTE test_stmt('test')"},
		{"DEALLOCATE", "DEALLOCATE test_stmt"},
	}

	for i, test := range testQueries {
		fmt.Printf("\n[%d] Running %s: %s\n", i+1, test.name, test.query)

		// Execute query
		if _, err := db.Exec(test.query); err != nil {
			fmt.Printf("   Error (expected for some queries): %v\n", err)
		} else {
			fmt.Printf("   Success\n")
		}

		// Small delay to ensure logs are processed
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== Test completed ===")
	fmt.Println("Check the relay logs and database tunnel_logs table to verify:")
	fmt.Println("1. All operations except EXECUTE should be logged to database")
	fmt.Println("2. protocol should be 'postgresql'")
	fmt.Println("3. operation should be cleaned and normalized")
	fmt.Println("4. query_text should be present and cleaned")
	fmt.Println("\nTo check logs in database, run:")
	fmt.Println("SELECT agent_id, client_id, protocol, operation, query_text, created_at FROM tunnel_logs WHERE protocol = 'postgresql' ORDER BY created_at DESC LIMIT 20;")
}

// Helper function to extract host from host:port
func getHost(hostPort string) string {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort // Return as-is if parsing fails
	}
	return host
}
