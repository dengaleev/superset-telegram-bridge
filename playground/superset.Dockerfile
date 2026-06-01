FROM apache/superset:6.1.0
USER root

# psycopg2-binary — Postgres driver for the metadata DB (the base image ships none).
# The venv is uv-managed and pip-less, so install via uv into the venv's python.
RUN uv pip install --python /app/.venv/bin/python --no-cache psycopg2-binary

# Firefox + geckodriver so the PNG/PDF alerts can screenshot the chart via Selenium
# (Superset's default driver, and the lightest real browser that runs on arm64).
# geckodriver isn't packaged, so fetch the release matching the target arch.
ARG TARGETARCH
RUN apt-get update \
    && apt-get install --no-install-recommends -y firefox-esr \
    && rm -rf /var/lib/apt/lists/* \
    && case "$TARGETARCH" in amd64) GA=linux64 ;; arm64) GA=linux-aarch64 ;; esac \
    && curl -fsSL "https://github.com/mozilla/geckodriver/releases/download/v0.35.0/geckodriver-v0.35.0-$GA.tar.gz" \
       | tar -xz -C /usr/local/bin

USER superset
