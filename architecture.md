# Pont Architecture Design

## Overview
Pont is a web-based tunnel management service that supports both Cloudflare Quick Tunnel and ngrok, allowing users to quickly share local services with multi-configuration management.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend (Vanilla JS)                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Tunnel List  │  │ Create/Edit  │  │ Log Viewer   │      │
│  │  Component   │  │   Component   │  │  Component   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │ REST API + SSE
┌─────────────────────────────────────────────────────────────┐
│                      Backend (Go)                            │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              HTTP Server (net/http)                   │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐     │   │
│  │  │ API Handler│  │ SSE Handler│  │Static Files│     │   │
│  │  └────────────┘  └────────────┘  └────────────┘     │   │
│  └──────────────────────────────────────────────────────┘   │
│                            │                                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │           Configuration Manager (JSON + RWMutex)      │   │
│  └──────────────────────────────────────────────────────┘   │
│                            │                                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Tunnel Service Manager                   │   │
│  │  ┌────────────────────┐  ┌────────────────────┐      │   │
│  │  │ Cloudflare Service │  │   Ngrok Service    │      │   │
│  │  │  - Quick Tunnel    │  │  - ngrok-go SDK    │      │   │
│  │  │  - Lifecycle Mgmt  │  │  - Lifecycle Mgmt  │      │   │
│  │  └────────────────────┘  └────────────────────┘      │   │
│  └──────────────────────────────────────────────────────┘   │
│                            │                                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         Logger (Zap + Broadcast + Circular Buffer)    │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────┐
│                    Persistent Storage                        │
│                  /app/data/config.json                       │
│                  /app/logs/supertunnel.log                   │
└─────────────────────────────────────────────────────────────┘
```

## Data Models

### Tunnel Configuration
```json
{
  "tunnels": [
    {
      "id": "uuid-string",
      "name": "My Dev Server",
      "type": "cloudflare|ngrok",
      "target": "http://localhost:8080",
      "enabled": true,
      "created_at": "2026-01-14T00:00:00Z",
      "updated_at": "2026-01-14T00:00:00Z",

      // Ngrok-specific fields (optional)
      "ngrok_authtoken": "optional-token",
      "ngrok_domain": "custom.example.com"
    }
  ],
  "settings": {
    "auto_start": false,
    "log_level": "info"
  }
}
```

### Tunnel Runtime State (in-memory)
```go
type TunnelState struct {
    ID          string
    Status      string // "stopped", "starting", "running", "error"
    PublicURL   string
    StartedAt   time.Time
    Error       string
    ctx         context.Context
    cancel      context.CancelFunc
}
```

## API Endpoints

### Tunnel Management
- `GET /api/tunnels` - List all tunnel configurations
- `POST /api/tunnels` - Create new tunnel configuration
- `GET /api/tunnels/:id` - Get tunnel configuration
- `PUT /api/tunnels/:id` - Update tunnel configuration
- `DELETE /api/tunnels/:id` - Delete tunnel configuration

### Tunnel Control
- `POST /api/tunnels/:id/start` - Start a tunnel
- `POST /api/tunnels/:id/stop` - Stop a tunnel
- `GET /api/tunnels/:id/status` - Get tunnel status

### System
- `GET /api/status` - Get overall system status
- `GET /api/logs/stream` - SSE log stream
- `GET /api/logs/recent` - Recent logs (last 500 lines)
- `GET /api/version` - Version info
- `GET /api/i18n/:lang` - Translations

### Static Assets
- `GET /` - Frontend HTML
- `GET /app.js` - Frontend JavaScript
- `GET /style.css` - Frontend CSS

## Component Details

### 1. Configuration Manager
**Responsibilities:**
- Load/save configuration from JSON file
- Thread-safe access with sync.RWMutex
- Validate configuration changes
- Provide default values

**Key Methods:**
```go
type Manager struct {
    mu       sync.RWMutex
    config   *Config
    filePath string
}

func (m *Manager) Get() *Config
func (m *Manager) Update(cfg *Config) error
func (m *Manager) GetTunnel(id string) (*TunnelConfig, error)
func (m *Manager) AddTunnel(tunnel *TunnelConfig) error
func (m *Manager) UpdateTunnel(id string, tunnel *TunnelConfig) error
func (m *Manager) DeleteTunnel(id string) error
```

### 2. Tunnel Service Manager
**Responsibilities:**
- Manage multiple tunnel instances
- Start/stop tunnels
- Track tunnel states
- Handle errors and auto-restart (optional)

**Key Methods:**
```go
type ServiceManager struct {
    mu      sync.RWMutex
    tunnels map[string]*TunnelState
    cfgMgr  *config.Manager
    logger  *logger.Logger
}

