.PHONY: generate build run clean

generate:
	go tool oapi-codegen -config internal/transport/http/oapi-codegen.yaml api/openapi.yaml

build: generate
	docker compose build

run:
	docker compose up --build

test:
	go test -v ./...

clean:
	rm -rf bin/ gen.go