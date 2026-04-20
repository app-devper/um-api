# User Management API

User management (UM) is defined as the effective management of users giving them access to systems.

## Features

- **CRUD API** — Full user and system management
- **Authentication** — JWT-based with session management (Redis)
- **Authorization** — Role-based access control (SUPER, ADMIN, USER)
- **SSO handoff** — One-time ticket exchange so other front-ends (e.g. `dpharm.web.app`) can hand the user off to the UM profile page without re-login
- **Brute-force protection** — IP rate limit on `/auth/login` (always on) + optional per-username account lockout after 5 consecutive failures for 15 minutes (opt-in via `LOGIN_LOCKOUT_ENABLED`, disabled by default)
- **CORS** — Configurable cross-origin resource sharing
- **Structured Error Handling** — Consistent error codes and messages across all endpoints

## Technologies

- [Gin](https://github.com/gin-gonic/gin) — HTTP web framework
- [MongoDB](https://www.mongodb.com) — Primary database
- [Redis](https://redis.io) — Session storage
- [JWT](https://github.com/golang-jwt/jwt) — Token-based authentication

## API Documentation

Full OpenAPI 3.0 specification is available at [`docs/openapi.yaml`](docs/openapi.yaml). You can view it with [Swagger Editor](https://editor.swagger.io) or any OpenAPI-compatible tool.

## API Endpoints

### Auth (`/api/um/v1/auth`)

| Method | Path               | Auth | Description                                  |
|--------|--------------------|------|----------------------------------------------|
| POST   | `/login`           | No   | Login and get token                          |
| GET    | `/keep-alive`      | Yes  | Refresh token                                |
| GET    | `/system`          | Yes  | Get current system                           |
| POST   | `/verify-password` | Yes  | Verify user password                         |
| POST   | `/sso-ticket`      | Yes  | Issue one-time SSO handoff ticket (TTL 60s)  |
| POST   | `/exchange`        | No   | Exchange SSO ticket for a new access token   |
| POST   | `/logout`          | Yes  | Logout and end session                       |

### User (`/api/um/v1/user`)

| Method | Path                  | Auth | Role          | Description          |
|--------|-----------------------|------|---------------|----------------------|
| GET    | `/info`               | Yes  | Any           | Get own user info    |
| PUT    | `/info`               | Yes  | Any           | Update own user info |
| PUT    | `/change-password`    | Yes  | Any           | Change own password  |
| GET    | `/`                   | Yes  | SUPER, ADMIN  | List users           |
| POST   | `/`                   | Yes  | SUPER, ADMIN  | Create user          |
| GET    | `/:id`                | Yes  | SUPER, ADMIN  | Get user by ID       |
| DELETE | `/:id`                | Yes  | SUPER, ADMIN  | Delete user          |
| PUT    | `/:id`                | Yes  | SUPER, ADMIN  | Update user          |
| PATCH  | `/:id/status`         | Yes  | SUPER, ADMIN  | Update user status   |
| PATCH  | `/:id/role`           | Yes  | SUPER, ADMIN  | Update user role     |
| PATCH  | `/:id/set-password`   | Yes  | SUPER, ADMIN  | Set user password    |
| POST   | `/:id/unlock`         | Yes  | SUPER, ADMIN  | Unlock locked user   |

### System (`/api/um/v1/system`)

| Method | Path   | Auth | Role  | Description       |
|--------|--------|------|-------|-------------------|
| GET    | `/`    | Yes  | SUPER | List systems      |
| POST   | `/`    | Yes  | SUPER | Create system     |
| GET    | `/:id` | Yes  | SUPER | Get system by ID  |
| DELETE | `/:id` | Yes  | SUPER | Delete system     |
| PUT    | `/:id` | Yes  | SUPER | Update system     |

## Error Codes

All error responses follow the format:

```json
{
  "code": "UM-XXX-YYY",
  "message": "description"
}
```

| Code         | HTTP Status | Description              |
|--------------|-------------|--------------------------|
| UM-401-001   | 401         | Missing auth header      |
| UM-401-002   | 401         | Token invalid            |
| UM-401-003   | 401         | Session invalid          |
| UM-401-004   | 401         | Wrong credentials        |
| UM-400-001   | 400         | Bad request              |
| UM-400-002   | 400         | Wrong password           |
| UM-400-003   | 400         | Invalid client ID        |
| UM-400-004   | 400         | Invalid role             |
| UM-400-005   | 400         | Cannot delete self       |
| UM-403-001   | 403         | Forbidden                |
| UM-403-002   | 403         | No permission            |
| UM-403-003   | 403         | Invalid role permission  |
| UM-409-001   | 409         | Username taken           |
| UM-404-001   | 404         | Not found                |
| UM-500-001   | 500         | Internal server error    |
| UM-500-002   | 500         | Token generation failed  |

## Prerequisites

- [Go](https://go.dev) 1.26 or newer
- [MongoDB](https://www.mongodb.com) 5.0+ (local or remote)
- [Redis](https://redis.io) 6.0+ (local or remote)
- Optional: [nodemon](https://www.npmjs.com/package/nodemon) for live-reload during development

## Setup

1. Create a `.env` file in the project root with the following variables:

```env
PORT=8585
MONGO_HOST=localhost:27017
MONGO_UM_DB_NAME=your_db_name
REDIS_HOST=localhost:6379
SECRET_KEY=your_secret_key
```

| Variable                 | Description                                                                                     | Required | Default |
|--------------------------|-------------------------------------------------------------------------------------------------|----------|---------|
| `PORT`                   | HTTP port to listen on                                                                          | Yes      | —       |
| `MONGO_HOST`             | MongoDB host:port (e.g. `localhost:27017`)                                                      | Yes      | —       |
| `MONGO_UM_DB_NAME`       | MongoDB database name                                                                           | Yes      | —       |
| `REDIS_HOST`             | Redis host:port                                                                                 | Yes      | —       |
| `SECRET_KEY`             | Secret key used to sign JWT tokens                                                              | Yes      | —       |
| `LOGIN_LOCKOUT_ENABLED`  | Turn on per-username account lockout (`1`/`true`/`yes`/`on` to enable, anything else disables). | No       | off     |

> **Important:** `SECRET_KEY` must be set to a non-empty value. The application will refuse to sign or verify tokens without it.

## Run

```bash
# Download dependencies
go mod download

# Run the application
go run main.go

# Run with nodemon (auto-reload)
nodemon --exec go run main.go --signal SIGTERM
```

The server listens on `http://localhost:<PORT>` (default `8585`). All endpoints are mounted under `/api/um/v1`.

## Quick Test

```bash
# Login
curl -X POST http://localhost:8585/api/um/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password","system":"UM"}'

# Use the returned token
curl http://localhost:8585/api/um/v1/user/info \
  -H "Authorization: Bearer <accessToken>"
```

## Project Structure

```
um-api/
├── app/
│   ├── core/
│   │   ├── config/      # Application configuration
│   │   ├── constant/    # Role and status constants
│   │   ├── errs/        # Error codes and response helpers
│   │   └── utils/       # Utility functions (password hashing, context)
│   ├── domain/
│   │   ├── model/       # Data models (User, System)
│   │   ├── repository/  # Database access layer (MongoDB, Redis)
│   │   └── usecase/     # Business logic handlers
│   ├── featues/
│   │   ├── api/         # Route definitions
│   │   └── request/     # Request DTOs
│   └── init.go          # Application bootstrap
├── db/                  # Database connection setup
├── middlewares/          # Auth, CORS, recovery, routing middlewares
├── main.go              # Entry point
├── go.mod
└── go.sum
```

## Related

- Web console: [`um-web`](../um-web) — Next.js 16 + shadcn/ui frontend that consumes this API
