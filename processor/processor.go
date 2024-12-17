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
	"github.com/google/wire"
)

var ErrProcessor = errors.New("processor")

type Processor struct {
	Context      context.Context
	Config       config.PubsubConfiguration
	Log          logging.LogFunc
	ScanEntryDAO *database.ScanEntryDAO
}

func (proc *Processor) Run() error {
	proc.Log().Info().Msg("processor starting")

	proc.Log().Debug().Str("project", proc.Config.ProjectID).Msg("connecting to pubsub")
	client, err := pubsub.NewClient(proc.Context, proc.Config.ProjectID)
	if err != nil {
		return fmt.Errorf("%w: failed connecting to pubsub (%s): %w", ErrProcessor, proc.Config.ProjectID, err)
	}

	proc.Log().Debug().Msg("getting subscription")
	subscription := client.Subscription(proc.Config.SubscriptionID)
	if exists, err := subscription.Exists(proc.Context); !exists || err != nil {
		return fmt.Errorf("%w: subscription does not exist (%s): %w", ErrProcessor, proc.Config.SubscriptionID, err)
	}

	subscription.Receive(proc.Context, proc.receive)

	proc.Log().Warn().Msg("processor shutting down")

	return nil
}

func (proc *Processor) receive(msgContext context.Context, msg *pubsub.Message) {
	L := proc.Log().With().Str("msgID", msg.ID).Logger()
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
		msg.Nack() // Do *not* acknowledge it; it is a valid entry and should be retried
		return
	}

	L.Info().Msg("successfully recorded entry")
	msg.Ack()
}

var ProvideProcessor = wire.Struct(new(Processor), "Context", "Config", "Log", "ScanEntryDAO")
