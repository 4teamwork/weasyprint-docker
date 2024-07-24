# weasyprint

A dockerized web service for creating PDFs from HTML using WeasyPrint

## Description

[WeasyPrint](https://weasyprint.readthedocs.io/en/stable/index.html)
is a visual rendering engine for HTML and CSS that can export to PDF.
It aims to support web standards for printing.

This web service exposes an endpoint for uploading a html file, an optional css
file and optional attachments. It responds with the generated PDF.

The web service is written in Python using the aiohttp web server.

## Usage

To start the webservice just run
```
docker-compose up
```

The html file and and any additional files must be uploaded as multipart/form-data
with a part named `html` containing the main HTML content. Assets (e.g. images)
that are referenced in HTML must be prefixed with `asset.`. Files that should
be attached to the generated pdf must be prefixed with `attachment.`

Example:

```
curl -F "html=@tests/index.html" -F "asset.universe.jpg=@tests/universe.jpg" http://localhost:3000 -o test.pdf
```

### Options

It's possible to overwrite some [weasyprint.DEFAULT_OPTIONS](https://doc.courtbouillon.org/weasyprint/stable/api_reference.html#weasyprint.DEFAULT_OPTIONS) by including a part named `options` in JSON format.

Example:

```
curl -F "html=@tests/index.html" -F "asset.universe.jpg=@tests/universe.jpg" -F options='{"pdf_variant": "pdf/a-3b"};type=application/json' http://localhost:3000 -o test.pdf
```

### Configuration

By default fetching external resource is prohibited for security reasons.
To allow fetching resources from external urls, set the environment variable
``WEASYPRINT_ALLOWED_URLS_PATTERN`` to a regex pattern that matches your
desired urls.

Example:

```
WEASYPRINT_ALLOWED_URLS_PATTERN="^https://fonts\.example\.com.*|^https://logos\.example\.com.*"
```