.PHONY: build
build:
	go build -o bin/myapp ./cmd/myapp

.PHONY: dev
dev:
	go run ./cmd/main.go 