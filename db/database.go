package db

import (
	"github.com/fsufitch/censys-takehome/config"
	"github.com/fsufitch/censys-takehome/logging"
	"github.com/google/wire"
)

type Database struct {
	Config config.PostgresConfiguration
	Log    logging.LogFunc
}

var ProvideDatabase = wire.Struct(new(Database), "Config", "Log")

func (db *Database) EnsureConnection() error {
	db.Log().Info().Msg("ensure connection")
	return nil
}
