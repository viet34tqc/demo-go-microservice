# Docker Setup

This project has two Docker modes:

- Development mode uses Air for hot reload.
- Production mode builds small runtime images from the service Dockerfiles.

## File Layout

```text
.docker/Dockerfile.dev          Shared development image with Go and Air
docker-compose.dev.yml          Development Compose stack
docker-compose.yml              Production-style Compose stack
cmd/gateway/Dockerfile          Gateway production Dockerfile
cmd/user-service/Dockerfile     User service production Dockerfile
cmd/todo-service/Dockerfile     Todo service production Dockerfile
cmd/*/.air.toml                 Per-service Air hot reload config
.dockerignore                   Files ignored by Docker build context
Makefile                        Shortcuts for common dev commands
```

## Prerequisites

1. Install Docker Desktop.
2. Start Docker Desktop before running Compose commands.
3. Optional: install GNU Make if you want to use `make dev` shortcuts.

On Windows, Make is usually available through WSL, Git Bash, MSYS2, Chocolatey, or Scoop. If `make` is not installed, use the Docker Compose commands directly.

## Development Mode

Development mode is for local coding. It mounts the repository into each container and runs Air so services rebuild when code changes.

Start the dev stack in the foreground:

```bash
docker compose -f docker-compose.dev.yml up
```

Or with Make:

```bash
make dev
```

Start the dev stack in the background:

```bash
docker compose -f docker-compose.dev.yml up -d
```

Or with Make:

```bash
make dev-up
```

Rebuild the dev image:

```bash
docker compose -f docker-compose.dev.yml build
```

Or with Make:

```bash
make dev-build
```

Follow logs:

```bash
docker compose -f docker-compose.dev.yml logs -f
```

Or with Make:

```bash
make dev-logs
```

Stop the dev stack:

```bash
docker compose -f docker-compose.dev.yml down
```

Or with Make:

```bash
make dev-down
```

Remove dev containers and cache volumes:

```bash
docker compose -f docker-compose.dev.yml down -v
```

Or with Make:

```bash
make dev-clean
```

## How Hot Reload Works

Each service has its own `.air.toml` file:

```text
cmd/gateway/.air.toml
cmd/user-service/.air.toml
cmd/todo-service/.air.toml
```

The dev Compose file starts each service like this:

```yaml
command: air -c cmd/gateway/.air.toml
```

Air watches only the service folder plus `go.mod` and `go.sum`. For example, gateway watches:

```toml
include_dir = ["cmd/gateway"]
include_file = ["go.mod", "go.sum"]
```

This means editing `cmd/user-service` will not restart the gateway.

Polling is enabled:

```toml
poll = true
poll_interval = 500
```

This is intentional. Docker Desktop on Windows can miss native filesystem events from bind mounts, and polling makes hot reload reliable.

## Why The Whole Repo Is Mounted In Dev

The dev stack uses this volume:

```yaml
volumes:
  - .:/app
```

That means every dev container can see the whole repository. This is okay in development because:

- The project is one Go module with multiple `cmd/...` services.
- Hot reload needs access to source code.
- Air is configured to watch only the relevant service folder.
- Production images do not use this bind mount.

The dev stack also uses Go cache volumes:

```yaml
- go-mod-cache:/go/pkg/mod
- go-build-cache:/root/.cache/go-build
```

These keep dependency downloads and rebuilds faster between container restarts.

## Production Mode

Production mode uses `docker-compose.yml` and each service's production Dockerfile.

Build production images:

```bash
docker compose -f docker-compose.yml build
```

Start production-style containers:

```bash
docker compose -f docker-compose.yml up -d
```

View logs:

```bash
docker compose -f docker-compose.yml logs -f
```

Stop production-style containers:

```bash
docker compose -f docker-compose.yml down
```

## How Production Images Work

Each production Dockerfile uses a multi-stage build:

1. The builder stage uses the Go image.
2. It downloads dependencies.
3. It compiles one service binary.
4. The final stage uses a small distroless image.
5. Only the compiled binary is copied into the final image.

So production containers do not include:

- Source code
- `.git`
- Air
- The Go toolchain
- Dockerfiles

That is the main difference from dev mode.

## Common Workflow

1. Start dev mode:

```bash
make dev
```

2. Edit a service file, for example:

```text
cmd/gateway/main.go
```

3. Watch Air rebuild in the logs.

4. Test the service:

```bash
curl http://localhost:8080/health
```

5. Stop dev mode:

```bash
make dev-down
```

6. Before production-style testing, build and run:

```bash
docker compose -f docker-compose.yml up -d --build
```

## Troubleshooting

If hot reload does not trigger, restart the dev stack:

```bash
docker compose -f docker-compose.dev.yml restart
```

If dependencies or build cache seem stale, remove dev volumes:

```bash
docker compose -f docker-compose.dev.yml down -v
docker compose -f docker-compose.dev.yml up --build
```

If ports are already in use, stop the existing containers or change the host ports in the Compose file.

If `make` is not recognized on Windows, run the Docker Compose commands directly or install GNU Make.
