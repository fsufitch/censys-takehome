package server

import (
	"context"
	"net"
	"time"

	"github.com/fsufitch/censys-takehome/database"
	"github.com/fsufitch/censys-takehome/logging"
	"github.com/google/wire"
)

type Server struct {
	Context      context.Context
	Log          logging.LogFunc
	ScanEntryDAO *database.ScanEntryDAO
}

func (srv Server) Run() error {
	srv.Log().Info().Msg("server starting")

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

		entry := database.ScanEntry{
			IP:      net.IPv4(127, 0, 0, 1),
			Port:    8080,
			Service: "",
			Updated: time.Now(),
			Data:    []byte("hello"),
		}

		srv.Log().Info().Any("entry", entry).Msg("upsert entry")

		err := srv.ScanEntryDAO.AddEntry(entry)
		if err != nil {
			srv.Log().Err(err).Msg("upsert failed")
		}
	}
}

var ProvideServer = wire.Struct(new(Server), "*")
