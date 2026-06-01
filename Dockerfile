# The binary is built by GoReleaser; this image only copies it in.
FROM gcr.io/distroless/static:nonroot
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/superset-telegram-bridge /app
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
