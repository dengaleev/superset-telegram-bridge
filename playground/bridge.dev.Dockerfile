# Build the bridge from source (the production Dockerfile is copy-only for GoReleaser).
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bridge ./cmd/superset-telegram-bridge

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bridge /app
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
