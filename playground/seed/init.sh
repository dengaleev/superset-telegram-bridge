#!/usr/bin/env bash
# One-shot: migrate the metadata DB, create the admin user, initialize roles.
set -euo pipefail

superset db upgrade
superset fab create-admin \
  --username admin --firstname Superset --lastname Admin \
  --email admin@superset.local --password admin || true
superset init
echo "superset init complete"
