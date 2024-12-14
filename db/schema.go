package db

func (db Database) InitializeSchema() {
	db.Log().Info().Msg("initializing schema")

	db.EnsureConnection()
}
