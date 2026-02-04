package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func main() {
	var err error

	os.MkdirAll("/app/data", 0755)

	db, err = sql.Open("sqlite3", "/app/data/test.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec(`PRAGMA cache_size = 0`)

	// Create table with some data
	db.Exec(`CREATE TABLE IF NOT EXISTS items (id INTEGER PRIMARY KEY, data TEXT, created_at DATETIME)`)

	// Seed some initial data
	for range 1000 {
		db.Exec(`INSERT INTO items (data, created_at) VALUES (?, ?)`,
			randomString(100), time.Now())
	}

	http.HandleFunc("/", handler)
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Force disk I/O: read random rows
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM items WHERE id < ?`, rand.Intn(1000)).Scan(&count)

	// Write new row (generates disk I/O)
	db.Exec(`INSERT INTO items (data, created_at) VALUES (?, ?)`,
		randomString(50), time.Now())

	// Clean up old rows (more I/O)
	db.Exec(`DELETE FROM items WHERE id < ?`, rand.Intn(100))

	fmt.Fprintf(w, "OK - processed %d items\n", count)
}

func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
