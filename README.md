# Audio2Voice Bot

This is a fun side project, lovingly vibe-coded to turn any Telegram audio into a voice message. It listens for audio files (mp3, m4a, ogg/vorbis, etc.), downloads them, runs `ffmpeg` to convert to an Opus voice note, and sends the result back to the user.

## Features

- Accepts audio attachments (Telegram audio or document uploads).
- Converts them to voice messages via `ffmpeg` (mono, 48kHz, Opus).
- Supports both polling and webhook modes.
- Includes concurrency, graceful shutdown, and error reporting with ffmpeg logs.
- Command replies show a GitHub link (injected at build time).

## Usage

Build with Go 1.21+ (or any recent Go supporting generics):

```bash
go build -ldflags "-X main.version=1.0.0 -X main.repoURL=https://github.com/you/audio2voice"
./audio2voice --token "<telegram-token>" --debug
```

### CLI Flags

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `-t`, `--token` | `TG_A2V_TOKEN` | _required_ | Telegram bot token. |
| `-u`, `--url` | `TG_A2V_WEBHOOK_URL` | empty | Webhook URL (switches to webhook mode if provided and `--debug` is off). |
| `-p`, `--port` | `TG_A2V_PORT` | `8080` | Local HTTP port for webhook listener. |
| `--debug` | — | `false` | Enables debug logging and forces polling mode. |
| `-v`, `--version` | — | — | Prints version and exits. |
| `-h`, `--help` | — | — | Shows usage info. *(Provided by Kong)* |

### Required Environment Variables

You can supply flags or rely on env vars:

- `TG_A2V_TOKEN`: Bot token (from BotFather).
- `TG_A2V_WEBHOOK_URL`: Optional webhook endpoint.
- `TG_A2V_PORT`: Port for webhook server (when `--url` set).

### Runtime Requirements

- `ffmpeg` must be installed and available in `$PATH`.
- The bot needs outbound network access to the Telegram API.
- In webhook mode, expose the port with HTTPS (handle TLS via reverse proxy or adjust server logic).

## How It Works

1. Bot pulls updates (polling) or receives them via webhook.
2. Each update is processed by a worker pool.
3. Audio is downloaded using Telegram's file API.
4. `ffmpeg` transforms the file into Opus `.ogg` suitable for Telegram voice.
5. The voice message is sent back, with detailed errors surfaced to the user if conversion fails.

Graceful shutdown waits for active jobs to finish and tears down the webhook HTTP server cleanly.

## Fun Fact

This bot was vibe-coded for fun—tinker, fork, remix, and enjoy! Pull requests welcome if you want to turn it into something more serious. :)
