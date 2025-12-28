package goosemigrate

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type Migrator struct {
	postgresURL    string
	migrationsPath string
	schemaName     string
}

func NewMigrator(postgresURL, migrationsPath, schemaName string) *Migrator {
	return &Migrator{
		postgresURL:    postgresURL,
		migrationsPath: migrationsPath,
		schemaName:     schemaName,
	}
}

func (m *Migrator) Up() error {
	goose.SetTableName(m.schemaName + "." + "migrations")
	db, err := goose.OpenDBWithDriver("postgres", m.postgresURL)
	if err != nil {
		return fmt.Errorf("failed to open DB for migration: %w", err)
	}

	_, err = db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", m.schemaName))
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
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
	goose.SetTableName(m.schemaName + "." + "migrations")
	db, err := goose.OpenDBWithDriver("postgres", m.postgresURL)
	if err != nil {
		return fmt.Errorf("failed to open DB for migration: %w", err)
	}

	err = goose.Down(db, m.migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to down migrations: %w", err)
	}

	_, err = db.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", m.schemaName))
	if err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	err = db.Close()
	if err != nil {
		return fmt.Errorf("failed to close db for migration: %w", err)
	}

	return nil
}
