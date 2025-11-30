package main

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestProcessUpdatesContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := processUpdates(ctx, nil, make(chan tgbotapi.Update))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", err)
	}
}

func TestProcessUpdatesClosedChannel(t *testing.T) {
	t.Parallel()

	updates := make(chan tgbotapi.Update, 1)
	updates <- tgbotapi.Update{} // message is nil; ensures no handler call
	close(updates)

	if err := processUpdates(context.Background(), nil, updates); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
