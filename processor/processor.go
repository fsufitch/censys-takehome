package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/fsufitch/censys-takehome/config"
	"github.com/fsufitch/censys-takehome/database"
	"github.com/fsufitch/censys-takehome/logging"
	"github.com/google/uuid"
	"github.com/google/wire"
)

var ErrProcessor = errors.New("processor")

type Processor struct {
	Context      context.Context
	Config       config.PubsubConfiguration
	Log          logging.LogFunc
	ScanEntryDAO *database.ScanEntryDAO

	subscriptionID string
}

func (proc *Processor) Run() error {
	if proc.subscriptionID != "" {
		return fmt.Errorf("%w: processor already has an active subscription (%s)", ErrProcessor, proc.subscriptionID)
	}

	newUUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("%w: failed creating subscription ID: %w", ErrProcessor, err)
	}
	proc.subscriptionID = newUUID.String()
	L := proc.Log().With().Str("sub", proc.subscriptionID).Logger()

	L.Info().Msg("processor starting")

	L.Debug().Str("project", proc.Config.ProjectID).Msg("connecting to pubsub")
	client, err := pubsub.NewClient(proc.Context, proc.Config.ProjectID)
	if err != nil {
		return fmt.Errorf("%w: failed connecting to pubsub (%s): %w", ErrProcessor, proc.Config.ProjectID, err)
	}

	L.Debug().Str("topic", proc.Config.TopicID).Msg("checking topic existence")
	topic := client.Topic(proc.Config.TopicID)
	if exists, err := topic.Exists(proc.Context); !exists || err != nil {
		return fmt.Errorf("%w: topic does not exist (%s): %w", ErrProcessor, proc.Config.TopicID, err)
	}

	L.Debug().Msg("creating subscription")
	subscription, err := client.CreateSubscription(proc.Context, proc.subscriptionID, pubsub.SubscriptionConfig{
		Topic: topic,
	})
	if err != nil {
		return fmt.Errorf("%w: error creating subscription: %w", ErrProcessor, err)
	}

	subscription.Receive(proc.Context, proc.receive)

	proc.Log().Info().Msg("processor shutting down")
	return nil
}

func (proc *Processor) receive(msgContext context.Context, msg *pubsub.Message) {
	L := proc.Log().With().Str("sub", proc.subscriptionID).Logger()
	L.Info().Msg("received message")

	scan := Scan{}
	err := json.Unmarshal(msg.Data, &scan)
	if err != nil {
		L.Err(err).Bytes("data", msg.Data).Msg("failed to unmarshal message data")
		msg.Ack() // Ack so it doesn't get delivered again
		return
	}

	scanData, err := scan.DataBytes()
	if err != nil {
		L.Err(err).Bytes("data", msg.Data).Msg("failed to extract entry data")
		msg.Ack() // Ack so it doesn't get delivered again
		return
	}

	entry := database.ScanEntry{
		IP:      net.ParseIP(scan.IP),
		Port:    scan.Port,
		Service: scan.Service,
		Updated: time.Unix(scan.Timestamp, 0),
		Data:    scanData,
	}

	L.Info().Any("entry", entry).Msg("extracted entry from message")

	err = proc.ScanEntryDAO.AddEntry(entry)
	if err != nil {
		L.Err(err).Msg("error upserting entry")
		msg.Ack()
		return
	}

	L.Info().Msg("successfully recorded entry")
	msg.Ack()
	return
}

var ProvideProcessor = wire.Struct(new(Processor), "Context", "Config", "Log", "ScanEntryDAO")
