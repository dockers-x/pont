package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"pont/ent"

	_ "modernc.org/sqlite"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

// Init initializes the database and returns an ent client
func Init(dataDir string) (*ent.Client, error) {
	dbPath := filepath.Join(dataDir, "pont.db")

	// Enable foreign key constraints
	dsn := fmt.Sprintf("%s?_fk=1", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ensure foreign keys are enabled
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))

	// Run auto migration
	if err := client.Schema.Create(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return client, nil
}
