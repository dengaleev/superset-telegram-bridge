"""Minimal Superset config for the playground (dev only)."""
import os

SECRET_KEY = os.environ.get("SUPERSET_SECRET_KEY", "playground-dev-secret-change-me")
SQLALCHEMY_DATABASE_URI = "postgresql+psycopg2://superset:superset@postgres:5432/superset"

FEATURE_FLAGS = {
    "ALERT_REPORTS": True,                  # Alerts & Reports
    "ALERT_REPORT_WEBHOOK": True,           # the Webhook notification method (required, or it won't send)
    "ALERTS_ATTACH_REPORTS": True,          # let alerts (not just reports) attach files (CSV/PNG/PDF)
}

# Allow the plain-HTTP bridge URL (defaults to HTTPS-only).
ALERT_REPORTS_WEBHOOK_HTTPS_ONLY = False

# The worker fetches CSV data over HTTP from this base URL; point it at the web
# service (the default 0.0.0.0:8080 is unreachable from the worker container).
WEBDRIVER_BASEURL = "http://superset:8088/"

# PNG/PDF alerts screenshot the chart via Selenium + headless Firefox. Point the
# service at our geckodriver so Selenium doesn't try to fetch one over the network;
# geckodriver then locates /usr/bin/firefox itself, so no binary_location is needed.
WEBDRIVER_TYPE = "firefox"
WEBDRIVER_OPTION_ARGS = ["--headless"]
WEBDRIVER_CONFIGURATION = {"service": {"executable_path": "/usr/local/bin/geckodriver"}}

# Headless Firefox cold-renders the React chart slowly here; give it generous
# windows so the screenshot doesn't time out before the chart appears.
SCREENSHOT_LOCATE_WAIT = 60
SCREENSHOT_LOAD_WAIT = 120

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
