package migration

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
)

const riverSchemaTargetVersion = 6

func upRiverSchema(ctx context.Context, db *sql.DB) error {

	migrator, err := rivermigrate.New(riverdatabasesql.New(db), nil)
	if err != nil {
		return fmt.Errorf("failed creating river migrator:%w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{
		TargetVersion: riverSchemaTargetVersion,
	})
	return err
}

func downRiverSchema(ctx context.Context, db *sql.DB) error {
	migrator, err := rivermigrate.New(riverdatabasesql.New(db), nil)
	if err != nil {
		return fmt.Errorf("failed creating river migrator: %w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionDown, &rivermigrate.MigrateOpts{
		TargetVersion: -1,
	})
	return err
}
