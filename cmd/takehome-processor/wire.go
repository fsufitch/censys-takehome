//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"context"

	"github.com/fsufitch/censys-takehome/config"
	"github.com/fsufitch/censys-takehome/database"
	"github.com/fsufitch/censys-takehome/server"
	"github.com/google/wire"
)

func initializeServer(context.Context, config.PostgresConfiguration, config.LoggingConfiguration) (server.Server, func(), error) {
	panic(wire.Build(ServerProvider))
}

func initializeSchemaDAO(context.Context, config.PostgresConfiguration, config.LoggingConfiguration) (database.SchemaDAO, func(), error) {
	panic(wire.Build(SchemaInitProvider))
}
