# Pont

A web-based tunnel management service supporting both Cloudflare Quick Tunnel and ngrok, with multi-configuration management and real-time monitoring.

## Features

- **Multi-Tunnel Support**: Manage multiple tunnel configurations
- **Cloudflare Quick Tunnel**: Free, no authentication required
- **ngrok Integration**: Support for custom domains and authentication
- **Real-time Monitoring**: Live log streaming via Server-Sent Events
- **Web Interface**: Clean, responsive UI with dark/light theme
- **Database Storage**: Persistent configuration with ent ORM
- **Docker Support**: Easy deployment with Docker and docker-compose
- **RESTful API**: Complete API for programmatic access

## Quick Start

### Using Docker

```bash
docker-compose up -d
```

Access the web interface at `http://localhost:13333`

### Building from Source

```bash
# Install dependencies
go mod download

# Generate ent code
go generate ./ent

# Build
go build -o pont

# Run
./pont
```

## Configuration

Environment variables:

- `PORT`: HTTP server port (default: 13333)
- `DATA_DIR`: Data directory for database (default: ./data)
- `LOG_DIR`: Log directory (default: ./data/logs)
- `LOG_LEVEL`: Log level (default: info)

## API Endpoints

### Tunnels

- `GET /api/tunnels` - List all tunnels
- `POST /api/tunnels` - Create tunnel
- `GET /api/tunnels/:id` - Get tunnel
- `PUT /api/tunnels/:id` - Update tunnel
- `DELETE /api/tunnels/:id` - Delete tunnel
- `POST /api/tunnels/:id/start` - Start tunnel
- `POST /api/tunnels/:id/stop` - Stop tunnel
- `GET /api/tunnels/:id/status` - Get tunnel status

### System

- `GET /api/status` - Get all tunnel statuses
- `GET /api/settings` - Get settings
- `PUT /api/settings` - Update settings
- `GET /api/logs/stream` - SSE log stream
- `GET /api/logs/recent` - Recent logs
- `GET /api/version` - Version info

## Architecture

Pont is built with:

- **Backend**: Go 1.24+ with standard library
- **Database**: SQLite with ent ORM (modernc.org/sqlite - pure Go)
- **Frontend**: Vanilla JavaScript (no frameworks)
- **Logging**: Zap with log broadcasting
- **Tunnels**:
  - Cloudflare: cloudflared supervisor
  - ngrok: ngrok-go SDK

## Development

### Project Structure

```
pont/
├── main.go                 # Entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── db/                # Database initialization
│   ├── logger/            # Logging with broadcasting
│   ├── server/            # HTTP server and API
│   └── service/           # Tunnel services
├── ent/                   # Ent ORM schemas
├── web/dist/              # Frontend assets
├── version/               # Version management
├── Dockerfile
├── docker-compose.yml
└── .github/workflows/     # CI/CD

```

### Running Tests

```bash
go test ./...
```

### Building Docker Image

```bash
docker build -t pont:latest .
```

## License

MIT

## Credits

Inspired by [cfui](https://github.com/dockers-x/cfui) - Cloudflare tunnel web UI
