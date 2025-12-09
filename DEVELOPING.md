# DerbyVote - Technical Documentation

Developer guide for building, testing, and deploying DerbyVote.

## Table of Contents

1. [Architecture](#architecture)
2. [Development Setup](#development-setup)
3. [Building](#building)
4. [Testing](#testing)
5. [API Overview](#api-overview)
6. [Deployment](#deployment)
7. [Database Schema](#database-schema)

---

## Architecture

### Technology Stack

- **Language**: Go 1.24+
- **Router**: Chi
- **Database**: SQLite with CGO
- **Templates**: Go html/template with Tailwind CSS
- **Real-time**: WebSocket for live updates
- **Assets**: Embedded using Go embed directive

### Design Patterns

**Layered Architecture**:
```
Handlers (HTTP) → Services (Business Logic) → Repository (Data Access)
```

**Repository Pattern**: Abstract database operations behind an interface, enabling mock implementations for testing.

**Service Layer**: Contains all business logic including vote validation, exclusivity enforcement, and conflict detection.

### Project Structure

```
derbyvote/
├── cmd/derbyvote/          # Main application entry point
├── internal/
│   ├── app/                # Application initialization
│   ├── auth/               # Authentication and sessions
│   ├── handlers/           # HTTP request handlers
│   ├── logger/             # Structured logging (slog)
│   ├── models/             # Data models
│   ├── repository/         # Database access layer
│   │   └── mock/           # Mock implementations
│   ├── services/           # Business logic
│   ├── testutil/           # Test utilities
│   └── websocket/          # WebSocket hub
├── pkg/
│   └── derbynet/           # DerbyNet client library
└── web/
    ├── static/js/          # Frontend JavaScript
    └── templates/          # HTML templates
```

---

## Development Setup

### Prerequisites

- Go 1.24 or later
- GCC (for SQLite CGO compilation)
- CMake 3.16+ (optional, for build automation)

### Quick Start

```bash
git clone https://github.com/abrezinsky/derbyvote.git
cd derbyvote

# Run in development mode
go run ./cmd/derbyvote/... -loglevel debug

# Or with CMake
mkdir build && cd build
cmake ..
cmake --build . --target run-dev
```

Server starts at `http://localhost:8081`. Admin credentials are displayed in the console.

### Development Tools

**Keyboard Shortcuts** (enabled by default):
- `h` - Toggle HTTP request logging
- `l` - Cycle log levels (debug → info → warn → error)
- `?` - Display help
- `Ctrl+C` - Graceful shutdown

Disable with `-nokeyboard` flag.

### Command-Line Flags

```bash
derbyvote [options]

  -port int         Server port (default: 8081)
  -db string        Database path (default: "voting.db")
  -adminpw string   Admin password (auto-generated if omitted)
  -loglevel string  Log level: debug|info|warn|error (default: "info")
  -noanimate        Skip startup animation
  -nokeyboard       Disable keyboard shortcuts
  -version          Display version
  -help             Display usage
```

---

## Building

### Standard Build

```bash
go build ./cmd/derbyvote/...
```

Output: `derbyvote` (or `derbyvote.exe` on Windows)

### Optimized Build

```bash
go build -ldflags "-s -w" ./cmd/derbyvote/...
```

Flags:
- `-s` - Omit symbol table
- `-w` - Omit DWARF debug information

### Version Injection

```bash
go build -ldflags "-X main.version=1.0.0" ./cmd/derbyvote/...
```

### CMake Build

```bash
mkdir build && cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
cmake --build .
```

Available targets:
- `derbyvote` - Build binary (default)
- `run` - Build and execute
- `run-dev` - Execute with `go run` (faster iteration)
- `test` - Run test suite
- `coverage` - Generate coverage report
- `upx` - Compress binary with UPX

### Cross-Compilation

CGO is required for SQLite. Install appropriate cross-compilers:

```bash
# ARM64 Linux
sudo apt install gcc-aarch64-linux-gnu
CC=aarch64-linux-gnu-gcc GOOS=linux GOARCH=arm64 go build

# Windows AMD64
sudo apt install gcc-mingw-w64-x86-64
CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build

# macOS (must build natively on macOS)
GOOS=darwin GOARCH=amd64 go build    # Intel
GOOS=darwin GOARCH=arm64 go build    # Apple Silicon
```

CMake targets for cross-compilation:
- `build-linux-amd64`
- `build-linux-arm64`
- `build-windows-amd64`
- `build-darwin-amd64`
- `build-darwin-arm64`
- `build-all` (requires all cross-compilers)

---

## Testing

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Specific package
go test ./internal/services/...

# With CMake
cmake --build build --target test
```

### Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# With CMake
cmake --build build --target coverage
```

### Test Coverage

- handlers: 100%
- services: 100%
- repository: 99.8% (would be 100% except for an impossible to test sqlite scenario)
- auth: 100%
- Overall: 99%+

### Integration Tests

Comprehensive end-to-end tests verify the complete voting workflow:

```bash
go test -v ./internal/services -run TestIntegration
```

Key tests:
- `TestIntegration_CompleteEndToEndWorkflow` - Full lifecycle from setup through result export
- `TestIntegration_PushResultsToDerbyNet` - DerbyNet integration
- `TestIntegration_VotingClosedAndReopened` - State management
- Plus 12 additional tests for exclusivity, concurrency, and edge cases

---

## API Overview

### Authentication

All `/admin` and `/api/admin` endpoints require authentication. Public endpoints (`/vote`, `/api/vote*`) are unauthenticated.

**Authentication Method**: Cookie-based sessions

**Login**: `POST /admin/login` with password

### Public API

**Voter Pages**:
- `GET /` - Landing page with code entry
- `GET /vote/{qrCode}` - Voter ballot interface

**Voter API**:
- `GET /api/vote-data/{qrCode}` - Fetch categories, cars, and existing votes
- `POST /api/vote` - Submit vote (payload: `{voter_qr, category_id, car_id}`)
  - Returns: `{conflict_cleared, conflict_category_id}` if exclusivity triggered
  - Setting `car_id` to 0 deselects the vote
- `GET /cars/{id}/photo` - Proxy car photo from DerbyNet

**WebSocket**:
- `GET /ws` - Real-time updates (voting status, countdown timer)

### Admin API

**Categories**:
- `GET /api/admin/categories` - List all
- `POST /api/admin/categories` - Create
- `PUT /api/admin/categories/{id}` - Update
- `DELETE /api/admin/categories/{id}` - Delete

**Cars**:
- `GET /api/admin/cars` - List all
- `POST /api/admin/cars` - Create
- `PUT /api/admin/cars/{id}` - Update
- `DELETE /api/admin/cars/{id}` - Delete

**Voters**:
- `GET /api/admin/voters` - List all
- `POST /api/admin/voters` - Create
- `POST /api/admin/generate-qr-codes` - Bulk generate (payload: `{count}`)

**Results**:
- `GET /api/admin/results` - Vote tallies with tie detection
- `GET /api/admin/stats` - Real-time statistics
- `POST /api/admin/categories/{id}/manual-winner` - Set manual override
- `DELETE /api/admin/categories/{id}/manual-winner` - Clear override

**Settings**:
- `GET /api/admin/settings` - Get all settings
- `PUT /api/admin/settings/voting-open` - Control voting state
- `POST /api/admin/settings/timer` - Start countdown (payload: `{minutes}`)
- `DELETE /api/admin/settings/timer` - Cancel countdown

**DerbyNet**:
- `POST /api/admin/sync-derbynet` - Import cars
- `POST /api/admin/sync-categories-derbynet` - Import categories
- `POST /api/admin/push-results-derbynet` - Export results

---

## Deployment

### Systemd Service

Create `/etc/systemd/system/derbyvote.service`:

```ini
[Unit]
Description=DerbyVote Server
After=network.target

[Service]
Type=simple
User=derbyvote
WorkingDirectory=/opt/derbyvote
Environment="ADMIN_PASSWORD=your-secure-password"
ExecStart=/opt/derbyvote/derbyvote -port 8081 -db /var/lib/derbyvote/voting.db -adminpw ${ADMIN_PASSWORD} -noanimate -nokeyboard
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable derbyvote
sudo systemctl start derbyvote
```

### Docker

```dockerfile
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache gcc musl-dev cmake make
WORKDIR /build
COPY . .
RUN mkdir build && cd build && \
    cmake -DCMAKE_BUILD_TYPE=Release .. && \
    cmake --build .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/bin/derbyvote /usr/local/bin/
EXPOSE 8081
VOLUME ["/data"]
CMD ["derbyvote", "-db", "/data/voting.db", "-port", "8081", "-noanimate", "-nokeyboard"]
```

```bash
docker build -t derbyvote .
docker run -d -p 8081:8081 -v derbyvote-data:/data derbyvote
```

### Reverse Proxy (Nginx)

```nginx
server {
    listen 80;
    server_name vote.example.com;

    location / {
        proxy_pass http://localhost:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Note: WebSocket upgrade headers are required for real-time functionality.

---

## Database Schema

### Core Tables

**voters**:
- `id` - Primary key
- `qr_code` - Unique identifier (indexed)
- `name`, `email` - Optional metadata
- `voter_type` - Classification (general, racer, etc.)
- `car_id` - Optional association with car entry

**cars**:
- `id` - Primary key
- `car_number` - Display identifier
- `racer_name`, `car_name` - Descriptive fields
- `photo_url` - Image reference
- `derbynet_racer_id` - DerbyNet integration field
- `eligible` - Availability flag

**categories**:
- `id` - Primary key
- `name` - Display name
- `group_id` - Optional group association
- `display_order` - Sort order
- `derbynet_award_id` - DerbyNet integration field
- `override_winner_car_id`, `override_reason` - Manual override fields

**category_groups**:
- `id` - Primary key
- `name`, `description` - Descriptive fields
- `exclusivity_pool_id` - Conflict prevention identifier
- `max_wins_per_car` - Optional limit
- `display_order` - Sort order

**votes**:
- `voter_id`, `category_id` - Composite primary key
- `car_id` - Selected car
- `voted_at` - Timestamp

**settings**:
- `key` - Setting identifier (primary key)
- `value` - Setting value

### Indexes

- `voters.qr_code` - Unique index for voter lookup
- `votes(voter_id, category_id)` - Primary key
- `votes.car_id` - Foreign key index

### Migrations

Schema initialization occurs automatically on first run. The repository layer includes migration logic in `migrations.go`.

---

## Troubleshooting

### Build Failures

**CGO errors**: Ensure GCC is installed. On Ubuntu: `sudo apt install build-essential`. On macOS: `xcode-select --install`.

**Cross-compilation failures**: Install appropriate cross-compiler for target platform.

### Runtime Issues

**Database locked**: SQLite uses single-writer concurrency model. Connection pool is set to 1. This is expected behavior.

**Port already in use**: Change port with `-port` flag or stop conflicting service.

**Can't access from other devices**: Server listens on `0.0.0.0` by default. Check firewall rules.

**WebSocket disconnects**: Verify reverse proxy configuration includes WebSocket upgrade headers.

### Performance

**Slow queries**: SQLite performance is adequate for typical event scale (50-100 concurrent voters). For larger deployments, monitor query execution times.

**Memory usage**: Application memory footprint is typically <100MB. WebSocket connections add ~1KB per active voter.

---

## Code Organization

### Handler Layer

HTTP handlers in `internal/handlers/` validate requests, invoke service methods, and format responses. Authentication middleware is applied to admin routes.

### Service Layer

Business logic in `internal/services/` includes:
- Vote validation and persistence
- Exclusivity conflict detection and resolution
- Result calculation and tie detection
- DerbyNet synchronization

### Repository Layer

Database operations in `internal/repository/` abstract SQLite access. The repository interface enables dependency injection and mock implementations for testing.

### Testing Strategy

- Unit tests for service logic
- Integration tests for end-to-end workflows
- Mock repository for handler tests
- Mock DerbyNet client for integration tests

Test files are co-located with implementation files using `_test.go` suffix.

---

## Contributing

### Code Style

Follow standard Go conventions. Run `gofmt` before committing.

### Pull Request Process

1. Fork repository
2. Create feature branch
3. Implement changes with tests
4. Ensure all tests pass: `go test ./...`
5. Verify coverage remains ≥99%
6. Submit pull request

### Testing Requirements

All new features must include:
- Unit tests for business logic
- Integration tests for workflows
- Test coverage ≥99%

---

## License

Licensed under the MIT License. See [LICENSE](LICENSE) file for details.
