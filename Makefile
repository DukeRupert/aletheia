.PHONY: build
build:
	go build -o bin/myapp ./cmd/myapp

.PHONY: dev
dev:
	go run ./cmd/main.go

.PHONY: migrate-up
migrate-up:
	goose up

.PHONY: migrate-down
migrate-down:
	goose down