func (sm *ServiceManager) Start(id string) error
func (sm *ServiceManager) Stop(id string) error
func (sm *ServiceManager) GetStatus(id string) (*TunnelState, error)
func (sm *ServiceManager) GetAllStatuses() map[string]*TunnelState
```

### 3. Cloudflare Tunnel Service
**Responsibilities:**
- Create Quick Tunnel via API
- Manage tunnel lifecycle
- Extract public URL

**Implementation Approach:**
```go
type CloudflareService struct {
    config    *TunnelConfig
    ctx       context.Context
    cancel    context.CancelFunc
    publicURL string
    logger    *logger.Logger
}

func (cs *CloudflareService) Start() error {
    // 1. Call Quick Tunnel API to get credentials
    // 2. Start tunnel daemon with supervisor
    // 3. Extract and store public URL
}

func (cs *CloudflareService) Stop() error {
    // Cancel context to stop tunnel
}

func (cs *CloudflareService) GetPublicURL() string
```

### 4. Ngrok Tunnel Service
**Responsibilities:**
- Create ngrok tunnel using ngrok-go SDK
- Manage tunnel lifecycle
- Extract public URL

**Implementation Approach:**
```go
type NgrokService struct {
    config    *TunnelConfig
    ctx       context.Context
    cancel    context.CancelFunc
    listener  net.Listener
    publicURL string
    logger    *logger.Logger
}

func (ns *NgrokService) Start() error {
    // 1. Create ngrok listener with options
    // 2. Get public URL from listener
    // 3. Forward traffic to target
}

func (ns *NgrokService) Stop() error {
    // Close listener
}

func (ns *NgrokService) GetPublicURL() string
```

### 5. Logger with Broadcasting
**Responsibilities:**
- Structured logging with zap
- Circular buffer for recent logs
- Broadcast logs to SSE clients
- Log rotation with lumberjack

**Key Features:**
- 500-line circular buffer
- Multiple SSE subscribers
- Auto-cleanup inactive subscribers (5min timeout)
- JSON file logs + colored console

### 6. HTTP Server
**Responsibilities:**
- Serve REST API
- Serve SSE endpoints
- Serve static frontend assets
- Middleware (logging, panic recovery, CORS)

**Middleware Chain:**
```go
handler = loggingMiddleware(
    panicRecoveryMiddleware(
        corsMiddleware(
            router
        )
    )
)
```

## Frontend Architecture

### Components

**1. Tunnel List Component**
- Display all tunnel configurations
- Show status (running/stopped/error)
- Show public URL when running
- Quick start/stop buttons
- Edit/delete actions

**2. Create/Edit Tunnel Component**
- Form for tunnel configuration
- Type selection (Cloudflare/ngrok)
- Target URL input
- Optional ngrok settings
- Validation

**3. Log Viewer Component**
- Real-time log streaming via SSE
- Auto-scroll toggle
- Log level filtering
- Clear logs button

**4. Settings Component**
- Global settings
- Theme toggle (dark/light)
- Language selection

### State Management
- No framework, use vanilla JS
- Local state in each component
- Event-driven communication
- LocalStorage for preferences

### Styling
- Modern CSS with CSS variables
- Responsive design
- Dark/light theme support
- Clean, minimal UI

## Deployment

### Docker
```dockerfile
# Multi-stage build
FROM golang:1.24-alpine AS builder
# Build static binary with embedded assets

FROM alpine:latest
# Copy binary, create non-root user
# Expose port, set volumes
```

### Docker Compose
```yaml
version: '3.8'
services:
  supertunnel:
    build: .
    ports:
      - "13333:13333"
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
    environment:
      - PORT=13333
      - LOG_LEVEL=info
```

### GitHub Actions
- Build and test on push
- Build Docker image
- Push to Docker Hub/GHCR
- Multi-arch support (amd64, arm64)

## Security Considerations

1. **No Authentication (Initial Version)**
   - Suitable for local/trusted networks
   - Can add auth later

2. **Input Validation**
   - Validate all API inputs
   - Sanitize URLs
   - Prevent path traversal

3. **Resource Limits**
   - Limit number of tunnels
   - Limit log buffer size
   - Timeout for HTTP requests

4. **Secrets Management**
   - Store ngrok tokens securely
   - Don't log sensitive data

## Performance Considerations

1. **Memory Management**
   - Object pooling for API responses (optional)
   - Circular buffer for logs
   - Cleanup inactive SSE subscribers

2. **Concurrency**
   - Thread-safe configuration access
   - Goroutines for tunnel management
   - Context-based cancellation

3. **Logging**
   - Async logging with zap
   - Log rotation to prevent disk fill
   - Configurable log levels

## Future Enhancements

1. **Authentication & Authorization**
   - Basic auth
   - JWT tokens
   - User management

2. **Multiple Simultaneous Tunnels**
   - Run multiple tunnels at once
   - Resource management

3. **Tunnel Templates**
   - Predefined configurations
   - Quick setup

4. **Metrics & Monitoring**
   - Prometheus metrics
   - Traffic statistics
   - Uptime tracking

5. **Webhooks**
   - Notify on tunnel events
   - Integration with external services
