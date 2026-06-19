-include .env
export

.PHONY: run build docker-up docker-down migrate-up migrate-down

run:
	go run ./cmd/marshal

build:
	go build -o bin/marshal ./cmd/marshal

docker-up:
	docker-compose -f ../docker-compose.yaml up -d

docker-down:
	docker-compose -f ../docker-compose.yaml down

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

.DEFAULT_GOAL := run
