include .env
export DB_DSN

.PHONY: up down migrate run

up:
	docker-compose up -d

down:
	docker-compose down

restart:
	docker-compose down
	docker-compose up -d

migrate:
	goose -dir ./migrations postgres "$(DB_DSN)" up

run:
	go run ./cmd/main.go

