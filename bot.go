package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func startBot(ctx context.Context, cli CLI) error {
	token := cli.Token

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}
	bot.Debug = cli.Debug

	log.Printf("authorized as %s (version=%s, debug=%v)", bot.Self.UserName, version, cli.Debug)

	switch {
	case cli.URL == nil:
		return runPolling(ctx, bot)
	default:
		return runWebhook(ctx, bot, cli.URL, cli.Port)
	}
}

func runPolling(ctx context.Context, bot *tgbotapi.BotAPI) error {
	_, _ = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := bot.GetUpdatesChan(u)
	go func() {
		<-ctx.Done()
		bot.StopReceivingUpdates()
	}()

	return processUpdates(ctx, bot, updates)
}

func runWebhook(ctx context.Context, bot *tgbotapi.BotAPI, hookURL *url.URL, port int) error {
	if hookURL == nil {
		return errors.New("webhook url is empty")
	}

	webhookCfg, err := tgbotapi.NewWebhook(hookURL.String())
	if err != nil {
		return fmt.Errorf("build webhook config: %w", err)
	}
	if _, err := bot.Request(webhookCfg); err != nil {
		return fmt.Errorf("set webhook: %w", err)
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}
	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}

	addr := fmt.Sprintf(":%d", port)
	updates := bot.ListenForWebhook("/")

	server := &http.Server{
		Addr: addr,
	}
	defer gracefulShutdown(ctx, server, bot, hookURL)()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("webhook listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	if err := processUpdates(ctx, bot, updates); err != nil {
		return err
	}
	if err := <-errCh; err != nil {
		return err
	}
	return ctx.Err()
}

func gracefulShutdown(ctx context.Context, server *http.Server, bot *tgbotapi.BotAPI, hookURL *url.URL) func() {
	var once sync.Once

	trigger := func() {
		once.Do(func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("webhook shutdown error: %v", err)
			}
			if bot != nil {
				bot.StopReceivingUpdates()
			}
		})
	}

	go func() {
		<-ctx.Done()
		trigger()
	}()

	return func() {
		cleanupWebhook(bot, hookURL)
		trigger()
	}
}

func cleanupWebhook(bot *tgbotapi.BotAPI, hookURL *url.URL) {
	if bot == nil {
		return
	}
	if _, err := bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false}); err != nil {
		log.Printf("failed to delete webhook: %v", err)
		return
	}
	log.Printf("webhook %s successfully deleted", hookURL)
}

func processUpdates(ctx context.Context, bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil {
				continue
			}
			go handleMessage(ctx, bot, update.Message)
		}
	}
}
