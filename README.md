# DerbyVote

A web-based voting system for Pinewood Derby awards. Enables mobile voting via QR codes with real-time results and DerbyNet integration.

## Overview

DerbyVote provides a streamlined solution for collecting and tallying votes for Pinewood Derby recognition awards ("Best Design", "Most Creative", etc.). The system handles voter registration, ballot presentation, vote collection, conflict resolution, and result reporting.

## Key Features

- Mobile-optimized voting interface with instant vote persistence
- QR code-based voter authentication (registered or open mode)
- Category grouping with exclusivity enforcement
- Real-time vote monitoring and statistics
- Automatic tie detection with manual resolution
- Bidirectional DerbyNet integration (import cars, export results)
- WebSocket-based live updates
- Single-binary deployment with embedded assets

## Quick Start

```bash
# Download and run
./derbyvote

# Server starts at http://localhost:8081
# Admin credentials printed to console
```

Or build from source:

```bash
go build ./cmd/derbyvote/...
./derbyvote
```

## Documentation

**[MANUAL.md](MANUAL.md)** - Administrator guide for event setup and operation

**[DEVELOPING.md](DEVELOPING.md)** - Technical documentation for developers and system administrators

## Architecture

- **Backend**: Go 1.24+
- **Database**: SQLite
- **Frontend**: Server-rendered templates with Tailwind CSS
- **Real-time**: WebSocket for live updates
- **Deployment**: Single binary, no external dependencies

## System Requirements

**Runtime**:
- Linux, macOS, or Windows
- 20MB disk space
- 100MB RAM
- Network connectivity for voters

**Build**:
- Go 1.24+
- GCC (for SQLite CGO)
- CMake 3.16+ (optional)

## Workflow

```
1. Import cars from DerbyNet or add manually
2. Configure voting categories and groups
3. Generate voter QR codes or enable open mode
4. Open voting and distribute access
5. Monitor results in real-time
6. Close voting and resolve conflicts
7. Export results to DerbyNet
```

## DerbyNet Integration

DerbyVote integrates with [DerbyNet](https://derbynet.org/) race management software:

- Import racer roster and car information
- Sync award categories
- Export voting results and winners

DerbyNet is optional; the system functions standalone.

## Development

```bash
# Clone repository
git clone https://github.com/abrezinsky/derbyvote.git
cd derbyvote

# Development mode
go run ./cmd/derbyvote/...

# Build
go build ./cmd/derbyvote/...

# Test
go test ./...
```

See [DEVELOPING.md](DEVELOPING.md) for detailed build instructions, API documentation, and deployment guides.

## License

Licensed under the MIT License. See [LICENSE](LICENSE) file for details.

## Support

- Documentation: [MANUAL.md](MANUAL.md) | [DEVELOPING.md](DEVELOPING.md)
- Issues: https://github.com/abrezinsky/derbyvote/issues
