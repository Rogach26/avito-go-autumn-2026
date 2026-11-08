SHELL := /bin/bash

.PHONY: generate migrate migrate-down migrate-status run build test test-integration lint

COVERAGE_FILE ?= /tmp/trip-service-coverage.out

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
	go generate ./...

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
	go test -coverprofile=$(COVERAGE_FILE) ./internal/domain/... ./internal/usecase/...
	go tool cover -func=$(COVERAGE_FILE)

test-integration:
	@$(load_env) go test -race -count=2 -tags=integration ./internal/repository/...

lint:
	golangci-lint run
