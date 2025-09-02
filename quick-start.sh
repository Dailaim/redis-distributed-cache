#!/bin/bash

# Script de inicio rápido para Sistema de Caché Distribuido
# Compatible con Docker o Podman
set -e

echo "🚀 Sistema de Caché Distribuido - Inicio Rápido"
echo "================================================"

# Detectar motor de contenedores
ENGINE=""
COMPOSE=""

if command -v docker &> /dev/null; then
    ENGINE="docker"
    if command -v docker-compose &> /dev/null; then
        COMPOSE="docker-compose"
    elif docker compose version &> /dev/null; then
        COMPOSE="docker compose"
    fi
elif command -v podman &> /dev/null; then
    ENGINE="podman"
    if command -v podman-compose &> /dev/null; then
        COMPOSE="podman-compose"
    else
        echo "❌ podman-compose no está instalado. Instálalo primero."
        exit 1
    fi
else
    echo "❌ Ni Docker ni Podman están instalados."
    exit 1
fi

echo "✅ Usando $ENGINE con $COMPOSE"

# Limpiar contenedores previos
echo "🧹 Limpiando contenedores previos..."
$COMPOSE down -v 2>/dev/null || true

# Construir y levantar servicios
echo "🏗️  Construyendo y levantando servicios..."
$COMPOSE up -d --build

echo "⏳ Esperando que los servicios estén listos..."
sleep 10

# Health check del caché
echo "🔍 Verificando estado del servidor de caché..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null; then
        echo "✅ Servidor de caché está listo"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ El servidor de caché no respondió a tiempo"
        echo "Logs del servidor:"
        $COMPOSE logs cache-server
        exit 1
    fi
    sleep 2
done

# Health check de Redis
echo "🔍 Verificando estado de Redis..."
for i in {1..30}; do
    if $COMPOSE exec -T redis redis-cli ping > /dev/null 2>&1; then
        echo "✅ Redis está listo"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ Redis no respondió a tiempo"
        echo "Logs de Redis:"
        $COMPOSE logs redis
        exit 1
    fi
    sleep 2
done

echo ""
echo "🎉 ¡Sistema listo!"
echo "================================================"
echo "📡 Endpoints disponibles:"
echo "  - API del Caché: http://localhost:8080/api/v1/cache"
echo "  - Health Check:  http://localhost:8080/health"
echo "  - Redis UI:      http://localhost:8081 (usuario: admin, contraseña: admin)"
echo ""
echo "🧪 Ejecutar pruebas de la API:"
echo "  ./test-api.sh"
echo ""
echo "📊 Ver logs en tiempo real:"
echo "  $COMPOSE logs -f"
echo ""
echo "🛑 Detener todos los servicios:"
echo "  $COMPOSE down"
echo ""
echo "📖 Para más información, consulta README.md"

# Ejecutar pruebas básicas
echo ""
echo "🧪 Ejecutando pruebas básicas..."
echo ""

# Test básico de SET
echo "💾 Test SET:"
curl -s -X PUT "http://localhost:8080/api/v1/cache/welcome" \
  -H "Content-Type: application/json" \
  -d '{"value": "¡Bienvenido al Sistema de Caché Distribuido!", "ttl": "1h"}' | jq .

echo ""
echo "📤 Test GET:"
curl -s -X GET "http://localhost:8080/api/v1/cache/welcome" | jq .

echo ""
echo "📊 Test STATS:"
curl -s -X GET "http://localhost:8080/api/v1/cache/stats" | jq .

echo ""
echo "✨ ¡Todo funcionando correctamente!"
echo "Puedes empezar a usar el sistema de caché distribuido."
