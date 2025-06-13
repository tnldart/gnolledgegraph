package db

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

func Init(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// enable foreign keys
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, err
	}
	// run schema migrations
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS entities (
			name TEXT PRIMARY KEY,
			entity_type TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_entity TEXT NOT NULL REFERENCES entities(name),
			to_entity TEXT NOT NULL REFERENCES entities(name),
			relation_type TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS observations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_name TEXT NOT NULL REFERENCES entities(name),
			content TEXT NOT NULL
		);`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return nil, err
		}
	}
	return db, nil
}
