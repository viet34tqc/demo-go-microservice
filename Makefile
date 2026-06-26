COMPOSE_DEV := docker compose -f docker-compose.dev.yml

.PHONY: help dev dev-build dev-up dev-down dev-restart dev-logs dev-ps dev-clean

help:
	@echo "Available commands:"
	@echo "  make dev          Start dev stack in foreground with hot reload"
	@echo "  make dev-up       Start dev stack in background"
	@echo "  make dev-build    Build dev images"
	@echo "  make dev-down     Stop dev stack"
	@echo "  make dev-restart  Restart dev stack"
	@echo "  make dev-logs     Follow dev logs"
	@echo "  make dev-ps       Show dev containers"
	@echo "  make dev-clean    Stop dev stack and remove volumes"

dev:
	$(COMPOSE_DEV) up

dev-up:
	$(COMPOSE_DEV) up -d

down:
	$(COMPOSE_DEV) down

dev-build:
	$(COMPOSE_DEV) build

dev-down:
	$(COMPOSE_DEV) down

dev-restart:
	$(COMPOSE_DEV) restart

dev-logs:
	$(COMPOSE_DEV) logs -f

dev-ps:
	$(COMPOSE_DEV) ps

dev-clean:
	$(COMPOSE_DEV) down -v
