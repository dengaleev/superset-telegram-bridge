"""Minimal Superset config for the playground (dev only)."""
import os

SECRET_KEY = os.environ.get("SUPERSET_SECRET_KEY", "playground-dev-secret-change-me")
SQLALCHEMY_DATABASE_URI = "postgresql+psycopg2://superset:superset@postgres:5432/superset"

# ALERT_REPORTS enables Alerts & Reports; ALERT_REPORT_WEBHOOK enables the
# Webhook notification method (the notifier refuses to send without it);
# ALERTS_ATTACH_REPORTS lets alerts (not just reports) attach files like the CSV.
FEATURE_FLAGS = {"ALERT_REPORTS": True, "ALERT_REPORT_WEBHOOK": True, "ALERTS_ATTACH_REPORTS": True}

# Allow the plain-HTTP bridge URL (defaults to HTTPS-only).
ALERT_REPORTS_WEBHOOK_HTTPS_ONLY = False

# The worker fetches CSV data over HTTP from this base URL; point it at the web
# service (the default 0.0.0.0:8080 is unreachable from the worker container).
WEBDRIVER_BASEURL = "http://superset:8088/"

_REDIS_URL = "redis://redis:6379/0"


class CeleryConfig:
    broker_url = _REDIS_URL
    result_backend = _REDIS_URL
    imports = ("superset.sql_lab", "superset.tasks.scheduler")
    # Beat enqueues due alerts/reports every minute; that tick is what fires our alert.
    beat_schedule = {
        "reports.scheduler": {"task": "reports.scheduler", "schedule": 60.0},
        "reports.prune_log": {"task": "reports.prune_log", "schedule": 3600.0},
    }


CELERY_CONFIG = CeleryConfig

# Uncomment to make Superset sign the webhook (X-Webhook-Signature) for Phase 3 capture.
# WEBHOOK_SECRET = "playground-webhook-secret"
