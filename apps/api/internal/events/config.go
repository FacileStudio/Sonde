// Package events publishes monitor transitions to the Antenne event bus through
// antenne-client's durable outbox. The recorder queues an event in the same
// transaction as the incident write that caused it, and the Publisher drains
// the queue to the bus in order. Fan-out to humans is Antenne's job; Sonde
// only reports what changed.
package events

import (
	stderrors "errors"

	antenne "github.com/FacileStudio/antenne-client/go"
	"github.com/FacileStudio/antenne-client/go/catalog"
)

// DefaultApp is the pool identity Sonde registers under.
const DefaultApp = "sonde"

// Channels lists what Sonde emits: an outage and a recovery, on two channels so
// a delivery target can subscribe to outages without taking every recovery.
func Channels() []string {
	return []string{string(catalog.MonitorDown), string(catalog.MonitorUp)}
}

// ConfigFromEnv reads the Antenne connection from ANTENNE_URL, or its older
// synonym ANTENNE_INSTANCE, with ANTENNE_APP, ANTENNE_SECRET and
// ANTENNE_INSTANCE_ID. Neither URL variable set returns nil and no error:
// publishing is off. A URL without a secret is an error, because the bus would
// refuse the registration and events would look configured while nothing left.
func ConfigFromEnv() (*antenne.Config, error) {
	config, err := antenne.ConfigFromEnv(DefaultApp)
	if stderrors.Is(err, antenne.ErrNotConfigured) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	config.Events = antenne.EventConfig{Emit: Channels()}
	return config, nil
}
