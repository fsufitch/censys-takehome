package database

import "github.com/google/wire"

var ProvideDatabase = wire.NewSet(
	ProvideDatabaseConnector,
)
