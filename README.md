# PDF Service

A web service for generating PDF files from HTML.  

The service is implemented as a single Go web service that validates JWT bearer tokens, securely processes multipart uploads and invokes [WeasyPrint](https://weasyprint.readthedocs.io/en/stable/index.html) through a `bubblewrap` sandbox.

## Usage
See the [documentation](./docs/README.md) for usage.

## Service Deployment

The service runs as a single container exposing port `8080`.

Required environment variables:

- `AUTH_AUTHORITY` - OIDC authority URL (for example `https://login.bcc.no`)
- `AUTH_AUDIENCES` - comma-separated accepted token audiences

Optional environment variables:

- `AUTH_ISSUER` (default: `AUTH_AUTHORITY`)
- `AUTH_REQUIRED_SCOPE` (default: `pdf#create`)
- `MAX_REQUEST_BYTES` (default: `104857600`)
- `MAX_PART_BYTES` (default: `52428800`)
- `REQUEST_TIMEOUT_SECONDS` (default: `120`)
