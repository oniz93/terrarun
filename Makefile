.PHONY: dev test lint build deploy migrate-up migrate-down logs clean

# Development
dev:
	docker compose up --build

test:
	cd backend && go test ./... -v -count=1

lint:
	cd backend && golangci-lint run ./...
	cd client && flutter analyze

build:
	cd backend && go build -o bin/api ./cmd/api
	cd backend && go build -o bin/migrate ./cmd/migrate

# Database
migrate-up:
	cd backend && go run ./cmd/migrate up

migrate-down:
	cd backend && go run ./cmd/migrate down

# Deployment
deploy:
	./deploy.sh

# Utilities
logs:
	docker compose logs -f

clean:
	docker compose down -v
	rm -rf backend/bin client/build
