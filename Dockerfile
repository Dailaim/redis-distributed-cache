# Multi-stage build para optimizar el tamaño de la imagen final
FROM golang:1.21-alpine AS builder

# Instalar dependencias necesarias
RUN apk add --no-cache git ca-certificates

# Establecer directorio de trabajo
WORKDIR /app

# Copiar go mod y sum files
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY . .

# Construir la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Imagen final
FROM alpine:latest

# Instalar ca-certificates para HTTPS
RUN apk --no-cache add ca-certificates

# Crear usuario no privilegiado
RUN adduser -D -s /bin/sh appuser

# Establecer directorio de trabajo
WORKDIR /root/

# Copiar el binario desde la imagen builder
COPY --from=builder /app/main .

# Copiar archivos de configuración si existen
COPY --from=builder /app/config.yaml* ./

# Cambiar propietario de archivos
RUN chown -R appuser:appuser /root/

# Cambiar a usuario no privilegiado
USER appuser

# Exponer puerto
EXPOSE 8080

# Comando por defecto
CMD ["./main"]
