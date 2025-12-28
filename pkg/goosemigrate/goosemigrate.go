package goosemigrate

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Migrator struct {
	postgresURL    string
	migrationsPath string
}

func NewMigrator(postgresURL, migrationsPath string) *Migrator {
	return &Migrator{
		postgresURL:    postgresURL,
		migrationsPath: migrationsPath,
	}
}

func (m *Migrator) Up() error {
	goose.SetTableName("migrations")
	db, err := goose.OpenDBWithDriver("postgres", m.postgresURL)
	if err != nil {
		return fmt.Errorf("failed to open DB for migration: %w", err)
	}

	err = goose.Up(db, m.migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to up migrations: %w", err)
	}

	err = db.Close()
	if err != nil {
		return fmt.Errorf("failed to close db for migration: %w", err)
	}

	return nil
}

func (m *Migrator) Down() error {
	goose.SetTableName("migrations")
	db, err := goose.OpenDBWithDriver("postgres", m.postgresURL)
	if err != nil {
		return fmt.Errorf("failed to open DB for migration: %w", err)
	}

	err = goose.Down(db, m.migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to down migrations: %w", err)
	}

	err = db.Close()
	if err != nil {
		return fmt.Errorf("failed to close db for migration: %w", err)
	}

	return nil
}
