# Shortener

A URL shortener REST API built with Go. Supports user accounts, custom short codes, link expiry, click analytics, Prometheus metrics, and distributed tracing via Jaeger.

## Features

- User registration and login with JWT authentication
- Shorten URLs with auto-generated or custom short codes
- Optional link expiration
- Click tracking (IP hash + user agent) per short link
- Prometheus metrics at `/metrics`
- OpenTelemetry tracing exported to Jaeger over gRPC
- Structured JSON logging with request IDs
- Graceful shutdown
- Auto-runs database migrations on startup

## Tech Stack

| Layer | Choice |
|---|---|
| Language | Go 1.25 |
| Database | PostgreSQL 16 (via `pgx/v5`) |
| Auth | JWT (`golang-jwt/jwt/v5`) + bcrypt |
| Metrics | Prometheus (`prometheus/client_golang`) |
| Tracing | OpenTelemetry → Jaeger (OTLP gRPC) |
| HTTP | stdlib `net/http` |

## Project Structure

```
shortener/
├── cmd/server/         # main entrypoint
├── internal/
│   ├── apperr/         # sentinel errors + HTTP status mapping
│   ├── auth/           # JWT generation/validation, bcrypt helpers
│   ├── config/         # env-based config loading
│   ├── server/         # HTTP handlers, middleware, route registration
│   ├── shortcode/      # random code generation and validation
│   ├── store/          # Store interface + PostgreSQL implementation
│   └── tracing/        # OpenTelemetry setup
├── migrations/         # SQL migration files (run automatically on startup)
├── Dockerfile
├── docker-compose.yml  # Postgres, Prometheus, Jaeger
└── prometheus.yml
```

## Getting Started

### Prerequisites

- Go 1.25+
- Docker and Docker Compose

### 1. Start dependencies

```bash
docker compose up -d
```

This starts:
- PostgreSQL on port `5433`
- Prometheus on port `9091`
- Jaeger UI on port `16686` (OTLP gRPC on `4317`)

### 2. Configure environment

Copy the example env or use the defaults:

```bash
cp .env .env.local
```

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `ENV` | `development` | Environment name |
| `BASE_URL` | `http://localhost:8080` | Public base URL |
| `DATABASE_URL` | *(see .env)* | PostgreSQL connection string |
| `JWT_SECRET` | *(required)* | Secret for signing JWTs |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FILE` | *(empty)* | Optional log file path |
| `JAEGER_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint |

### 3. Run the server

```bash
go run ./cmd/server
```

Migrations run automatically on startup. The server listens on `:8080` by default.

## API Reference

### Auth

#### Register
```
POST /api/register
Content-Type: application/json

{
  "username": "alice",
  "email": "alice@example.com",
  "password": "hunter2"
}
```

#### Login
```
POST /api/login
Content-Type: application/json

{
  "username": "alice",
  "password": "hunter2"
}
```
Returns `{ "token": "<jwt>" }`. Pass this as `Authorization: Bearer <token>` on protected routes.

---

### URLs

All URL management endpoints require `Authorization: Bearer <token>`.

#### Shorten a URL
```
POST /api/urls
Authorization: Bearer <token>
Content-Type: application/json

{
  "long_url": "https://example.com/very/long/path",
  "custom_code": "mylink",        // optional, 4–20 alphanumeric chars
  "expires_at": "2026-12-31T00:00:00Z"  // optional
}
```

#### List your URLs
```
GET /api/urls
Authorization: Bearer <token>
```

#### Update a URL
```
PATCH /api/urls/{shortCode}
Authorization: Bearer <token>
Content-Type: application/json

{ "long_url": "https://new-destination.com" }
```

#### Delete a URL
```
DELETE /api/urls/{shortCode}
Authorization: Bearer <token>
```

#### Get click analytics
```
GET /api/urls/{shortCode}/clicks
Authorization: Bearer <token>
```

---

### Redirect

```
GET /{shortCode}
```

Redirects to the original URL (`302`). Returns `410 Gone` if the link has expired.

---

### Observability

| Endpoint | Description |
|---|---|
| `GET /metrics` | Prometheus metrics |
| Jaeger UI `localhost:16686` | Distributed traces |

Prometheus metrics exposed:
- `http_requests_total` — counter by method, path, status
- `http_request_duration_seconds` — histogram by method, path

## Docker

Build and run the app container:

```bash
docker build -t shortener .
docker run --env-file .env -p 8080:8080 shortener
```

Or run everything together (add an `app` service to `docker-compose.yml` if desired).

## Database Schema

Three tables, created automatically via migrations:

- **users** — `id`, `username`, `email`, `password_hash`, `created_at`
- **urls** — `id`, `user_id`, `short_code`, `long_url`, `expires_at`, `created_at`
- **clicks** — `id`, `url_id`, `ip_hash`, `user_agent`, `clicked_at`

Short codes are indexed for fast lookups. Deleting a user cascades to their URLs and clicks.

## Short Code Rules

- Auto-generated codes are 6 characters (alphanumeric)
- Custom codes must be 4–20 characters, alphanumeric only (`[a-zA-Z0-9]`)
