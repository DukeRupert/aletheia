.PHONY: build
build:
	go build -o bin/aletheia ./cmd

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

.PHONY: test-cleanup
test-cleanup:
	@echo "Cleaning up test data from database..."
	@docker exec -i aletheia-db psql -U postgres -d postgres -c "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE email LIKE '%@example.com' OR email LIKE '%test%');" > /dev/null 2>&1 || true
	@docker exec -i aletheia-db psql -U postgres -d postgres -c "DELETE FROM users WHERE email LIKE '%@example.com' OR email LIKE '%test%';" > /dev/null 2>&1 || true
	@echo "Test cleanup complete"

.PHONY: test-handlers
test-handlers:
	@echo "Running handler tests..."
	@go test ./internal/handlers/ -v
