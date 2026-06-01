# Playground

A local Superset 6.1.0 + bridge stack. A seeded alert fires every minute and the
bridge forwards it to your Telegram chat — useful for inspecting the real Superset
webhook payload end-to-end.

## Run

1. `cp .env.example .env` and set `TELEGRAM_TOKEN` / `TELEGRAM_CHAT_ID`.
2. From this directory: `docker compose up --build`
3. Wait for `superset-seed` to finish. Within ~60s a Telegram message arrives.
4. Superset UI: http://localhost:8088 (admin / admin).

## Attachments

Two alerts fire every minute: the original text-only alert (JSON webhook) and a
CSV alert (`multipart/form-data` carrying `report.csv`, forwarded to Telegram as a
document). PNG/PDF attachments need a headless browser in the Superset image and
are intentionally left out to keep the stack light (tier B).

## Stop the pings

The alerts fire every minute. Disable them in **Settings → Alerts & Reports**, or
`docker compose down`.

## Inspect the raw payload

`docker compose logs -f bridge` shows each forwarded request.

## Capture the signature (Phase 4)

Uncomment `WEBHOOK_SECRET` in `superset_config.py` to make Superset add
`X-Webhook-Signature`; restart to see it on the bridge.
