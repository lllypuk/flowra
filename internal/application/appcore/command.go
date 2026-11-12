package appcore

import (
	"context"
	"errors"
)

// ErrDuplicateCommand is returned by an event store when events with the same command key are already stored.
var ErrDuplicateCommand = errors.New("command already applied")

// CommandKey identifies a client command so that a repeated request is applied once.
type CommandKey struct {
	// ID is the client key, unique within one aggregate.
	ID string
	// Fingerprint describes the request; the same ID with another fingerprint is a different request.
	Fingerprint string
}

type commandKey struct{}

// WithCommand marks events saved under ctx as the effect of cmd.
// The key travels in the context so that repositories and event stores keep their signatures.
func WithCommand(ctx context.Context, cmd CommandKey) context.Context {
	if cmd.ID == "" {
		return ctx
	}
	return context.WithValue(ctx, commandKey{}, cmd)
}

// CommandFrom returns the command set by WithCommand.
func CommandFrom(ctx context.Context) (CommandKey, bool) {
	cmd, ok := ctx.Value(commandKey{}).(CommandKey)
	return cmd, ok
}

// CommandRecord is the stored effect of a command.
type CommandRecord struct {
	// Version is the aggregate version after the command's events.
	Version     int
	Fingerprint string
}
