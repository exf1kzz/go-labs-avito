.PHONY: generate run test test-race start stop status connect \
	migrate migrate-down migrate-down-all migrate-status

generate:
	mkdir -p internal/generated
	go tool oapi-codegen \
		-config oapi-codegen.yaml \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

run:
	set -a; . ./.env; set +a; go run ./cmd/trip-service

test:
	go test ./...

test-race:
	go test -race ./...

start:
	tripgoctl environment start

stop:
	tripgoctl environment stop

status:
	tripgoctl environment status

connect:
	tripgoctl connect

migrate:
	set -a; . ./.env; set +a; go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	set -a; . ./.env; set +a; go tool goose -dir migrations postgres "$$DATABASE_URL" down

migrate-down-all:
	set -a; . ./.env; set +a; go tool goose -dir migrations postgres "$$DATABASE_URL" down-to 0

migrate-status:
	set -a; . ./.env; set +a; go tool goose -dir migrations postgres "$$DATABASE_URL" status