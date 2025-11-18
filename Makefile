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

.PHONY: sqlc-generate
sqlc-generate:
	sqlc generate

.PHONY: test
test:
	go test ./... -v

.PHONY: test-coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
