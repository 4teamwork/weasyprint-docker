# PDF Service

A web service for generating PDF files from HTML.

The service is implemented as a single Go web service that validates JWT bearer tokens, securely processes multipart uploads and invokes [WeasyPrint](https://weasyprint.readthedocs.io/en/stable/index.html) through a `bubblewrap` sandbox.

## Usage

See the [documentation](./docs/README.md) for usage.
See [architecture notes](./docs/architecture.md) for structure.

## Service Deployment

The service runs as a single container exposing port `8080`.

Required environment variables:

- `AUTH_AUTHORITY` - OIDC authority URL (for example `https://login.sandbox.bcc.no`)
- `AUTH_AUDIENCE` - accepted token audience (for example `sandbox-api.bcc.no`)

Optional environment variables:

- `PORT` (default: `8080`)
- `OTEL_SERVICE_NAME` (when set, enables OpenTelemetry tracing/logging exporter)

All other settings use sane built-in defaults in the service code.

## Local development

Use a local `.env` file so required settings are loaded automatically.

1. Create local env file:

```bash
cp .env.example .env
```

2. Start service with env auto-loading:

```bash
bash scripts/run-local.sh
```

The script reads `.env`, exports all values, and runs `go run ./cmd/pdfservice/main.go`.
It also validates `AUTH_AUTHORITY`, `AUTH_AUDIENCE`, `bwrap`, and `weasyprint` before startup.

## Common commands

```bash
make run
make test
make docker-build
make docker-up
make sample-pdf
```
