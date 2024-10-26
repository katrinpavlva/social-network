package datab

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"path/filepath"
)

func ConnectDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "datab.db")
	if err != nil {
		log.Fatalf("Failed to connect to the datab: %v", err)
		return nil, err
	}
	return db, nil
}

func CreateTables(db *sql.DB) error {
	sqlQueries, err := ioutil.ReadFile("./datab/table.sql")
	if err != nil {
		log.Fatalf("Failed to read SQL file: %v", err)
		return err
	}

	_, err = db.Exec(string(sqlQueries))
	if err != nil {
		log.Fatalf("Failed to execute SQL queries: %v", err)
		return err
	}

	return nil
}

func ApplyMigrations(db *sql.DB, migrationsPath string) error {
	// List all .up.sql files in the migrations directory
	files, err := filepath.Glob(filepath.Join(migrationsPath, "*.up.sql"))
	if err != nil {
		return err
	}

	for _, file := range files {
		log.Printf("Applying migration file: %s", file)
		migration, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		// Execute migration query
		_, err = db.Exec(string(migration))
		if err != nil {
			return err
		}
	}

	return nil
}
