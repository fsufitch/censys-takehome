//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"context"

	"github.com/fsufitch/censys-takehome/config"
	"github.com/fsufitch/censys-takehome/database"
	"github.com/fsufitch/censys-takehome/logging"
	"github.com/fsufitch/censys-takehome/processor"
	"github.com/google/wire"
)

func initializeProcessor(context.Context, config.PostgresConfiguration, config.LoggingConfiguration, config.PubsubConfiguration) (processor.Processor, func(), error) {
	panic(wire.Build(
		processor.ProvideProcessor,
		logging.ProvideLogFunc,
		database.ProvideScanEntryDAO,
	))
}

func initializeSchemaDAO(context.Context, config.PostgresConfiguration, config.LoggingConfiguration) (database.SchemaDAO, func(), error) {
	panic(wire.Build(
		database.ProvideSchemaDAO,
		logging.ProvideLogFunc,
	))
}
