# Playground

A local Superset 6.1.0 + bridge stack. Seeded alerts fire every minute and the
bridge forwards them to your Telegram chat — handy for seeing the real Superset
webhook payload end-to-end.

## Run

1. `cp .env.example .env` and set `TELEGRAM_TOKEN` / `TELEGRAM_CHAT_ID`.
2. From this directory: `docker compose up --build`.
3. Wait for `superset-seed` to finish. The text and CSV alerts arrive within ~60s;
   the PNG/PDF screenshots follow a few seconds later.
4. Superset UI: http://localhost:8088 (admin / admin).

## What fires

Four alerts run every minute, one per bridge path:

- text → JSON webhook → Telegram **message**;
- PNG → screenshot → Telegram **photo**;
- CSV → `multipart/form-data` → Telegram **document**;
- PDF → screenshot → Telegram **document**.

The PNG and PDF alerts screenshot the chart, so the playground builds its own
Superset image with headless Firefox + geckodriver (see `superset.Dockerfile`).

## Inspect the payload

The bridge runs at `LOG_LEVEL=debug` here, so `docker compose logs -f bridge`
prints each inbound payload — content type, text fields, and any attachment
names, types, and sizes.

## Stop the pings

Disable the alerts in **Settings → Alerts & Reports**, or `docker compose down`.
