package config

type LoggingConfiguration struct {
	Debug  bool
	Pretty bool
}

type PostgresConfiguration struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

type PubsubConfiguration struct {
	ProjectID string
	TopicID   string
}
