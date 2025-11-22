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

# Docker commands
.PHONY: docker-build
docker-build:
	docker build -t aletheia:latest .

.PHONY: docker-build-tag
docker-build-tag:
	@if [ -z "$(TAG)" ]; then echo "Usage: make docker-build-tag TAG=v1.0.0"; exit 1; fi
	docker build -t aletheia:$(TAG) -t aletheia:latest .

.PHONY: docker-run
docker-run:
	docker run --rm -p 1323:1323 --env-file .env aletheia:latest

.PHONY: docker-push
docker-push:
	@if [ -z "$(DOCKER_USERNAME)" ]; then echo "Usage: make docker-push DOCKER_USERNAME=youruser [TAG=latest]"; exit 1; fi
	@TAG=$${TAG:-latest}; \
	docker tag aletheia:$$TAG $(DOCKER_USERNAME)/aletheia:$$TAG && \
	docker push $(DOCKER_USERNAME)/aletheia:$$TAG

.PHONY: docker-compose-up
docker-compose-up:
	docker compose -f docker-compose.prod.yml --env-file .env.prod up -d

.PHONY: docker-compose-down
docker-compose-down:
	docker compose -f docker-compose.prod.yml down

.PHONY: docker-compose-logs
docker-compose-logs:
	docker compose -f docker-compose.prod.yml logs -f

.PHONY: docker-compose-restart
docker-compose-restart:
	docker compose -f docker-compose.prod.yml restart

# Deployment helpers
.PHONY: deploy-files
deploy-files:
	@if [ ! -f .env.prod ]; then \
		echo "❌ Error: .env.prod not found. Create it first:"; \
		echo "   cp example.env .env.prod"; \
		echo "   # Edit .env.prod with production values"; \
		exit 1; \
	fi
	./deploy.sh

.PHONY: deploy-ssh
deploy-ssh:
	ssh dukerupert@angmar.dev "cd /home/dukerupert/aletheia && bash"
