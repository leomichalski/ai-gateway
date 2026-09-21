.PHONY: generate build run clean

generate:
	go tool oapi-codegen -config api/oapi-codegen.yaml api/openapi.yaml

build:
	go build -o bin/server cmd/server/main.go

run:
	go run main.go

test:
	go test -v ./api/...

clean:
	rm -rf bin/ gen.go