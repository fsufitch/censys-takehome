package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/fsufitch/censys-takehome/config"
	cli "github.com/urfave/cli/v2"
)

func main() {
	ctx := context.Background()

	signal.Ignore(os.Interrupt)
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	app := NewCLI()
	if err := app.RunContext(ctx, os.Args); err != nil {
		panic(err)
	}
}

type CLI *cli.App

func NewCLI() *cli.App {
	return &cli.App{
		Args: false,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "pghost",
				EnvVars: []string{"POSTGRES_HOST"},
				Usage:   "host of the output database server",
			},
			&cli.IntFlag{
				Name:    "pgport",
				EnvVars: []string{"POSTGRES_PORT"},
				Usage:   "port of the output database server",
				Value:   5432,
			},
			&cli.StringFlag{
				Name:    "pguser",
				EnvVars: []string{"POSTGRES_USER"},
				Usage:   "user for connecting to the output database server",
			},
			&cli.StringFlag{
				Name:    "pgpass",
				EnvVars: []string{"POSTGRES_PASSWORD"},
				Usage:   "password for connecting to the output database server; you should use the POSTGRES_PASSWORD env var to specify this",
			},
			&cli.StringFlag{
				Name:    "pgdb",
				EnvVars: []string{"POSTGRES_DB"},
				Usage:   "database name to use",
			},

			&cli.StringFlag{
				Name:    "pubsub-project",
				Aliases: []string{"P"},
				EnvVars: []string{"PUBSUB_PROJECT_ID"},
				Value:   "test-project",
				Usage:   "what Pubsub project to receive data from",
			},
			&cli.StringFlag{
				Name:    "pubsub-topic",
				Aliases: []string{"T"},
				EnvVars: []string{"PUBSUB_TOPIC_ID"},
				Value:   "scan-topic",
				Usage:   "what Pubsub topic to receive data from",
			},

			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"D"},
				Usage:   "enable more thorough debugging",
			},
			&cli.BoolFlag{
				Name:  "pretty",
				Usage: "enable pretty logging",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "server",
				Action: ServerMain,
			},
			{
				Name:   "schema",
				Action: SchemaInitMain,
			},
		},
	}
}

func ServerMain(cctx *cli.Context) error {
	server, cleanup, err := initializeProcessor(
		cctx.Context,
		config.PostgresConfiguration{
			Host:     cctx.String("pghost"),
			Port:     cctx.Int("pgport"),
			User:     cctx.String("pguser"),
			Password: cctx.String("pgpass"),
			Database: cctx.String("pgdb"),
		},
		config.LoggingConfiguration{
			Debug:  cctx.Bool("debug"),
			Pretty: cctx.Bool("pretty"),
		},
		config.PubsubConfiguration{
			ProjectID: cctx.String("pubsub-project"),
			TopicID:   cctx.String("pubsub-topic"),
		},
	)
	if err != nil {
		return err
	}
	err = server.Run()
	cleanup()
	return err
}

func SchemaInitMain(cctx *cli.Context) error {
	dao, cleanup, err := initializeSchemaDAO(
		cctx.Context,
		config.PostgresConfiguration{
			Host:     cctx.String("pghost"),
			Port:     cctx.Int("pgport"),
			User:     cctx.String("pguser"),
			Password: cctx.String("pgpass"),
			Database: cctx.String("pgdb"),
		},
		config.LoggingConfiguration{
			Debug:  cctx.Bool("debug"),
			Pretty: cctx.Bool("pretty"),
		},
	)
	if err != nil {
		return err
	}
	err = dao.InitializeSchema()
	cleanup()
	return err

}
