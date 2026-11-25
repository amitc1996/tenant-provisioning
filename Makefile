.PHONY: up down restart logs ps build env

env:
	@sh scripts/setup-env.sh

up: env
	docker compose up -d --build --wait --wait-timeout 120

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

ps:
	docker compose ps -a

build: env
	docker compose build