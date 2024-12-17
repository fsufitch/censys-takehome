package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/wire"
	"github.com/rs/zerolog"
)

var ErrScanEntry = errors.New("database schema")

type ScanEntryDAO struct {
	*DatabaseConnector
}

type ScanEntry struct {
	IP      net.IP
	Port    uint32
	Service string
	Updated time.Time
	Data    string
}

const upsertEntryQuery = `
	INSERT INTO scan_entries (ip, port, service, updated_on, data)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (ip, port, service) DO UPDATE SET
		updated_on = $4,
		data = $5
`

func (dao ScanEntryDAO) AddEntry(e ScanEntry) error {
	return dao.RunTransaction(nil, func(L zerolog.Logger, tx *sql.Tx) error {
		L = L.With().Str("action", "initSchema").Logger()

		L.Debug().Msg("running query")
		_, err := tx.ExecContext(dao.Context, upsertEntryQuery,
			e.IP.String(), e.Port, e.Service, e.Updated, e.Data,
		)

		if err != nil {
			return fmt.Errorf("%w: query failed: %w", ErrScanEntry, err)
		}

		L.Debug().Msg("commiting")
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("%w: query failed: %w", ErrScanEntry, err)
		}

		L.Debug().Msg("upsert successful")

		return nil
	})
}

var ProvideScanEntryDAO = wire.NewSet(
	ProvideConnector,
	wire.Struct(new(ScanEntryDAO), "*"),
)
