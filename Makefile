# Makefile for Docker commands

# Variables
DOCKER_COMPOSE = docker compose
DEV_COMPOSE_FILE = dev.compose.yml
PROD_COMPOSE_FILE = compose.yml

# Development environment
.PHONY: dev
dev:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE_FILE) up --build

# Production environment
.PHONY: prod
prod:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE_FILE) up --build

# Clean up containers, networks, and volumes
.PHONY: clean
clean:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE_FILE) down -v:q
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE_FILE) down -v
	#docker system prune -af --volumes

# Stop all running containers
.PHONY: stop
stop:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE_FILE) stop
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE_FILE) stop

# Display container logs
.PHONY: logs
logs:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE_FILE) logs -f

# Help command to display available commands
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make dev    - Start development environment"
	@echo "  make prod   - Start production environment"
	@echo "  make clean  - Clean up all Docker resources"
	@echo "  make stop   - Stop all running containers"
	@echo "  make logs   - Display container logs"
	@echo "  make help   - Display this help message"
