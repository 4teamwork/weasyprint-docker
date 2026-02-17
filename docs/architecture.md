# Architecture

## Overview

The PDF service runs as a single Go application that:

1. Validates JWT bearer tokens from `AUTH_AUTHORITY` against OIDC JWKS.
2. Requires `pdf#create` scope.
3. Accepts `multipart/form-data` on `POST /`.
4. Persists request files to a temporary directory.
5. Invokes `weasyprint` through `bubblewrap` (`bwrap`) sandbox.
6. Streams generated PDF back as HTTP response.

`GET /healthcheck` returns `200 OK`.

## Request contract

Supported multipart fields:

- `html` (required)
- `css` (optional)
- `attachment.*` (optional)
- `asset.*` (optional)
- `file.*` (optional; backwards compatible attachment alias)

## Runtime dependencies

- `bwrap` (bubblewrap)
- `weasyprint`
- Fonts and native libs required by WeasyPrint

## Configuration

The service intentionally exposes a minimal env surface:

- `PORT` (optional, default `8080`)
- `AUTH_AUTHORITY` (required)
- `AUTH_AUDIENCE` (required)
- `OTEL_SERVICE_NAME` (optional; enables real OpenTelemetry provider when set, otherwise mock/local observability is used)

All other settings are hardcoded defaults in code.
