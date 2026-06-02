# superset-telegram-bridge

[![CI](https://github.com/dengaleev/superset-telegram-bridge/actions/workflows/ci.yml/badge.svg)](https://github.com/dengaleev/superset-telegram-bridge/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dengaleev/superset-telegram-bridge)](https://goreportcard.com/report/github.com/dengaleev/superset-telegram-bridge)
[![Go Version](https://img.shields.io/github/go-mod/go-version/dengaleev/superset-telegram-bridge)](go.mod)
[![Release](https://img.shields.io/github/v/release/dengaleev/superset-telegram-bridge?sort=semver)](https://github.com/dengaleev/superset-telegram-bridge/releases)
[![License: MIT](https://img.shields.io/github/license/dengaleev/superset-telegram-bridge)](LICENSE)

A single-binary service that forwards [Apache Superset](https://superset.apache.org/) alert and report notifications to a Telegram chat.

Superset can POST notifications to a webhook URL. This bridge receives that webhook, turns it into a Telegram message with an inline "Open in Superset" link, forwards any attachments as photos or documents, and sends it through the Telegram Bot API.

```
  Superset alert/report
      │
      │  POST /webhook   (JSON, or multipart/form-data with attachments)
      ▼
  superset-telegram-bridge
      │
      │  Telegram Bot API
      │      text        → sendMessage   (HTML + "Open in Superset" link)
      │      PNG         → sendPhoto
      │      CSV, PDF    → sendDocument
      ▼
  Telegram chat
```

## Contents

- [What it does](#what-it-does)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Endpoints](#endpoints)
- [Superset setup](#superset-setup)
- [Security](#security)
- [Local playground](#local-playground)
- [Development](#development)
- [Releasing](#releasing)

## What it does

- Formats text notifications as HTML Telegram messages — bold title, italic description, an inline "Open in Superset" link, link previews off.
- Forwards report attachments: `image/png` files go as photos, other files (CSV, PDF, …) as documents. Several files of one kind are grouped into an album, with the notification text as the caption.
- Retries a failed send once after a short fixed delay; a non-2xx response is returned as-is. The bot token is stripped from transport errors before they're logged.
- Logs JSON via `slog`, serves `GET /healthz`, shuts down gracefully, and caps the request body at 50 MiB.
- Builds to a static, non-root distroless image (under 10 MB) for `linux/amd64` and `linux/arm64`, published to GHCR on each release.

## Quick start

With `docker run`:

```bash
docker run -d --name superset-bridge \
  -p 8080:8080 \
  -e TELEGRAM_TOKEN="123456:your-bot-token" \
  -e TELEGRAM_CHAT_ID="-1001234567890" \
  ghcr.io/dengaleev/superset-telegram-bridge:latest
```

Or with Docker Compose (`compose.yaml`):

```yaml
services:
  bridge:
    image: ghcr.io/dengaleev/superset-telegram-bridge:latest
    ports:
      - "8080:8080"
    environment:
      TELEGRAM_TOKEN: "123456:your-bot-token"
      TELEGRAM_CHAT_ID: "-1001234567890"
    restart: unless-stopped
```

```bash
docker compose up -d
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

## Security

The bridge has no authentication on `/webhook`; it trusts its network.

- Run it next to Superset — a sidecar or a container on the same private network — and let Superset reach it there (e.g. `http://bridge:8080/webhook`).
- Don't expose it on a public or shared network.

There's no token/basic-auth option, on purpose: Superset can only put a secret in the webhook URL, which is stored in the alert config and visible to anyone who can edit alerts, so it wouldn't keep anyone out.

Superset doesn't currently restrict which hosts a webhook can target; the open PR [apache/superset#39301](https://github.com/apache/superset/pull/39301) would validate targets and block private ranges.

## Local playground

A self-contained Superset + bridge stack lives in [`playground/`](playground/). Seeded alerts fire every minute — text plus PNG, CSV, and PDF attachments — so you can inspect the real end-to-end payload across every path:

```bash
cd playground
cp .env.example .env   # set TELEGRAM_TOKEN / TELEGRAM_CHAT_ID
docker compose up --build
```

See [`playground/README.md`](playground/README.md) for details.

## Development

Requires Go 1.26+. CI runs each of these on every push.

```bash
go build ./...
go vet ./...
go test ./... -race
golangci-lint run ./...   # lint + format check; config in .golangci.yml
govulncheck ./...         # known-vulnerability scan
```

Mocks are generated with [mockery](https://vektra.github.io/mockery/) (`mockery`; config in `.mockery.yaml`).

## Releasing

Push a semver tag — GoReleaser builds the binaries and publishes the GitHub
release (with changelog and checksums) and the GHCR image:

```bash
git tag -a v1.2.3 -m "v1.2.3" && git push origin v1.2.3
```

Preview a release locally without publishing: `goreleaser release --snapshot --clean`.

## License

[MIT](LICENSE) © Denis Galeev
