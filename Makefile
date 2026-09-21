.PHONY: generate build run clean

generate:
	go tool oapi-codegen -config internal/transport/http/oapi-codegen.yaml api/openapi.yaml

build:
	go build -o bin main.go

run:
	go run main.go

test:
	go test -v ./...

clean:
	rm -rf bin/ gen.go