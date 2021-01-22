#!/usr/bin/env python3
"""
weasyprint server

A tiny aiohttp based web server that wraps weasyprint
It expects a multipart/form-data upload containing a html file, an optional
css file and optional attachments.
"""
from aiohttp import web
from weasyprint import HTML, CSS
import logging
import os.path
import tempfile

CHUNK_SIZE = 65536

logger = logging.getLogger('weasyprint')


async def render_pdf(request):

    form_data = {}
    temp_dir = None

    if not request.content_type == 'multipart/form-data':
        logger.info(
            'Bad request. Received content type %s instead of multipart/form-data.',
            request.content_type,
        )
        return web.Response(status=400, text=f"Multipart request required.")

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
            return web.Response(status=400, text=f"No html file provided.")

        html = HTML(filename=form_data['html'])
        if 'css' in form_data:
            css = CSS(filename=form_data['css'])
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
    with open(os.path.join(directory, filename), 'wb') as file_:
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
    return web.Response(status=200, text=f"OK")


if __name__ == '__main__':
    logging.basicConfig(
        format='%(asctime)s %(levelname)s %(name)s %(message)s',
        level=logging.INFO,
    )
    app = web.Application()
    app.add_routes([web.post('/', render_pdf)])
    app.add_routes([web.get('/healthcheck', healthcheck)])
    web.run_app(app)
