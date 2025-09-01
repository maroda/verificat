# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Verificat is a Go-based autonomous agent that performs continuous verification tests against production services. It implements a "Production Readiness Checklist" system that scores services based on eight principles: stability, reliability, scalability, performance, fault tolerance, catastrophe-preparedness, monitoring, and documentation.

## Architecture

- **Entry Point**: `main.go` - starts HTTP server on port 4330
- **Core Package**: `server/` - contains all business logic
- **Database**: JSON file storage (`almanac.db.json`) 
- **Web Interface**: Go HTML templates in `server/templates/`
- **Kubernetes**: Deployment manifests in `kube/`

The system uses a scoring mechanism where services start at 100 and lose points for failed verifications. Currently implements GitHub CODEOWNERS verification against Backstage service catalog.

## Development Commands

### Build and Run
```bash
# Build the application
go build -o verificat

# Run locally (requires environment variables)
./verificat

# Run with Docker
docker run -ti --rm --name verificat -p 4330:4330 ghcr.io/maroda/verificat:develop
```

### Testing
```bash
# Run all tests (requires VPN access and environment variables)
go test ./...

# Test specific package
go test ./server

# Run integration smoketest (after starting verificat)
./server/testdata/smoketest.sh http://localhost:4330
```

### Required Environment Variables
- `GH_TOKEN`: GitHub Personal Access Token with `repo, package:read` scope
- `BACKSTAGE`: Backstage API endpoint (e.g., "https://backstage.rainbowq.co")

## API Endpoints

- `GET /` - Web interface
- `GET /healthz` - Health check
- `GET /v0/almanac` - All service scores
- `POST /v0/{service-name}` - Test specific service
- `GET /almanac` - JSON dump of all scores

## Key Components

- **VerificationServ** (`server/server.go`): Main HTTP handler
- **ServiceStore** interface: Database abstraction for service storage
- **GitHub Verification** (`server/ghVerify.go`): CODEOWNERS validation
- **Backstage Integration** (`server/bsRead.go`): Service catalog integration

## Deployment

Uses Kubernetes with External Secrets Operator (ESO) for secret management. See `kube/README.md` for detailed LocalStack setup instructions.