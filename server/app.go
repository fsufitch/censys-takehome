package server

import (
	"context"
	"time"

	"github.com/fsufitch/censys-takehome/database"
	"github.com/fsufitch/censys-takehome/logging"
	"github.com/google/wire"
)

type Server struct {
	Context  context.Context
	Log      logging.LogFunc
	Database *database.DatabaseConnector
}

func (srv Server) Run() error {
	srv.Log().Info().Msg("hello world")

	go srv.doStuff()

	<-srv.Context.Done()
	srv.Log().Info().Msg("server shutting down")
	return nil
}

func (srv Server) doStuff() {
	srv.Log().Info().Msg("doing stuff")

	for {
		<-time.After(1 * time.Second)
		select {
		case <-srv.Context.Done():
			srv.Log().Info().Msg("doing stuff is done")
			return
		default:
		}
		srv.Log().Info().Msg("trying to get db")
		db, err := srv.Database.DB()
		if err != nil {
			srv.Log().Err(err).Msg("failed to get db :(")
			continue
		}

		err = db.PingContext(srv.Context)
		if err != nil {
			srv.Log().Err(err).Msg("ping failed :(")
			continue
		}

		srv.Log().Info().Msg("yay database")
	}
}

var ProvideServer = wire.Struct(new(Server), "*")
