FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/pdf-service ./cmd/pdfservice

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        bubblewrap \
        python3 \
        python3-pip \
        python3-setuptools \
        python3-wheel \
        libcairo2 \
        libpango-1.0-0 \
        libpangoft2-1.0-0 \
        libharfbuzz0b \
        libfontconfig1 \
        libgdk-pixbuf-2.0-0 \
        libffi8 \
        shared-mime-info \
        fonts-liberation \
        fonts-linuxlibertine \
    && python3 -m pip install --no-cache-dir --break-system-packages weasyprint \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --system --create-home --uid 10001 pdfservice

WORKDIR /app
COPY --from=builder /out/pdf-service /usr/local/bin/pdf-service

USER pdfservice

EXPOSE 8080

ENV PORT=8080 \
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin \
    AUTH_AUTHORITY=https://login.bcc.no \
    AUTH_AUDIENCE=api.bcc.no

ENTRYPOINT ["/usr/local/bin/pdf-service"]
