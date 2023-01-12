# PDF Service

A web service for generating PDF files from HTML.  

The service is based on a fork from https://github.com/4teamwork/weasyprint-docker which uses [WeasyPrint](https://weasyprint.readthedocs.io/en/stable/index.html).

## Usage
See the [documentation](./docs/README.md) for usage.

## Service Deployment

The service runs as an Azure Container App containing two containers:

1. Ingress [reverse proxy](/proxy) - .net application which handles authentication and forwards authenticated requests to the weasyprint service. This container is public on port 443.  

2. [Weasyprint server](server.py) - python web service which passes requests to the weasyprint library for processing. This container runs as a sidecar on port 8080
