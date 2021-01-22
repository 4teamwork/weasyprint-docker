FROM alpine:3.13 as builder

# Needs dependencies form edge/community
# RUN apk add --no-cache --repository http://dl-cdn.alpinelinux.org/alpine/edge/testing \
#   py3-weasyprint \
#   py3-aiohttp

RUN apk add --no-cache \
    gcc \
    musl-dev \
    jpeg-dev \
    zlib-dev \
    libffi-dev \
    cairo-dev \
    pango-dev \
    gdk-pixbuf-dev \
    python3 \
    py3-pip \
    py3-cffi \
    py3-pillow \
    py3-lxml

RUN pip3 install weasyprint


FROM alpine:3.13

RUN addgroup --system weasyprint \
     && adduser --system --ingroup weasyprint weasyprint

RUN apk add --no-cache \
    jpeg \
    zlib \
    libimagequant \
    lcms2 \
    openjpeg \
    libwebp \
    libffi \
    cairo \
    pango \
    gdk-pixbuf \
    ttf-liberation \
    ttf-linux-libertine \
    python3 \
    py3-aiohttp

COPY --from=builder /usr/lib/python3.8/site-packages/ /usr/lib/python3.8/site-packages/

ENV PYTHONUNBUFFERED 1
WORKDIR /app
USER weasyprint

EXPOSE 8080

COPY server.py .
CMD ["python3", "server.py"]