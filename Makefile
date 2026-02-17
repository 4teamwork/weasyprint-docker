SHELL := /bin/bash

.PHONY: run test docker-build docker-up docker-down sample-pdf

run:
	bash scripts/run-local.sh

test:
	go test ./...

docker-build:
	docker compose build --no-cache

docker-up:
	docker compose up --build

docker-down:
	docker compose down

sample-pdf:
	cd samples && ./generate_sample_pdf.sh
