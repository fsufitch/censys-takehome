package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/fsufitch/censys-takehome/config"
	"github.com/fsufitch/censys-takehome/logging"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	_ "github.com/lib/pq"
)

var ErrConnection = errors.New("database connection error")

type DatabaseConnector struct {
	Context context.Context
	Config  config.PostgresConfiguration
	Log     logging.LogFunc

	Finalized chan error // Channel is closed once connector work is finalized

	connectionTrigger chan struct{} // Send on this channel to cause a connection
	newConnections    chan *sql.DB  // New connections are delivered here; if "nil" is delivered, then no connection is currently available
	currentConections chan *sql.DB  // Used for serving connections to users of the connector
}

func ProvideConnector(ctx context.Context, config config.PostgresConfiguration, logFunc logging.LogFunc) (*DatabaseConnector, func(), error) {
	dbc := &DatabaseConnector{
		Context: ctx,
		Config:  config,
		Log:     logFunc,

		Finalized: make(chan error),

		connectionTrigger: make(chan struct{}),
		newConnections:    make(chan *sql.DB),
		currentConections: make(chan *sql.DB),
	}

	go dbc.newConnectionWorker()
	go dbc.connectionRepeaterWorker()

	cleanup := func() {
		dbc.Log().Info().Msg("cleaning up database connector")
		close(dbc.newConnections)

		var db *sql.DB
		select {
		case db = <-dbc.currentConections:
		default:
		}

		close(dbc.currentConections)

		if db != nil {
			dbc.Log().Info().Msg("cleaning up database connection")
			err := db.Close()
			if err != nil {
				dbc.Log().Err(err).Msg("error cleaning up database connection")
				return
			}
		}
	}

	return dbc, cleanup, nil

}

func (dbc *DatabaseConnector) DB() (*sql.DB, error) {
	errQuit := fmt.Errorf("%w: connections unavailable (connector quit)", ErrConnection)
	for {
		select {
		case <-dbc.Context.Done():
			return nil, errQuit
		case db, ok := <-dbc.currentConections:
			if !ok {
				return nil, errQuit
			}
			if db == nil {
				return nil, fmt.Errorf("%w: current connection is nil (should be impossible)", ErrConnection)
			}
			return db, nil
		case <-time.After(1 * time.Second):
			if err := dbc.Reconnect(); err != nil {
				return nil, fmt.Errorf("reconnect failed: %w", err)
			}
			select {
			case <-dbc.Context.Done():
				return nil, fmt.Errorf("%w: connections unavailable (connector quit)", ErrConnection)
			case db, ok := <-dbc.currentConections:
				if !ok {
					return nil, errQuit
				}
				if db == nil {
					return nil, fmt.Errorf("%w: current connection is nil (should be impossible)", ErrConnection)
				}
				return db, nil
			}
		}
	}
}

func (dbc *DatabaseConnector) Reconnect() error {
	select {
	case dbc.connectionTrigger <- struct{}{}:
		return nil
	default:
		return fmt.Errorf("%w: failed to send connection trigger; is the worker dead?", ErrConnection)
	}
}

