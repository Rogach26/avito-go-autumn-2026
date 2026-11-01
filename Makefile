SHELL := /bin/bash

.PHONY: generate migrate migrate-down migrate-status run build test

define load_env
set -a; source .env.example || exit 1; source .env || exit 1; set +a;
endef

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-include-operation-ids createTrip,getTrip,finishTrip,health,ready \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

migrate:
	@$(load_env) go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	@$(load_env) go tool goose -dir migrations postgres "$$DATABASE_URL" down

migrate-status:
	@$(load_env) go tool goose -dir migrations postgres "$$DATABASE_URL" status

run:
	@$(load_env) go run ./cmd/trip-service

build:
	go build ./...

test:
	go test -race -count=2 ./...
