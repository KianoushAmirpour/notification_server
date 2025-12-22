include .env
export DB_DSN

.PHONY: up down migrate run create restart swag down_volumes

up:
	docker-compose up -d

down:
	docker-compose down

down_volumes:
	docker-compose down -v

restart:
	docker-compose down
	docker-compose up -d

migrate:
	goose -dir ./migrations postgres "$(DB_DSN)" up

create:
	$(if $(NAME),,$(error NAME is not set. Usage: make migration NAME="your_migration_name"))
	goose -dir ./migrations create "$(NAME)" sql

run:
	go run ./cmd/main.go

swag:
	swag init -g cmd/main.go -d .