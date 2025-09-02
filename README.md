# Sistema de Caché Distribuido

Un sistema de caché distribuido escalable y de alto rendimiento implementado en Go con Redis como backend.

## 🚀 Características

- **Alta Concurrencia**: Manejo eficiente de múltiples requests simultáneos
- **Baja Latencia**: Operaciones optimizadas con Redis como backend
- **Escalabilidad**: Arquitectura distribuida que soporta múltiples instancias
- **Tolerancia a Fallos**: Manejo robusto de errores y reconexión automática
- **API REST**: Interfaz HTTP completa para todas las operaciones
- **Observabilidad**: Logging estructurado y métricas de salud
- **Containerización**: Deployable con Docker y Docker Compose

## 🏗️ Arquitectura

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Cliente 1     │    │   Cliente 2     │    │   Cliente N     │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                       │                       │
          └───────────────────────┼───────────────────────┘
                                  │
          ┌───────────────────────▼───────────────────────┐
          │          Load Balancer (Nginx)                 │
          └─────────┬─────────────────────────┬───────────┘
                    │                         │
     ┌──────────────▼─────────────┐ ┌─────────▼──────────────┐
     │   Cache Server 1           │ │   Cache Server 2       │
     │   (Puerto 8080)            │ │   (Puerto 8082)        │
     └──────────────┬─────────────┘ └─────────┬──────────────┘
                    │                         │
                    └─────────────┬───────────┘
                                  │
              ┌───────────────────▼───────────────────┐
              │           Redis Cluster                │
              │         (Puerto 6379)                  │
              └───────────────────────────────────────┘
```

## 📋 Requisitos

- Go 1.21 o superior
- Docker y Docker Compose
- Redis 7.0 o superior (si ejecutas localmente)

## 🛠️ Instalación y Configuración

### Opción 1: Docker Compose (Recomendado)

```bash
# Clonar el repositorio
git clone <repository-url>
cd distributed-cache

# Levantar todos los servicios
make docker-up-build

# O manualmente
docker-compose up -d --build
```

### Opción 2: Ejecución Local

```bash
# Instalar dependencias
make deps

# Ejecutar Redis localmente
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Compilar y ejecutar
make build
./bin/distributed-cache

# O ejecutar directamente
make run
```

## 🧪 Pruebas

### Ejecutar todas las pruebas
```bash
make test
```

### Pruebas unitarias
```bash
make test-unit
```

### Pruebas de integración
```bash
make test-integration
```

### Benchmarks
```bash
make benchmark
```

### Cobertura de código
```bash
make coverage
```

## 📡 API REST

### Operaciones Básicas

#### Almacenar un elemento
```bash
curl -X PUT "http://localhost:8080/api/v1/cache/mi_clave" \
  -H "Content-Type: application/json" \
  -d '{"value": "mi_valor", "ttl": "1h"}'
```

#### Recuperar un elemento
```bash
curl -X GET "http://localhost:8080/api/v1/cache/mi_clave"
```

#### Verificar existencia
```bash
curl -I "http://localhost:8080/api/v1/cache/mi_clave"
```

#### Eliminar un elemento
```bash
curl -X DELETE "http://localhost:8080/api/v1/cache/mi_clave"
```

### Operaciones Batch

#### Almacenar múltiples elementos
```bash
curl -X POST "http://localhost:8080/api/v1/cache/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "items": {
      "clave1": {"value": "valor1", "ttl": "30m"},
      "clave2": {"value": "valor2", "ttl": "1h"},
      "clave3": {"value": "valor3"}
    }
  }'
```

#### Recuperar múltiples elementos
```bash
curl -X POST "http://localhost:8080/api/v1/cache/batch/get" \
  -H "Content-Type: application/json" \
  -d '{"keys": ["clave1", "clave2", "clave3"]}'
```

#### Eliminar múltiples elementos
```bash
curl -X DELETE "http://localhost:8080/api/v1/cache/batch" \
  -H "Content-Type: application/json" \
  -d '{"keys": ["clave1", "clave2"]}'
```

### Operaciones de Gestión

#### Obtener todas las claves
```bash
curl -X GET "http://localhost:8080/api/v1/cache/keys?pattern=*"
```

#### Obtener estadísticas
```bash
curl -X GET "http://localhost:8080/api/v1/cache/stats"
```

#### Limpiar todo el caché
```bash
curl -X DELETE "http://localhost:8080/api/v1/cache/"
```

#### Estado de salud
```bash
curl -X GET "http://localhost:8080/health"
```

## ⚙️ Configuración

### Variables de Entorno

```bash
# Servidor
DC_SERVER_HOST=0.0.0.0
DC_SERVER_PORT=8080
DC_SERVER_READ_TIMEOUT=30s
DC_SERVER_WRITE_TIMEOUT=30s
DC_SERVER_IDLE_TIMEOUT=120s

