package migration

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

const gooseVersionRaiseRiverToV1 int64 = 1

// compile time: bundles the SQL files into the binary
//
//go:embed sql/*.sql
var embeddedMigrations embed.FS

func NewSchemaMigrator(db *sql.DB) (Migrator, error) {

	sqlFS, err := fs.Sub(embeddedMigrations, "sql")
	if err != nil {
		return nil, fmt.Errorf("migrations: accessing sql subdirectory: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres, db, sqlFS,
		goose.WithAllowOutofOrder(false),
		goose.WithGoMigrations(
			goose.NewGoMigration(
				gooseVersionRaiseRiverToV1,
				&goose.GoFunc{
					RunDB: upRiverSchema,
				},
				&goose.GoFunc{
					RunDB: downRiverSchema,
				},
			),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("migrations: creating provider: %w", err)
	}

	return &gooseMigrator{provider}, nil
}

type gooseMigrator struct {
	*goose.Provider
}

func (gm *gooseMigrator) Up(ctx context.Context) error {
	_, err := gm.Provider.Up(ctx)
	return err
}

func (gm *gooseMigrator) Down(ctx context.Context) error {
	_, err := gm.Provider.Down(ctx)
	return err
}

type Migrator interface {
	Up(ctx context.Context) error
	Down(ctx context.Context) error
}
