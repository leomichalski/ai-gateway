.PHONY: generate build run clean

generate:
	go tool oapi-codegen -config internal/transport/http/oapi-codegen.yaml api/openapi.yaml

build: generate
	docker compose build

run:
	docker compose up --build

test:
	@echo "ATTENTION: To pass all E2E tests, export the corresponding LLM Provider API key e.g"
	@echo "export GOOGLE_API_KEY=***"
	@echo ""
	@echo ""
	go test -v ./...

clean:
	rm -rf bin/ gen.go