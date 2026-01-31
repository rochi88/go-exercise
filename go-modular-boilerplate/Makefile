SHELL := /bin/zsh

.PHONY: run dev test build tidy migrate-up migrate-down migrate-status migrate-version migrate-reset

run:
	go run cmd/api/main.go

dev:
	air

build:
	go build -o bin/api cmd/api/main.go

test:
	go test ./... -v

tidy:
	go mod tidy

# Migration commands
migrate-up:
	go run cmd/migrate/main.go up

migrate-down:
	go run cmd/migrate/main.go down $(n)

migrate-status:
	go run cmd/migrate/main.go status

migrate-version:
	go run cmd/migrate/main.go version

migrate-reset:
	go run cmd/migrate/main.go reset

migrate-force:
	go run cmd/migrate/main.go force $(v)
