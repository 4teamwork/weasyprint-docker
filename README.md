# PDF Server

An API for generating PDF files.

The API is based on a fork from https://github.com/4teamwork/weasyprint-docker (based on WeasyPrint)

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
with a part named `html` containing the main HTML content. Additional files
have to be in parts with names that start with the preifx `file.`.

Example:

```
curl -F "html=@tests/test.html" http://localhost:3000 -o test.pdf
```