// newConnectionWorker is a worker function which creates a new worker whenever connectTrigger is closed
func (dbc *DatabaseConnector) newConnectionWorker() {
	workerLog := dbc.Log().With().Str("worker", "newConnection").Logger()
	workerLog.Debug().Msg("worker starting")

	// Loop for ignoring triggers while connection work is in progress
	dedupedTrigger := make(chan struct{}, 32)
	stopDedupe := make(chan struct{})
	defer close(dedupedTrigger)
	defer close(stopDedupe)
	go func() {
		for range dbc.connectionTrigger {
			dedupedTrigger <- struct{}{}
		stopDedupeLoop:
			for {
				select {
				case <-stopDedupe:
					break stopDedupeLoop
				case <-dbc.connectionTrigger:
					dbc.Log().Debug().Msg("connection already in progress")
				}
			}
		}
	}()

worker:
	// Overall worker loop
	for {
		// Wait until a connect is triggered, or the context is canceled
		select {
		case <-dbc.Context.Done():
			break worker
		case <-dedupedTrigger:
		}

		workerLog.Info().Msg("new connection triggered")

		// Drain any connection triggers while we're actually connecting
		stopDrainingTriggers := make(chan struct{})
		defer close(stopDrainingTriggers)
		go func() {
			for {
				select {
				case <-stopDrainingTriggers:
					return
				case <-dbc.connectionTrigger:
					workerLog.Debug().Msg("connection already in progress")
				}
			}
		}()

		var db *sql.DB
		var err error
		attempt := 0

		// Connection attempt loop
	connectSuccess:
		for {
			attempt++
			attemptLog := workerLog.With().Int("attempt", attempt).Logger()

			// If the worker's context is done, quit
			select {
			case <-dbc.Context.Done():
				break worker
			default:
				// Otherwise, just do another attempt
			}

			// Make an attempt to connect
			connURL := &url.URL{
				Scheme:   "postgres",
				Host:     fmt.Sprintf("%s:%d", dbc.Config.Host, dbc.Config.Port),
				User:     url.UserPassword(dbc.Config.User, dbc.Config.Password),
				Path:     fmt.Sprintf("/%s", dbc.Config.Database),
				RawQuery: url.Values{"sslmode": []string{"disable"}}.Encode(),
			}

			attemptLog.Info().Str("host", connURL.Host).Str("db", dbc.Config.Database).Str("user", dbc.Config.User).Msg("trying to connect")
			db, err = sql.Open("postgres", connURL.String())

			// If success, quit the loop
			if err == nil {
				if err = db.PingContext(dbc.Context); err == nil {
					break connectSuccess
				}
			}

			// Otherwise, report the failure, wait a second, and try again
			attemptLog.Err(err).Msg("connection failed")
			select {
			case <-dbc.Context.Done():
				break worker
			case <-time.After(1 * time.Second):
			}
		}

		// End of connection loop, `db` should be a successful connection; deliver it and reset the connect trigger
		workerLog.Info().Msg("connection successful")

		// If an old connection exists, close it
		select {
		case oldDB := <-dbc.currentConections:
			workerLog.Info().Msg("old connection exists, closing")
			err := oldDB.Close()
			if err != nil {
				workerLog.Err(err).Msg("error closing connection")
			}
		default:
		}

		workerLog.Debug().Msg("sending new connection")
		dbc.newConnections <- db

		stopDedupe <- struct{}{}
	}

	workerLog.Warn().Msg("worker terminating")
}

// connectionRepeaterWorker reads newConnections and delivers their results repeatedly to currentConnections
func (dbc *DatabaseConnector) connectionRepeaterWorker() {
	workerLog := dbc.Log().With().Str("worker", "connectionRepeater").Logger()
	workerLog.Debug().Msg("worker starting")

	var conn *sql.DB
	var ok bool

worker:
	for {
		for conn == nil {
			select {
			case <-dbc.Context.Done():
				break worker
			case conn, ok = <-dbc.newConnections:
				if !ok {
					workerLog.Warn().Msg("newConnections closed")
					break worker
				}
			}
		}

		select {
		case <-dbc.Context.Done():
			break worker
		case dbc.currentConections <- conn:
			workerLog.Debug().Msg("sent connection")
		case conn, ok = <-dbc.newConnections:
			workerLog.Debug().Msg("received next connection")
			if !ok {
				workerLog.Warn().Msg("newConnections closed")
				break worker
			}
		}
	}
	workerLog.Warn().Msg("worker terminating")
}

func (dbc *DatabaseConnector) RunTransaction(opts *sql.TxOptions, cb func(zerolog.Logger, *sql.Tx) error) error {
	txID, err := uuid.NewRandom()
	if err != nil {
		dbc.Log().Err(err).Msg("failed to create UUID")
		return err
	}

	db, err := dbc.DB()
	if err != nil {
		return err
	}

	txLog := dbc.Log().With().Str("tx", txID.String()).Logger()
	txLog.Debug().Msg("begin")
	tx, err := db.BeginTx(dbc.Context, opts)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	return cb(txLog, tx)
}
