package server

import (
	"context"

	"github.com/fsufitch/censys-takehome/db"
	"github.com/fsufitch/censys-takehome/logging"
	"github.com/google/wire"
)

type Server struct {
	Log      logging.LogFunc
	Database *db.Database
}

func (srv Server) Run(ctx context.Context) error {
	srv.Log().Info().Msg("hello world")

	<-ctx.Done()
	srv.Log().Info().Msg("server shutting down")
	return nil
}

var ProvideServer = wire.Struct(new(Server), "*")
