# AGENTS.md - AI Coding Agent Guidelines

## Project: Payment Service (Go + Gin + GraphQL)

This project implements an Enterprise Payment Service using Clean Architecture in Go.

## Tech Stack
- **Language**: Go 1.22
- **Web Framework**: Gin
- **GraphQL**: github.com/graphql-go/graphql
- **Database**: PostgreSQL 16 (via pgx v5)
- **Telemetry**: OpenTelemetry (OTLP), Prometheus, Zap
- **Observability**: LGTM Stack (Loki, Grafana, Tempo, Mimir)
- **Load Testing**: k6
- **Orchestration**: Kubernetes (KinD) + Floci (EKS simulator)

## Architecture
Clean Architecture (Hexagonal):
```
internal/
  domain/model/       - Domain entities (zero framework annotations)
  application/
    port/outgoing/    - Repository port interfaces
    usecase/          - Application use cases
  infrastructure/
    persistence/      - PostgreSQL implementation
  presentation/
    rest/             - REST API handlers
    graphql/          - GraphQL schema + resolvers
  telemetry/          - OTel + Prometheus setup
```

## Conventions
- Use `github.com/enterprise/payment-service` as module path
- Domain entities: pure Go structs, no ORM annotations
- Use cases receive ports via constructor injection
- All use case methods accept `context.Context`
- Each use case creates its own OpenTelemetry span
- PostgreSQL: raw SQL via pgx, no ORM
- REST routes: `/v1/` prefix
- GraphQL: `/graphql` endpoint with GraphiQL playground
- Health check: `GET /health`
- Metrics: `GET /metrics` (Prometheus)

## Testing
- Domain unit tests: zero mocks
- Use case tests: mock port implementations
- Integration tests: Testcontainers or real PostgreSQL
- Contract tests: HTTP integration

## Environment Variables
- `PORT` (default: 8080)
- `SERVICE_NAME` (default: payment-service)
- `DATABASE_URL` (PostgreSQL connection string)
- `OTEL_EXPORTER_OTLP_ENDPOINT` (default: localhost:4317)

## Branching
- `main` - production
- `develop` - integration
- `###-feature-name` - feature branches

## Commit Messages
Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`
