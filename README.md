# graphserv

A RESTful Go service for managing a Neo4j-backed topology graph. Designed for infrastructure observability use cases including root cause analysis (RCA) and blast radius impact assessment.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Graph Schema](#graph-schema)
- [Prerequisites](#prerequisites)
- [Configuration](#configuration)
- [Getting Started](#getting-started)
- [API Reference](#api-reference)
- [Swagger UI](#swagger-ui)
- [Postman Collection](#postman-collection)
- [Docker](#docker)
- [Testing](#testing)
- [Project Structure](#project-structure)
- [Makefile Targets](#makefile-targets)
- [License](#license)

## Overview

graphserv models your infrastructure as a directed graph in Neo4j. Applications, storage, and network resources are represented as nodes. Relationships like `CALLS`, `USES_STORAGE`, and `CONNECTS_TO` capture how components depend on each other. On top of this topology, graphserv supports:

- **Anomaly tracking** — attach anomalies (latency spikes, disk usage alerts) to topology nodes via `DETECTED_ON` relationships.
- **Incident and change management** — model incident tickets, change tickets, RCA tickets, and corrective actions as graph nodes linked to affected infrastructure.
- **Root cause analysis** — traverse downstream from an impacted node through topology edges to find the nearest active anomalies.
- **Blast radius / impact analysis** — reverse-traverse incoming edges to determine which components depend on a failed node.

## Architecture

```
                         +------------------+
    HTTP clients ------> |    graphserv     |
    (Postman, curl,      |  (Go HTTP mux)   |
     Swagger UI)         +--------+---------+
                                  |
                         +--------+---------+
                         |   Neo4j (Bolt)   |
                         |  Graph Database  |
                         +------------------+
```

The service is a single Go binary with no external dependencies beyond Neo4j. It uses:

- Go's built-in `net/http` with the Go 1.22+ method+path routing (`GET /path`, `POST /path`).
- The official [neo4j-go-driver](https://github.com/neo4j/neo4j-go-driver) for Bolt protocol communication.
- [swaggo/swag](https://github.com/swaggo/swag) for auto-generated Swagger/OpenAPI documentation.

## Graph Schema

### Node Labels

| Label | Purpose | Key Properties |
|-------|---------|----------------|
| `Application` | Apps/services in your topology | `id`, `name`, `tier` (frontend/backend), `owner`, `criticality` |
| `Storage` | Storage resources (DBs, disks) | `id`, `name`, `type` (SQL/NoSQL), `capacity` |
| `Network` | Network elements (switches, routers, VPCs) | `id`, `name`, `type`, `location` |
| `IncidentTicket` | Operational incidents | `id`, `severity`, `status`, `startTime`, `endTime` |
| `ChangeTicket` | Changes impacting topology | `id`, `description`, `startTime`, `endTime`, `status` |
| `RCATicket` | Root cause analysis records | `id`, `description`, `status` |
| `Action` | Corrective actions linked to RCA | `id`, `description`, `owner`, `status` |
| `Anomaly` | Detected abnormal behavior | `id`, `type`, `severity`, `status`, `startTime`, `endTime` |
| `Call` | Reified app-to-app interactions (optional) | `id`, `latency`, `errorRate` |

### Relationships

| Relationship | Direction | Properties |
|-------------|-----------|------------|
| `CALLS` | Application -> Application | `startTime`, `endTime` |
| `USES_STORAGE` | Application -> Storage | `startTime`, `endTime` |
| `CONNECTS_TO` | Application -> Network | optional |
| `STORED_ON_NETWORK` | Storage -> Network | optional |
| `IMPACTS` | IncidentTicket -> Application/Storage/Network | `startTime`, `endTime`, `role` |
| `AFFECTS` | ChangeTicket -> Application/Storage/Network | `startTime`, `endTime`, `description` |
| `ROOT_CAUSE_OF` | RCATicket -> IncidentTicket | none |
| `HAS_ACTION` | RCATicket -> Action | none |
| `DETECTED_ON` | Anomaly -> Application/Storage/Network | `startTime`, `endTime`, `severity` |
| `TO` | Call -> Application | `latency`, `errorRate` |
| `DEPENDS_ON_TRANSITIVE` | Application -> any (precomputed) | none |

## Prerequisites

- **Go 1.22+** (the project uses method+path routing introduced in Go 1.22)
- **Neo4j 5.x** (Community or Enterprise) running and accessible via Bolt
- **swag CLI** for Swagger doc generation:
  ```bash
  go install github.com/swaggo/swag/cmd/swag@latest
  ```
- **Docker** (optional, for containerized deployment)

## Configuration

graphserv is configured entirely via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `NEO4J_URI` | Yes | — | Neo4j Bolt URI (e.g. `bolt://localhost:7687`) |
| `NEO4J_PASSWORD` | Yes | — | Neo4j password |
| `NEO4J_USER` | No | `neo4j` | Neo4j username |
| `NEO4J_DATABASE` | No | `neo4j` | Neo4j database name |
| `PORT` | No | `8080` | HTTP listen port |

Create a `.env` file for local development (already in `.gitignore`):

```bash
NEO4J_URI=bolt://localhost:7687
NEO4J_USER=neo4j
NEO4J_PASSWORD=your-password
NEO4J_DATABASE=neo4j
PORT=8080
```

## Getting Started

### 1. Set up Neo4j

Start a local Neo4j instance. The easiest way is Docker:

```bash
docker run -d \
  --name neo4j \
  -p 7474:7474 -p 7687:7687 \
  -e NEO4J_AUTH=neo4j/your-password \
  neo4j:5
```

Then create constraints and indexes. Open the Neo4j Browser at `http://localhost:7474` and run the Cypher statements from `neo_4_j_setup_and_schema_execution_guide.md`, or execute them programmatically.

### 2. Build and run

```bash
# Install swag CLI (one-time)
go install github.com/swaggo/swag/cmd/swag@latest

# Build (generates Swagger docs, then compiles)
make build

# Run
export NEO4J_URI=bolt://localhost:7687
export NEO4J_PASSWORD=your-password
make run
```

The server starts on `http://localhost:8080`. You should see:

```
graphserv listening on :8080 (Neo4j bolt://localhost:7687)
Swagger UI at http://localhost:8080/swagger/index.html
```

### 3. Verify

```bash
# Health check
curl http://localhost:8080/health
# {"status":"ok"}

# API root — lists all endpoints
curl http://localhost:8080/api/v1
```

### 4. Ingest sample data

Create an Application node:

```bash
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "label": "Application",
    "properties": {
      "id": "app-123",
      "name": "PaymentsService",
      "tier": "backend",
      "owner": "team-payments",
      "criticality": "high"
    }
  }'
```

Create a relationship:

```bash
curl -X POST http://localhost:8080/api/v1/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "type": "CALLS",
    "from": {"label": "Application", "id": "app-123"},
    "to": {"label": "Application", "id": "app-124"},
    "properties": {"startTime": "2026-03-25T09:00:00"}
  }'
```

## API Reference

### Health

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check — returns `{"status": "ok"}` |
| `GET` | `/api/v1` | Lists all available API endpoints |

### Nodes

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/nodes` | Ingest (merge) a node. Body: `{"label": "...", "properties": {"id": "...", ...}}` |
| `GET` | `/api/v1/nodes/{label}` | List nodes by label. Query: `?limit=50` |
| `GET` | `/api/v1/nodes/{label}/{id}` | Get a single node by label and id |
| `PATCH` | `/api/v1/nodes/{label}/{id}` | Update (merge) properties on a node |
| `DELETE` | `/api/v1/nodes/{label}/{id}` | Detach-delete a node and all its relationships |

**Allowed labels:** `Application`, `Storage`, `Network`, `IncidentTicket`, `ChangeTicket`, `RCATicket`, `Action`, `Anomaly`, `Call`

### Relationships

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/relationships` | Ingest (merge) a relationship |
| `GET` | `/api/v1/relationships` | List relationships. Required query: `fromLabel`, `fromId`, `type`. Optional: `toLabel`, `toId`, `limit` |
| `PATCH` | `/api/v1/relationships` | Update properties on a relationship |
| `DELETE` | `/api/v1/relationships` | Delete a relationship |

**Allowed types:** `CALLS`, `USES_STORAGE`, `CONNECTS_TO`, `STORED_ON_NETWORK`, `IMPACTS`, `AFFECTS`, `ROOT_CAUSE_OF`, `HAS_ACTION`, `DETECTED_ON`, `TO`, `DEPENDS_ON_TRANSITIVE`

### Anomalies

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/anomalies` | Upsert an anomaly and create `DETECTED_ON` edges to topology nodes |

Request body:

```json
{
  "anomaly": {"id": "ANOM-1", "type": "latency_spike", "severity": "high", "status": "active", "startTime": "2026-03-25T09:55:00"},
  "detectedOn": [{"label": "Application", "id": "app-123"}],
  "relationshipProperties": {"severity": "high"}
}
```

### Analysis

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/analysis/root-cause` | Traverse topology from a start node to find active anomalies |
| `GET` | `/api/v1/analysis/impact` | Blast radius — find all nodes that depend on a given node |

**Root cause** request body:

```json
{
  "startLabel": "Application",
  "startId": "app-123",
  "maxDepth": 5,
  "anomalyStatus": "active",
  "limit": 50
}
```

**Impact** query parameters: `?label=Application&id=app-123&useTransitive=false`

### Error Responses

All errors return JSON with an `error` field:

```json
{"error": "not found"}
```

| Status | Meaning |
|--------|---------|
| `400` | Bad request (validation error) |
| `404` | Node or relationship not found |
| `502` | Neo4j backend error |
| `503` | Context canceled or deadline exceeded |

## Swagger UI

Swagger UI is served at runtime. Once the server is running, open:

```
http://localhost:8080/swagger/index.html
```

You can visualize all endpoints and test them directly from the browser. The OpenAPI spec is auto-generated from Go annotations using [swaggo/swag](https://github.com/swaggo/swag).

To regenerate the Swagger docs after modifying annotations:

```bash
make swagger
```

## Postman Collection

A ready-to-import Postman collection is included at `graphserv.postman_collection.json`.

**Import:** Open Postman -> File -> Import -> select the file.

The collection is organized into folders to run in order:

| Folder | Description |
|--------|-------------|
| Health | Health check and API root |
| 1 - Ingest Nodes | Creates all 9 node types with sample data |
| 2 - Ingest Relationships | Creates all relationship types between the sample nodes |
| 3 - Anomalies | Posts anomalies with DETECTED_ON edges |
| 4 - Analysis | Root cause traversal and blast radius queries |
| 5 - CRUD Operations | List, get, patch nodes and relationships |
| 6 - Cleanup | Sample delete operations |

The `baseUrl` collection variable defaults to `http://localhost:8080`.

## Docker

### Build the image

```bash
make docker-build
# or directly:
docker build -t rajeshkurup77/graphserv:latest .
```

The Dockerfile uses a multi-stage build:
1. **Build stage:** `golang:1.26-alpine` — compiles a static binary with `CGO_ENABLED=0`
2. **Runtime stage:** `alpine:3.21` — minimal image with just the binary and CA certificates

### Run the container

```bash
docker run -p 8080:8080 \
  -e NEO4J_URI=bolt://host.docker.internal:7687 \
  -e NEO4J_PASSWORD=your-password \
  rajeshkurup77/graphserv:latest
```

> Use `host.docker.internal` to reach a Neo4j instance running on your host machine.

### Push to Docker Hub

```bash
# Login first (one-time)
docker login

# Build and push
make docker-push
```

Image is published to: `docker.io/rajeshkurup77/graphserv:latest`

## Testing

Run all unit tests:

```bash
make test
```

### Test coverage by package

| Package | Test File | Tests | What is covered |
|---------|-----------|-------|----------------|
| `main` | `main_test.go` | 4 | healthHandler, writeJSON, loggingMiddleware |
| `internal/api` | `server_test.go` | 19 | JSON parsing, response writing, error mapping, input validation for all handlers |
| `internal/config` | `config_test.go` | 8 | Env loading with defaults, custom values, and missing required vars |
| `internal/graphstore` | `store_test.go` | 2 | Nil store/driver Close safety |
| `internal/graphstore` | `nodes_test.go` | 10 | Label validation, nil properties, missing id for all CRUD methods |
| `internal/graphstore` | `relationships_test.go` | 11 | Rel type validation, from/to label validation for all methods |
| `internal/graphstore` | `anomalies_test.go` | 6 | Nil props, missing id, empty/invalid targets |
| `internal/graphstore` | `validate_test.go` | 7 | Label/rel type allow-lists, sortedKeys, clampDepth |
| `internal/graphstore` | `coerce_test.go` | 10 | anyInt type coercion (int, int32, int64, float64, nil, string, bool) |
| `internal/graphstore` | `convert_test.go` | 11 | ToJSONValue for all Neo4j types (Node, Relationship, Path, Point2D, Point3D), nodeMap, relMap, pathMap |
| `internal/graphstore` | `analysis_test.go` | 3 | sQuote helper for Cypher IN-clause generation |

**Total: 91 unit tests**

> The store methods that execute Cypher against Neo4j are covered for their validation/error paths. Full happy-path testing requires a running Neo4j instance (integration tests).

## Project Structure

```
graphserv/
├── main.go                          # Entry point: config, Neo4j init, HTTP server, graceful shutdown
├── main_test.go                     # Tests for health handler, middleware
├── go.mod                           # Go module definition
├── go.sum                           # Dependency checksums
├── Makefile                         # Build, test, swagger, docker targets
├── Dockerfile                       # Multi-stage Docker build
├── .gitignore                       # Git ignore rules
├── README.md                        # This file
├── graphserv.postman_collection.json # Postman collection for all APIs
├── neo_4_j_topology_schema.md       # Graph schema documentation
├── neo_4_j_setup_and_schema_execution_guide.md  # Neo4j setup guide with Cypher
├── docs/                            # Auto-generated Swagger docs (swag init)
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
└── internal/
    ├── api/
    │   ├── server.go                # HTTP handlers: nodes, relationships, anomalies, analysis
    │   └── server_test.go           # Handler unit tests
    ├── config/
    │   ├── config.go                # Environment-based configuration
    │   └── config_test.go           # Config unit tests
    └── graphstore/
        ├── store.go                 # Neo4j driver lifecycle, session helpers
        ├── store_test.go            # Store unit tests
        ├── nodes.go                 # Node CRUD: ingest, get, list, update, delete
        ├── nodes_test.go            # Node unit tests
        ├── relationships.go         # Relationship CRUD: ingest, list, update, delete
        ├── relationships_test.go    # Relationship unit tests
        ├── anomalies.go             # Anomaly upsert with DETECTED_ON edges
        ├── anomalies_test.go        # Anomaly unit tests
        ├── analysis.go              # Root cause traversal and impact analysis
        ├── analysis_test.go         # Analysis unit tests
        ├── validate.go              # Label and relationship type allow-lists
        ├── validate_test.go         # Validation unit tests
        ├── coerce.go                # Numeric type coercion helpers
        ├── coerce_test.go           # Coercion unit tests
        ├── convert.go               # Neo4j types to JSON-friendly maps
        └── convert_test.go          # Conversion unit tests
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Generate Swagger docs and compile the binary |
| `make clean` | Remove the compiled binary |
| `make run` | Build and run the server |
| `make test` | Run all unit tests |
| `make swagger` | Regenerate Swagger docs from annotations |
| `make docker-build` | Build Docker image (`rajeshkurup77/graphserv:latest`) |
| `make docker-push` | Build and push Docker image to Docker Hub |

## License

This project is for learning and educational purposes.
