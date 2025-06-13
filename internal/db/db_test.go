package db

import (
	"database/sql"
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	// Create a temporary database file
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Test database initialization
	db, err := Init(tmpfile.Name())
	if err != nil {
		t.Fatalf("Init() failed: %v", err)
	}
	defer db.Close()

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign_keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("Foreign keys should be enabled")
	}

	// Verify tables were created
	tables := []string{"entities", "relations", "observations"}
	for _, table := range tables {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("Table %s should exist", table)
		}
	}
}

func TestInitInvalidPath(t *testing.T) {
	// Test with invalid path
	_, err := Init("/invalid/path/database.db")
	if err == nil {
		t.Error("Init() should fail with invalid path")
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	tmpfile, err := os.CreateTemp("", "test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	
	db, err := Init(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	
	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpfile.Name())
	})
	
	return db
}