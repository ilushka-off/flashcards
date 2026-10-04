package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func toPgxMigrateURI(databaseURI string) (string, error) {
	switch {
	case strings.HasPrefix(databaseURI, "postgres://"):
		return "pgx5://" + strings.TrimPrefix(databaseURI, "postgres://"), nil
	case strings.HasPrefix(databaseURI, "postgresql://"):
		return "pgx5://" + strings.TrimPrefix(databaseURI, "postgresql://"), nil
	}
	return "", fmt.Errorf("unsupported database uri scheme: %q", databaseURI)
}

func Run(databaseURI string) error {
	sourceDriver, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migrations source: %w", err)
	}

	migrateURI, err := toPgxMigrateURI(databaseURI)
	if err != nil {
		return fmt.Errorf("build migrate uri: %w", err)
	}

	migrator, err := migrate.NewWithSourceInstance("iofs", sourceDriver, migrateURI)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	defer migrator.Close()

	err = migrator.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
