package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/wire"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

var ErrDatabaseSchema = errors.New("database schema")

type SchemaDAO struct {
	*DatabaseConnector
}

const createSchemaSQL = `
	CREATE TABLE IF NOT EXISTS scan_entries (
		ip inet NOT NULL,
		port integer NOT NULL,
		service varchar NOT NULL,
		updated_on timestamp without time zone NOT NULL,
		data text,
		PRIMARY KEY (ip, port, service)
	)
`

func (dsm SchemaDAO) InitializeSchema() error {
	return dsm.RunTransaction(nil, func(L zerolog.Logger, tx *sql.Tx) error {
		L = L.With().Str("action", "initSchema").Logger()
		L.Debug().Msg("run schema init query")
		if _, err := tx.ExecContext(dsm.Context, createSchemaSQL); err != nil {
			return fmt.Errorf("%w: query failed (%w)", ErrDatabaseSchema, err)
		}

		L.Debug().Msg("commiting")
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("%w: commit failed: %w", ErrDatabaseSchema, err)
		}

		L.Debug().Msg("schema init successful")
		return nil
	})
}

var ProvideSchemaDAO = wire.NewSet(
	wire.Struct(new(SchemaDAO), "*"),
	ProvideConnector,
)
