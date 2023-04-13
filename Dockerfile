FROM alpine:3.17 as pkg-builder

RUN apk -U add \
    sudo \
    alpine-sdk \
    apkbuild-pypi

RUN mkdir -p /var/cache/distfiles && \
    adduser -D packager && \
    addgroup packager abuild && \
    chgrp abuild /var/cache/distfiles && \
    chmod g+w /var/cache/distfiles && \
    echo "packager ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

WORKDIR /work
USER packager

RUN abuild-keygen -a -i -n

COPY --chown=packager:packager packages/ ./

RUN cd py3-pydyf && \
    abuild -r && \
    cd ../py3-weasyprint && \
    abuild -r


FROM alpine:3.17

RUN addgroup --system weasyprint \
     && adduser --system --ingroup weasyprint weasyprint

COPY --from=pkg-builder /home/packager/packages/work/ /packages/
COPY --from=pkg-builder /home/packager/.abuild/*.pub /etc/apk/keys/

RUN apk add --no-cache --repository /packages \
    font-liberation \
    font-liberation-sans-narrow \
    ttf-linux-libertine \
    python3 \
    py3-aiohttp \
    py3-weasyprint

ENV PYTHONUNBUFFERED 1
WORKDIR /app
USER weasyprint

EXPOSE 8080

COPY server.py .
CMD ["python3", "server.py"]
