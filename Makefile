.PHONY: build run test clean docker-build docker-up docker-down docker-logs help

# Variables
APP_NAME = distributed-cache
DOCKER_COMPOSE_FILE = docker-compose.yml
GO_FILES = $(shell find . -name '*.go' -not -path './vendor/*')

# Comandos por defecto
help: ## Mostrar este mensaje de ayuda
	@echo "Comandos disponibles:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Comandos de desarrollo
build: ## Compilar la aplicación
	@echo "Compilando aplicación..."
	go build -o bin/$(APP_NAME) ./cmd/server

run: ## Ejecutar la aplicación localmente
	@echo "Ejecutando aplicación..."
	go run ./cmd/server

test: ## Ejecutar todas las pruebas
	@echo "Ejecutando pruebas..."
	go test -v ./...

test-unit: ## Ejecutar solo pruebas unitarias
	@echo "Ejecutando pruebas unitarias..."
	go test -v ./internal/cache

test-integration: ## Ejecutar pruebas de integración
	@echo "Ejecutando pruebas de integración..."
	go test -v ./tests

benchmark: ## Ejecutar benchmarks
	@echo "Ejecutando benchmarks..."
	go test -bench=. -benchmem ./internal/cache

coverage: ## Generar reporte de cobertura
	@echo "Generando reporte de cobertura..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Reporte disponible en coverage.html"

# Comandos de limpieza
clean: ## Limpiar archivos temporales y binarios
	@echo "Limpiando..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache

fmt: ## Formatear código
	@echo "Formateando código..."
	go fmt ./...

lint: ## Ejecutar linter
	@echo "Ejecutando linter..."
	golangci-lint run

# Comandos de Docker
docker-build: ## Construir imagen Docker
	@echo "Construyendo imagen Docker..."
	docker build -t $(APP_NAME):latest .

docker-up: ## Levantar servicios con Docker Compose
	@echo "Levantando servicios..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-up-build: ## Construir y levantar servicios
	@echo "Construyendo y levantando servicios..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d --build

docker-down: ## Bajar servicios
	@echo "Bajando servicios..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

docker-down-clean: ## Bajar servicios y limpiar volúmenes
	@echo "Bajando servicios y limpiando volúmenes..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down -v

docker-logs: ## Mostrar logs de los servicios
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

docker-logs-cache: ## Mostrar logs del servicio de caché
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f cache-server

docker-logs-redis: ## Mostrar logs de Redis
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f redis

# Comandos para testing con perfiles
docker-up-monitoring: ## Levantar con monitoreo
	docker-compose -f $(DOCKER_COMPOSE_FILE) --profile monitoring up -d

docker-up-cluster: ## Levantar cluster con múltiples instancias
	docker-compose -f $(DOCKER_COMPOSE_FILE) --profile cluster up -d

docker-up-full: ## Levantar todos los servicios
	docker-compose -f $(DOCKER_COMPOSE_FILE) --profile monitoring --profile cluster --profile loadbalancer up -d

# Comandos de gestión de dependencias
deps: ## Descargar dependencias
	@echo "Descargando dependencias..."
	go mod download

deps-update: ## Actualizar dependencias
	@echo "Actualizando dependencias..."
	go get -u ./...
	go mod tidy

deps-vendor: ## Crear vendor directory
	@echo "Creando vendor..."
	go mod vendor

# Comandos de testing de la API
test-api: ## Ejecutar pruebas de la API REST
	@echo "Probando API REST..."
	@echo "Asegúrate de que el servidor esté corriendo en localhost:8080"
	@echo "\n--- Test SET ---"
	curl -X PUT "http://localhost:8080/api/v1/cache/test_key" \
		-H "Content-Type: application/json" \
		-d '{"value": "test_value", "ttl": "1h"}'
	@echo "\n--- Test GET ---"
	curl -X GET "http://localhost:8080/api/v1/cache/test_key"
	@echo "\n--- Test Health ---"
	curl -X GET "http://localhost:8080/health"

# Comando para generar documentación
docs: ## Generar documentación
	@echo "Generando documentación..."
	godoc -http=:6060 &
	@echo "Documentación disponible en http://localhost:6060"

# Comando para análisis de seguridad
security: ## Ejecutar análisis de seguridad
	@echo "Ejecutando análisis de seguridad..."
	gosec ./...

# Performance testing
load-test: ## Ejecutar prueba de carga básica
	@echo "Ejecutando prueba de carga..."
	@echo "Requiere 'hey' instalado: go install github.com/rakyll/hey@latest"
	hey -n 1000 -c 10 -m GET http://localhost:8080/health

# Desarrollo
dev: ## Modo desarrollo con live reload
	@echo "Iniciando modo desarrollo..."
	@echo "Requiere 'air' instalado: go install github.com/cosmtrek/air@latest"
	air

# Quick start
quick-start: docker-up-build test-api ## Inicio rápido completo

# CI/CD helpers
ci-test: deps test lint ## Ejecutar pruebas para CI/CD

# Info
info: ## Mostrar información del proyecto
	@echo "Información del proyecto:"
	@echo "  Nombre: $(APP_NAME)"
	@echo "  Go version: $(shell go version)"
	@echo "  Docker version: $(shell docker --version)"
	@echo "  Docker Compose version: $(shell docker-compose --version)"
