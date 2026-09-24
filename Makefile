.PHONY: generate test test-race

generate:
	mkdir -p internal/generated
	go tool oapi-codegen \
		-config oapi-codegen.yaml \
		-o internal/generated/api.gen.go \
		contracts/openapi/trip-service.openapi.yaml

test:
	go test ./...

test-race:
	go test -race ./...
