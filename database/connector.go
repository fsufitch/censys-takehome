package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/fsufitch/censys-takehome/config"
	"github.com/fsufitch/censys-takehome/logging"

	_ "github.com/lib/pq"
)

type DatabaseConnector struct {
	Context context.Context
	Config  config.PostgresConfiguration
	Log     logging.LogFunc

	Finalized chan error // Channel is closed once connector work is finalized

	connectTrigger    chan struct{} // Close this chan to cause a new connection
	newConnections    chan *sql.DB  // New connections are delivered here; if "nil" is delivered, then no connection is currently available
	currentConections chan *sql.DB  // Used for serving connections to users of the connector
}

func ProvideDatabaseConnector(ctx context.Context, config config.PostgresConfiguration, logFunc logging.LogFunc) (*DatabaseConnector, func(), error) {
	dbc := &DatabaseConnector{
		Context: ctx,
		Config:  config,
		Log:     logFunc,

		Finalized: make(chan error),

		connectTrigger:    make(chan struct{}), // close this chan whenever
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
			dbc.Reconnect()
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

func (dbc *DatabaseConnector) Reconnect() {
	select {
	case <-dbc.connectTrigger:
		// If the trigger is already closed, no need to do anything
		dbc.Log().Debug().Msg("reconnect already in progress")
		return
	default:
	}
	// Otherwise, close it
	dbc.Log().Info().Msg("requesting reconnect")
	close(dbc.connectTrigger)
}

// newConnectionWorker is a worker function which creates a new worker whenever connectTrigger is closed
func (dbc *DatabaseConnector) newConnectionWorker() {
	workerLog := dbc.Log().With().Str("worker", "newConnection").Logger()
	workerLog.Debug().Msg("worker starting")
worker:
	// Overall worker loop
	for {
		select {
		case <-dbc.Context.Done():
			break worker
		case <-dbc.connectTrigger:
			// Exit this select if a new connection is triggered
		}

		workerLog.Info().Msg("new connection triggered")

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
		dbc.connectTrigger = make(chan struct{})
	}
	workerLog.Warn().Msg("worker terminating")
}

// connectionRepeaterWorker reads newConnections and delivers their results repeatedly to currentConnections
func (dbc *DatabaseConnector) connectionRepeaterWorker() {
	workerLog := dbc.Log().With().Str("worker", "newConnection").Logger()
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

		workerLog.Debug().Msg("have non-nil connection")

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