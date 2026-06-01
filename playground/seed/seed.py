"""Idempotently create a database connection and an always-true webhook alert.

Runs inside the Superset image (uses its bundled `requests`). Talks to the
Superset REST API, so it must run after the web service is healthy.
"""
import json
import os
import sys
import time

import requests

BASE = os.environ.get("SUPERSET_BASE_URL", "http://superset:8088")
USER = "admin"
PASSWORD = "admin"
DB_NAME = "playground"
DASHBOARD_NAME = "playground-dashboard"
DATASET_NAME = "playground_numbers"
CHART_NAME = "playground-chart"
ALERT_NAME = "playground-webhook-alert"
# (alert name, report_format) — each fires an attachment to exercise a bridge path:
# CSV/PDF -> Telegram document, PNG -> Telegram photo. PNG/PDF are screenshots, so
# they need the headless Firefox built into the playground's Superset image.
ATTACHMENT_ALERTS = [
    ("playground-csv-alert", "CSV"),
    ("playground-png-alert", "PNG"),
    ("playground-pdf-alert", "PDF"),
]
WEBHOOK_TARGET = os.environ.get("WEBHOOK_TARGET", "http://bridge:8080/webhook")
# The alert queries this DB; the metadata Postgres is already there, so reuse it.
DB_URI = "postgresql+psycopg2://superset:superset@postgres:5432/superset"


def wait_for_health(timeout=180):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            if requests.get(f"{BASE}/health", timeout=5).ok:
                return
        except requests.RequestException:
            pass
        time.sleep(3)
    sys.exit("superset did not become healthy in time")


def session():
    s = requests.Session()
    token = s.post(
        f"{BASE}/api/v1/security/login",
        json={"username": USER, "password": PASSWORD, "provider": "db", "refresh": True},
        timeout=10,
    ).json()["access_token"]
    s.headers["Authorization"] = f"Bearer {token}"
    s.headers["X-CSRFToken"] = s.get(f"{BASE}/api/v1/security/csrf_token/", timeout=10).json()["result"]
    s.headers["Referer"] = BASE
    return s


def find_id(s, resource, col, value):
    r = s.get(
        f"{BASE}/api/v1/{resource}/",
        params={"q": f"(filters:!((col:{col},opr:eq,value:'{value}')))"},
        timeout=10,
    )
    r.raise_for_status()
    ids = r.json().get("ids", [])
    return ids[0] if ids else None


def ensure_database(s):
    existing = find_id(s, "database", "database_name", DB_NAME)
    if existing:
        return existing
    r = s.post(
        f"{BASE}/api/v1/database/",
        json={"database_name": DB_NAME, "sqlalchemy_uri": DB_URI},
        timeout=10,
    )
    r.raise_for_status()
    return r.json()["id"]


def ensure_dashboard(s):
    # An alert must reference a chart or dashboard; an empty dashboard is the
    # cheapest way to satisfy that (no dataset/chart needed).
    existing = find_id(s, "dashboard", "dashboard_title", DASHBOARD_NAME)
    if existing:
        return existing
    r = s.post(f"{BASE}/api/v1/dashboard/", json={"dashboard_title": DASHBOARD_NAME}, timeout=10)
    r.raise_for_status()
    return r.json()["id"]


def ensure_alert(s, database_id, dashboard_id):
    if find_id(s, "report", "name", ALERT_NAME):
        print("alert already exists; nothing to do")
        return
    r = s.post(
        f"{BASE}/api/v1/report/",
        json={
            "type": "Alert",
            "name": ALERT_NAME,
            "description": "Always-true alert that POSTs a text webhook to the bridge.",
            "active": True,
            "crontab": "* * * * *",
            "database": database_id,
            "sql": "SELECT 1 AS value",
            "validator_type": "operator",
            # Superset 6.1.0 expects these as JSON objects, not JSON-encoded strings.
            "validator_config_json": {"op": ">", "threshold": 0},
            # TEXT format => no screenshot/CSV => the notifier POSTs JSON, not multipart.
            "dashboard": dashboard_id,
            "report_format": "TEXT",
            "recipients": [{
                "type": "Webhook",
                "recipient_config_json": {"target": WEBHOOK_TARGET},
            }],
            "working_timeout": 60,
            "grace_period": 60,
        },
        timeout=10,
    )
    if not r.ok:
        sys.exit(f"alert creation failed ({r.status_code}): {r.text}")
    print("alert created")


def ensure_dataset(s, database_id):
    existing = find_id(s, "dataset", "table_name", DATASET_NAME)
    if existing:
        return existing
    r = s.post(f"{BASE}/api/v1/dataset/", json={
        "database": database_id,
        "schema": "public",
        "table_name": DATASET_NAME,
        "sql": "SELECT 1 AS value",
    }, timeout=10)
    r.raise_for_status()
    return r.json()["id"]


def ensure_chart(s, dataset_id, chart_name):
    existing = find_id(s, "chart", "slice_name", chart_name)
    if existing:
        return existing
    # The CSV alert exports the chart's data, which needs a saved query_context;
    # the PNG/PDF alerts ignore it and screenshot the rendered chart instead.
    query_context = json.dumps({
        "datasource": {"id": dataset_id, "type": "table"},
        "force": False,
        "queries": [{
            "columns": ["value"],
            "metrics": [],
            "orderby": [],
            "row_limit": 1000,
            "filters": [],
            "extras": {"having": "", "where": ""},
        }],
        "result_format": "csv",
        "result_type": "full",
    })
    r = s.post(f"{BASE}/api/v1/chart/", json={
        "slice_name": chart_name,
        "viz_type": "table",
        "datasource_id": dataset_id,
        "datasource_type": "table",
        "params": json.dumps({"viz_type": "table", "query_mode": "raw", "all_columns": ["value"]}),
        "query_context": query_context,
    }, timeout=10)
    r.raise_for_status()
    return r.json()["id"]


def ensure_attachment_alert(s, database_id, chart_id, name, report_format):
    if find_id(s, "report", "name", name):
        print(f"{name} already exists; nothing to do")
        return
    r = s.post(f"{BASE}/api/v1/report/", json={
        "type": "Alert",
        "name": name,
        "description": f"Always-true alert that POSTs a {report_format} attachment to the bridge.",
        "active": True,
        "crontab": "* * * * *",
        "database": database_id,
        "sql": "SELECT 1 AS value",
        "validator_type": "operator",
        "validator_config_json": {"op": ">", "threshold": 0},
        # A chart + non-TEXT format => the notifier POSTs multipart/form-data.
        "chart": chart_id,
        "report_format": report_format,
        "recipients": [{"type": "Webhook", "recipient_config_json": {"target": WEBHOOK_TARGET}}],
        "working_timeout": 60,
        "grace_period": 60,
    }, timeout=10)
    if not r.ok:
        sys.exit(f"{name} creation failed ({r.status_code}): {r.text}")
    print(f"{name} created")


def main():
    wait_for_health()
    s = session()
    database_id = ensure_database(s)
    dataset_id = ensure_dataset(s, database_id)
    ensure_alert(s, database_id, ensure_dashboard(s))  # text-only: JSON webhook
    # Each attachment alert gets its own chart (Superset allows one report per chart).
    for name, report_format in ATTACHMENT_ALERTS:
        chart_id = ensure_chart(s, dataset_id, f"{CHART_NAME}-{report_format.lower()}")
        ensure_attachment_alert(s, database_id, chart_id, name, report_format)


if __name__ == "__main__":
    main()
