package migrations

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed *.sql
var migrationFiles embed.FS

func Run(ctx context.Context, db *pgxpool.Pool) error {
	// Check if this is an existing DB (has tables but no schema_migrations)
	isExistingDB := false
	var tableExists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'peers')
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check existing tables: %w", err)
	}

	var migTableExists bool
	err = db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'schema_migrations')
	`).Scan(&migTableExists)
	if err != nil {
		return fmt.Errorf("failed to check schema_migrations: %w", err)
	}

	if tableExists && !migTableExists {
		isExistingDB = true
	}

	// Create migrations tracking table
	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get already applied migrations
	rows, err := db.Query(ctx, "SELECT filename FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return err
		}
		applied[filename] = true
	}

	// Read all SQL files
	entries, err := migrationFiles.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var filenames []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			filenames = append(filenames, e.Name())
		}
	}
	sort.Strings(filenames)

	// If existing DB without schema_migrations, seed all previously-applied migrations
	// so only truly new ones get executed
	if isExistingDB && len(applied) == 0 {
		log.Println("Existing database detected, seeding migration history...")
		for _, filename := range filenames {
			// The last migration is the new one we actually need to apply
			// Seed everything except migrations that haven't been applied yet
			// We detect this by trying to check if the column/table from migration exists
			// Simpler: seed all known pre-existing migrations (001-005), run 006+
			if filename >= "006_" {
				break
			}
			if _, err := db.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING", filename); err != nil {
				return fmt.Errorf("failed to seed migration %s: %w", filename, err)
			}
			applied[filename] = true
			log.Printf("Seeded existing migration: %s", filename)
		}
	}

	// Apply pending migrations
	for _, filename := range filenames {
		if applied[filename] {
			continue
		}

		content, err := migrationFiles.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		log.Printf("Applying migration: %s", filename)

		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin tx for %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to apply migration %s: %w", filename, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1)", filename); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}

		log.Printf("Applied migration: %s", filename)
	}

	return nil
}
