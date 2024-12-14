//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/fsufitch/censys-takehome/config"
	"github.com/fsufitch/censys-takehome/server"
	"github.com/google/wire"
)

func initializeApp(config.PostgresConfiguration, config.LoggingConfiguration) (server.Server, error) {
	panic(wire.Build(AppProviders))
}
