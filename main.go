package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Token   string   `short:"t" long:"token" env:"TG_A2V_TOKEN" required:"" placeholder:"201204456:AAFFJJ" help:"Bot token."`
	URL     *url.URL `short:"u" long:"url" env:"TG_A2V_WEBHOOK_URL" placeholder:"https://example.com/bot-secret-url" help:"Webhook URL (used when debug is off)."`
	Port    int      `short:"p" long:"port" env:"TG_A2V_PORT" default:"${defaultPort}" help:"HTTP port for webhook listener."`
	Debug   bool     `long:"debug" help:"Enable debug log and force polling mode."`
	Version bool     `short:"v" long:"version" help:"Print version."`
}

var (
	version = "dev"
	repoURL = "https://github.com/azhinu/audio2voice"
	cli     CLI
)

const (
	defaultTimeout = 2 * time.Minute
	defaultPort    = 8080
	appDescription = "Telegram audio-to-voice bot (version {{version}}). Source: {{repoURL}}"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	kong.Parse(&cli,
		kong.Description(appDescription),
		kong.Vars{
			"version":     version,
			"defaultPort": fmt.Sprintf("%d", defaultPort),
		},
	)

	if cli.Version {
		log.Println(version)
		return
	}

	if err := startBot(ctx, cli); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("bot exited with error: %v", err)
	}
}
