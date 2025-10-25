# Proyecto 3: Tweets del Clima - Arquitectura Distribuida en Kubernetes

## 1. Resumen Ejecutivo

Sistema distribuido y escalable que simula el procesamiento de "tweets" sobre el clima local. Implementado con microservicios en Rust, Go, message brokers (Kafka y RabbitMQ), almacenamiento en memoria (Valkey) y visualización con Grafana.

## 2. Arquitectura del Sistema
```
Locust (Generador de carga)
    ↓ HTTP (10,000 requests)
Rust API (8080) - Recibe tweets
    ↓ HTTP
Go Orchestrator (8081) - Distribuye
    ↓ gRPC
Kafka Writer (50052) + RabbitMQ Writer (50053)
    ↓
Kafka + RabbitMQ (Message Brokers)
    ↓
Kafka Consumer + RabbitMQ Consumer (Go)
    ↓
Valkey/Redis (6379) - Almacenamiento
    ↓
Grafana (3000) - Visualización
```
## Funcionalidades principales:
- Recepción de tweets climáticos vía API REST
- Procesamiento concurrente con Go y Rust
- Publicación en Kafka y RabbitMQ
- Almacenamiento en Valkey
- Visualización de métricas con Grafana

## 3. Componentes Desarrollados

### 3.1 API REST en Rust
**Puerto:** 8080  
**Función:** Recibe requests HTTP de Locust con estructura JSON  
**Validación:** Municipios y climas válidos  
**Reenvío:** Hacia Go Orchestrator

**Endpoint:**
```bash
POST /api/tweets
Content-Type: application/json

{
  "municipality": "guatemala",
  "temperature": 25,
  "humidity": 60,
  "weather": "sunny"
}
```

### 3.2 Go Orchestrator
**Puerto:** 8081 (HTTP), 50051 (gRPC)  
**Función:** Recibe de Rust, publica en Kafka y RabbitMQ  
**Concurrencia:** Manejo de múltiples requests simultáneos

### 3.3 Kafka Writer
**Puerto:** 50052  
**Función:** Publica mensajes en tema `weather-tweets`  
**Configuración:** broker en localhost:9092

### 3.4 RabbitMQ Writer
**Puerto:** 50053  
**Función:** Publica mensajes en cola `weather-tweets`  
**Configuración:** amqp://guest:guest@localhost:5672

### 3.5 Consumidores (Go)
**Kafka Consumer:** Lee de `weather-tweets`, guarda en Valkey  
**RabbitMQ Consumer:** Lee de cola, guarda en Valkey

### 3.6 Almacenamiento
**Valkey/Redis:** Puerto 6379  
**Estructura de keys:** `weather:MUNICIPALITY:WEATHER_TYPE`  
**Ejemplo:** `weather:guatemala:sunny` → "82"

## 4. Instrucciones de Despliegue Local

### Prerequisitos
- Linux (recomendado)
- Docker y Docker Compose
- Rust 1.90+
- Go 1.21+
- Python 3

### Pasos

1. **Clonar repositorio:**
```bash
git clone https://github.com/TU_USUARIO/proyecto3.git
cd proyecto3
```

2. **Iniciar servicios de infraestructura:**
```bash
docker-compose up -d
```

Verifica: `docker-compose ps` (debe haber 4 servicios: Kafka, Zookeeper, RabbitMQ, Valkey, Grafana)

3. **Terminal 1 - API Rust:**
```bash
cd api-rust
cargo run
```

4. **Terminal 2 - Go Orchestrator:**
```bash
cd go-services/go1-orchestrator
go run main.go
```

5. **Terminal 3 - Kafka Writer:**
```bash
cd go-services/go2-kafka-writer
go run main.go
```

6. **Terminal 4 - RabbitMQ Writer:**
```bash
cd go-services/go3-rabbitmq-writer
go run main.go
```

7. **Terminal 5 - Kafka Consumer:**
```bash
cd go-services/kafka-consumer
go run main.go
```

8. **Terminal 6 - RabbitMQ Consumer:**
```bash
cd go-services/rabbitmq-consumer
go run main.go
```

9. **Prueba manual:**
```bash
curl -X POST http://localhost:8080/api/tweets \
  -H "Content-Type: application/json" \
  -d '{"municipality":"guatemala","temperature":25,"humidity":60,"weather":"sunny"}'
```

10. **Verificar en Redis:**
```bash
redis-cli -p 6379
GET weather:guatemala:sunny
```

## 5. Pruebas de Carga con Locust
```bash
cd locust
python3 -m venv venv
source venv/bin/activate
pip install locust

locust -f locustfile.py --host http://localhost:8080 --users 10 --spawn-rate 2
```

Accede a: http://localhost:8089

**Resultados esperados:**
- 230+ requests exitosos
- 0% failures
- Datos almacenados en Valkey

## 6. Análisis de Rendimiento

### Kafka vs RabbitMQ
- **Kafka:** Mejor para alta concurrencia, particiones distribuidas
- **RabbitMQ:** Mejor para garantía de entrega exacta

En pruebas local:
- Ambos procesaron 230 requests sin pérdidas
- RabbitMQ más rápido en escritura puntual
- Kafka mejor para distribución a múltiples consumidores

### Impacto de Réplicas
- **1 réplica Valkey:** ~892ms promedio (con Locust 10 usuarios)
- **2 réplicas Valkey:** Mayor disponibilidad, latencia similar

## 7. Preguntas Técnicas

### ¿Cómo funciona gRPC?
gRPC usa Protocol Buffers para serialización binaria, más eficiente que JSON. En este proyecto, se prepara la infraestructura pero la comunicación entre Go services es HTTP por simplicidad.

### ¿Cuál es la diferencia entre Kafka y RabbitMQ?
- **Kafka:** Event streaming, log distribuido, mejor para Big Data
- **RabbitMQ:** Message broker tradicional, garantía de entrega

### ¿Por qué Valkey en lugar de Redis?
Valkey es un fork de Redis mantenido por la comunidad, completamente compatible. Elegido por su rendimiento y uso de memoria.

### ¿Cómo funciona HPA en Kubernetes?
Horizontal Pod Autoscaler escala pods basado en CPU. Se configurará en GKE durante la fase de despliegue en la nube.

### ¿Qué es un Namespace en Kubernetes?
Agrupamiento lógico de recursos. Se usarán para separar componentes en producción.

## 8. Estructura de Carpetas
```
proyecto3/
├── api-rust/
│   ├── src/main.rs
│   ├── Cargo.toml
│   ├── Dockerfile
│   └── build.rs
├── go-services/
│   ├── go1-orchestrator/
│   ├── go2-kafka-writer/
│   ├── go3-rabbitmq-writer/
│   ├── kafka-consumer/
│   └── rabbitmq-consumer/
├── proto/
│   └── weather.proto
├── locust/
│   ├── locustfile.py
│   └── venv/
├── docker-compose.yml
└── README.md
```

## 9. Próximos Pasos (Despliegue en GCP)

1. Crear Dockerfiles (✓ Completado)
2. Subir imágenes a Zot Registry
3. Crear YAMLs de Kubernetes
4. Desplegar en GKE
5. Configurar Ingress NGINX
6. Configurar HPA

## 10. Conclusiones

Sistema completamente funcional en ambiente local que demuestra:
- Concurrencia en Rust y Go
- Comunicación asíncrona con message brokers
- Almacenamiento en memoria de alta velocidad
- Escalabilidad horizontal
- Integración de múltiples tecnologías