# Redis
DC_CACHE_ADDRESSES=localhost:6379
DC_CACHE_PASSWORD=
DC_CACHE_DATABASE=0
DC_CACHE_MAX_RETRIES=3
DC_CACHE_POOL_SIZE=20
DC_CACHE_MIN_IDLE_CONNS=10

# Logging
DC_LOGGER_LEVEL=info
DC_LOGGER_FORMAT=json
DC_LOGGER_OUTPUT_PATH=stdout
```

### Archivo de Configuración

Ver `config.yaml` para un ejemplo completo de configuración.

## 🐳 Docker

### Servicios Disponibles

- `cache-server`: Servidor principal de caché
- `redis`: Base de datos Redis
- `redis-commander`: Interface web para Redis (perfil monitoring)
- `nginx`: Load balancer (perfil loadbalancer)

### Perfiles de Docker Compose

```bash
# Solo servicios básicos
docker-compose up -d

# Con monitoreo
docker-compose --profile monitoring up -d

# Cluster con múltiples instancias
docker-compose --profile cluster up -d

# Todo incluido
docker-compose --profile monitoring --profile cluster --profile loadbalancer up -d
```

## 📊 Monitoreo

### Redis Commander
Accede a http://localhost:8081 cuando uses el perfil de monitoring:
- Usuario: admin
- Contraseña: admin

### Logs
```bash
# Logs de todos los servicios
make docker-logs

# Logs del servidor de caché
make docker-logs-cache

# Logs de Redis
make docker-logs-redis
```

### Métricas de Salud
```bash
# Estado del sistema
curl http://localhost:8080/health

# Estadísticas del caché
curl http://localhost:8080/api/v1/cache/stats
```

## 🔧 Desarrollo

### Comandos Útiles

```bash
# Desarrollo con live reload
make dev

# Formatear código
make fmt

# Ejecutar linter
make lint

# Prueba rápida de la API
make test-api

# Inicio rápido completo
make quick-start
```

### Estructura del Proyecto

```
distributed-cache/
├── cmd/server/          # Punto de entrada de la aplicación
├── internal/            # Código interno de la aplicación
│   ├── cache/          # Lógica del caché
│   ├── config/         # Gestión de configuración
│   ├── handlers/       # Handlers HTTP
│   └── middleware/     # Middleware HTTP
├── pkg/models/         # Modelos compartidos
├── tests/              # Pruebas de integración
├── docker/             # Archivos de configuración Docker
├── Dockerfile          # Imagen Docker
├── docker-compose.yml  # Orquestación de servicios
└── Makefile           # Comandos de automatización
```

## 🚀 Performance

### Benchmarks Típicos

```
BenchmarkRedisCache_Set-8     100000    10052 ns/op    1024 B/op    5 allocs/op
BenchmarkRedisCache_Get-8     200000     8543 ns/op    1152 B/op    7 allocs/op
```

### Pruebas de Carga

```bash
# Instalar hey
go install github.com/rakyll/hey@latest

# Ejecutar prueba de carga
make load-test

# O manualmente
hey -n 10000 -c 50 -m GET http://localhost:8080/health
```

## 🔒 Seguridad

- Autenticación Redis configurable
- Rate limiting (básico incluido)
- Headers de seguridad HTTP
- Validación de entrada
- Usuario no privilegiado en Docker

## 🤝 Contribución

1. Fork el proyecto
2. Crea una branch para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la branch (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

## 📝 Licencia

Este proyecto está bajo la Licencia MIT - ver el archivo [LICENSE](LICENSE) para detalles.

## 🐛 Issues y Soporte

Si encuentras algún problema o tienes sugerencias, por favor abre un issue en el repositorio.

## 📚 Documentación Técnica

### Patrones de Diseño Implementados

1. **Repository Pattern**: Abstracción de la capa de datos
2. **Factory Pattern**: Creación de instancias de caché
3. **Middleware Pattern**: Procesamiento de requests HTTP
4. **Observer Pattern**: Logging y monitoreo

### Consideraciones de Escalabilidad

1. **Conexión Pooling**: Reutilización eficiente de conexiones Redis
2. **Batch Operations**: Operaciones en lote para reducir latencia
3. **Graceful Shutdown**: Cierre ordenado de conexiones
4. **Load Balancing**: Soporte para múltiples instancias

### Tolerancia a Fallos

1. **Reconnection Logic**: Reconexión automática a Redis
2. **Circuit Breaker**: Protección contra cascading failures
3. **Health Checks**: Monitoreo continuo del estado
4. **Graceful Degradation**: Manejo elegante de errores
