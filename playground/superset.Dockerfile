# Superset 6.1.0 ships without DB drivers; add the Postgres one for the metadata DB.
# The image's venv (/app/.venv) is uv-managed and pip-less, so a bare `pip install`
# lands in the wrong interpreter — install via uv targeting the venv's python.
FROM apache/superset:6.1.0
USER root
RUN uv pip install --python /app/.venv/bin/python --no-cache psycopg2-binary
USER superset
