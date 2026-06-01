# superset-telegram-bridge

[![CI](https://github.com/dengaleev/superset-telegram-bridge/actions/workflows/ci.yml/badge.svg)](https://github.com/dengaleev/superset-telegram-bridge/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dengaleev/superset-telegram-bridge)](https://goreportcard.com/report/github.com/dengaleev/superset-telegram-bridge)
[![Go Version](https://img.shields.io/github/go-mod/go-version/dengaleev/superset-telegram-bridge)](go.mod)
[![Release](https://img.shields.io/github/v/release/dengaleev/superset-telegram-bridge?sort=semver)](https://github.com/dengaleev/superset-telegram-bridge/releases)
[![License: MIT](https://img.shields.io/github/license/dengaleev/superset-telegram-bridge)](LICENSE)

A tiny, single-binary service that forwards [Apache Superset](https://superset.apache.org/) alert and report notifications to a Telegram chat.

Superset can POST notifications to a webhook URL. This bridge receives that webhook, renders it as a clean Telegram message with an inline **Open in Superset** link, forwards any attachments as photos or documents, and delivers everything via the Telegram Bot API — so your team gets Superset alerts where it already talks.

```
Superset  ──POST /webhook──▶  superset-telegram-bridge  ──Bot API──▶  Telegram chat
```

## Features

- **Single static binary** — no runtime dependencies; ships as a distroless, non-root container image (under 10 MB: a ~7 MB CGO-free binary on a `static:nonroot` base).
- **Clean messages** — HTML-formatted title (bold), body, and description (italic), with an inline **Open in Superset** link back to the chart or dashboard; link previews are disabled on text messages.
- **Attachment forwarding** — Superset report screenshots (`image/png`) are sent as Telegram photos and other files (CSV, PDF, …) as documents; two or more of a kind become an album, with the notification text as the caption.
- **Resilient delivery** — transport-level failures are retried once after a short fixed delay (a non-2xx response is surfaced immediately); the Telegram bot token is stripped from transport errors before they are logged or returned.
- **Operationally boring** — structured JSON logging (`slog`), a `GET /healthz` endpoint, graceful shutdown, and a request body bounded to 50 MiB.
- **Multi-arch** — `linux/amd64` and `linux/arm64` images published to GHCR on every release.

## Quick start

```bash
docker run -d --name superset-bridge \
  -p 8080:8080 \
  -e TELEGRAM_TOKEN="123456:your-bot-token" \
  -e TELEGRAM_CHAT_ID="-1001234567890" \
  ghcr.io/dengaleev/superset-telegram-bridge:latest
```

Then point a Superset notification at `http://<host>:8080/webhook` (see [Superset setup](#superset-setup)).

> Need a bot token and chat ID? Create a bot with [@BotFather](https://t.me/BotFather), add it to your target chat, and read the chat ID from `https://api.telegram.org/bot<TOKEN>/getUpdates`.

## Configuration

All configuration is via environment variables.

| Variable           | Required | Default | Description                                                                  |
| ------------------ | :------: | ------- | ---------------------------------------------------------------------------- |
| `TELEGRAM_TOKEN`   |   yes    | —       | Telegram Bot API token from BotFather.                                       |
| `TELEGRAM_CHAT_ID` |   yes    | —       | Target chat ID (a user, group, or channel the bot can post to).              |
| `LISTEN_ADDR`      |    no    | `:8080` | Address the HTTP server binds to.                                            |
| `LOG_LEVEL`        |    no    | `info`  | One of `debug`, `info`, `warn` (or `warning`), `error` — case-insensitive.   |

## Endpoints

| Method | Path       | Purpose                                                                                  |
| ------ | ---------- | ---------------------------------------------------------------------------------------- |
| `POST` | `/webhook` | Receives a Superset notification: `application/json` (text) or `multipart/form-data` (with attachments). Any other content type returns `415`. |
| `GET`  | `/healthz` | Liveness probe; always returns `200 OK`.                                                 |

## Superset setup

1. Enable alerts/reports in Superset and configure a webhook notification method.
2. Set the webhook URL to your bridge: `http://<bridge-host>:8080/webhook`.
3. Trigger an alert — a formatted message should arrive in your Telegram chat.

For an attachment-free notification Superset sends a JSON body like this (an optional `header` metadata object may also be present; unknown fields are ignored):

```json
{
  "name": "Sales dropped below threshold",
  "text": "Daily revenue is below $10k.",
  "description": "Fires when the daily total drops.",
  "url": "https://superset.example.com/superset/dashboard/1/"
}
```

When a report has attachments, Superset sends `multipart/form-data` instead, and the bridge forwards the files to Telegram.

### Attachments

- **Routing by MIME type:** `image/png` → Telegram **photo**; everything else (CSV, PDF, …) → **document**.
- **Single vs album:** one file of a kind is sent with `sendPhoto`/`sendDocument`; two or more of the same kind are sent as a `sendMediaGroup` album. Photos and documents are delivered as separate messages.
- **Caption:** the rendered notification text (the same HTML body, including the **Open in Superset** link) becomes the caption, truncated to Telegram's 1024-character limit with the link always preserved.
- **Album limit:** Telegram caps albums at 10 items _per kind_; beyond that, the first 10 are forwarded and the rest are dropped with a warning log.

## Local playground

A self-contained Superset + bridge stack lives in [`playground/`](playground/). Seeded alerts fire every minute so you can inspect the real end-to-end payload — including a CSV-attachment alert:

```bash
cd playground
cp .env.example .env   # set TELEGRAM_TOKEN / TELEGRAM_CHAT_ID
docker compose up --build
```

See [`playground/README.md`](playground/README.md) for details.

## Development

Requires Go 1.26+.

```bash
go build ./...
go vet ./...
go test ./... -race
govulncheck ./...      # known-vulnerability scan; CI runs this on every push
```

## License

[MIT](LICENSE) © Denis Galeev
