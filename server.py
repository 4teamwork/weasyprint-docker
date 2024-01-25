#!/usr/bin/env python3
"""
weasyprint server

A tiny aiohttp based web server that wraps weasyprint
It expects a multipart/form-data upload containing a html file, an optional
css file and optional attachments.
"""
from aiohttp import web
from urllib.parse import urlparse
from weasyprint import CSS
from weasyprint import default_url_fetcher
from weasyprint import HTML
from os import environ
from re import match
import logging
import os.path
import tempfile

CHUNK_SIZE = 65536

logger = logging.getLogger('weasyprint')

class Config:
    def __init__(self):
        self.items = {
            "allowed_url_pattern": os.environ.get('ALLOWED_URL_PATTERN', False)
        }

    def get(self, name):
        if name not in self.items:
            raise KeyError(f'Unknown configuration variable {name}')
        return self.items[name]
    
config = Config()

class URLFetcher:
    """URL fetcher that only allows data URLs and known files"""
    def __init__(self, valid_paths):
        self.valid_paths = valid_paths

    def __call__(self, url):
        if config.get('allowed_url_pattern') and match(config.get('allowed_url_pattern'), url):
          return default_url_fetcher(url)
    
        parsed = urlparse(url)

        if parsed.scheme == 'data':
            return default_url_fetcher(url)

        if parsed.scheme in ['', 'file'] and parsed.path:
            if os.path.abspath(parsed.path) in self.valid_paths:
                return default_url_fetcher(url)
            else:
                raise ValueError('Only known path allowed')

        raise ValueError('External resources are not allowed')


async def render_pdf(request):

    form_data = {}
    temp_dir = None

    if not request.content_type == 'multipart/form-data':
        logger.info(
            'Bad request. Received content type %s instead of multipart/form-data.',
            request.content_type,
        )
        return web.Response(status=400, text="Multipart request required.")

    reader = await request.multipart()

    with tempfile.TemporaryDirectory() as temp_dir:
        while True:
            part = await reader.next()

            if part is None:
                break

            if (
                part.name in ['html', 'css']
                or part.name.startswith('attachment.')
                or part.name.startswith('asset.')
            ):
                form_data[part.name] = await save_part_to_file(part, temp_dir)

        if 'html' not in form_data:
            logger.info('Bad request. No html file provided.')
            return web.Response(status=400, text="No html file provided.")

        html = HTML(filename=form_data['html'], url_fetcher=URLFetcher(form_data.values()))
        if 'css' in form_data:
            css = CSS(filename=form_data['css'], url_fetcher=URLFetcher(form_data.values()))
        else:
            css = CSS(string='@page { size: A4; margin: 2cm 2.5cm; }')

        attachments = [
            attachment for name, attachment in form_data.items()
            if name.startswith('attachment.')
        ]
        pdf_filename = os.path.join(temp_dir, 'output.pdf')

        try:
            html.write_pdf(
                pdf_filename, stylesheets=[css], attachments=attachments)
        except Exception:
            logger.exception('PDF generation failed')
            return web.Response(
                status=500, text="PDF generation failed.")
        else:
            return await stream_file(request, pdf_filename, 'application/pdf')


async def save_part_to_file(part, directory):
    filename = os.path.join(directory, part.filename)
    with open(filename, 'wb') as file_:
        while True:
            chunk = await part.read_chunk(CHUNK_SIZE)
            if not chunk:
                break
            file_.write(chunk)
    return filename


async def stream_file(request, filename, content_type):
    response = web.StreamResponse(
        status=200,
        reason='OK',
        headers={
            'Content-Type': content_type,
            'Content-Disposition':
            f'attachment; filename="{os.path.basename(filename)}"',
        },
    )
    await response.prepare(request)

    with open(filename, 'rb') as outfile:
        while True:
            data = outfile.read(CHUNK_SIZE)
            if not data:
                break
            await response.write(data)

    await response.write_eof()
    return response


async def healthcheck(request):
    return web.Response(status=200, text="OK")


if __name__ == '__main__':
    logging.basicConfig(
        format='%(asctime)s %(levelname)s %(name)s %(message)s',
        level=logging.INFO,
    )
    app = web.Application()
    app.add_routes([web.post('/', render_pdf)])
    app.add_routes([web.get('/healthcheck', healthcheck)])
    web.run_app(app)
