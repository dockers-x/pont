package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"pont/internal/config"
	"pont/internal/logger"
	"pont/internal/mcp"
	"pont/internal/service"
	"pont/internal/web"
	"pont/version"
	"time"

	"github.com/google/uuid"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server represents the HTTP server
type Server struct {
	addr       string
	cfgMgr     *config.Manager
	svcMgr     *service.Manager
	mcpServer  *mcp.Server
	httpServer *http.Server
}

// NewServer creates a new HTTP server
func NewServer(addr string, cfgMgr *config.Manager, svcMgr *service.Manager) *Server {
	// Create MCP server
	mcpServer := mcp.NewServer(cfgMgr, svcMgr)

	return &Server{
		addr:      addr,
		cfgMgr:    cfgMgr,
		svcMgr:    svcMgr,
		mcpServer: mcpServer,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/tunnels", s.handleTunnels)
	mux.HandleFunc("/api/tunnels/", s.handleTunnelByID)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/logs/stream", s.handleLogsStream)
	mux.HandleFunc("/api/logs/recent", s.handleLogsRecent)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/api/mcp/info", s.handleMCPInfo)

	// MCP endpoint (SSE)
	mcpHandler := mcpsdk.NewSSEHandler(func(r *http.Request) *mcpsdk.Server {
		return s.mcpServer.GetServer()
	}, nil)
	mux.Handle("/mcp", mcpHandler)

	// Static files
	distFS, _ := fs.Sub(web.DistFS, "dist")
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	// Wrap with middleware
	handler := s.loggingMiddleware(s.corsMiddleware(mux))

	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: handler,
	}

	logger.Sugar.Infof("Starting HTTP server on %s", s.addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Middleware
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Sugar.Infof("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// API Handlers
func (s *Server) handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getTunnels(w, r)
	case http.MethodPost:
		s.createTunnel(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTunnelByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/tunnels/"):]
	if id == "" {
		http.Error(w, "Tunnel ID required", http.StatusBadRequest)
		return
	}

	// Check for action endpoints
	if len(id) > 6 && id[len(id)-6:] == "/start" {
		s.startTunnel(w, r, id[:len(id)-6])
		return
	}
	if len(id) > 5 && id[len(id)-5:] == "/stop" {
		s.stopTunnel(w, r, id[:len(id)-5])
		return
	}
	if len(id) > 7 && id[len(id)-7:] == "/status" {
		s.getTunnelStatus(w, r, id[:len(id)-7])
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getTunnel(w, r, id)
	case http.MethodPut:
		s.updateTunnel(w, r, id)
	case http.MethodDelete:
		s.deleteTunnel(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getTunnels(w http.ResponseWriter, r *http.Request) {
	tunnels, err := s.cfgMgr.GetAllTunnels()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, tunnels)
}

func (s *Server) getTunnel(w http.ResponseWriter, r *http.Request, id string) {
	tunnel, err := s.cfgMgr.GetTunnel(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.jsonResponse(w, tunnel)
}

func (s *Server) createTunnel(w http.ResponseWriter, r *http.Request) {
	var tunnel config.TunnelConfig
	if err := json.NewDecoder(r.Body).Decode(&tunnel); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.cfgMgr.AddTunnel(&tunnel); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, tunnel)
}

func (s *Server) updateTunnel(w http.ResponseWriter, r *http.Request, id string) {
	var tunnel config.TunnelConfig
	if err := json.NewDecoder(r.Body).Decode(&tunnel); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.cfgMgr.UpdateTunnel(id, &tunnel); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, tunnel)
}

func (s *Server) deleteTunnel(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.cfgMgr.DeleteTunnel(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) startTunnel(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.svcMgr.Start(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, map[string]string{"status": "started"})
}

func (s *Server) stopTunnel(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.svcMgr.Stop(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.jsonResponse(w, map[string]string{"status": "stopped"})
}

func (s *Server) getTunnelStatus(w http.ResponseWriter, r *http.Request, id string) {
	status, err := s.svcMgr.GetStatus(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	s.jsonResponse(w, status)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	statuses := s.svcMgr.GetAllStatuses()
	s.jsonResponse(w, statuses)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := s.cfgMgr.GetSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.jsonResponse(w, settings)

	case http.MethodPut:
		var settings config.Settings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.cfgMgr.UpdateSettings(&settings); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.jsonResponse(w, settings)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to logs
	subID := uuid.New().String()
	sub := logger.Subscribe(subID)
	defer logger.Unsubscribe(subID)

	// Send logs
	for {
		select {
		case entry, ok := <-sub.Channel:
			if !ok {
				return
			}

			data, _ := json.Marshal(entry)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleLogsRecent(w http.ResponseWriter, r *http.Request) {
	logs := logger.GetRecentLogs()
	s.jsonResponse(w, logs)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, map[string]string{
		"version":    version.GetVersion(),
		"build_time": version.GetBuildTime(),
		"git_commit": version.GetGitCommit(),
	})
}

func (s *Server) handleMCPInfo(w http.ResponseWriter, r *http.Request) {
	// Get server address
	addr := s.addr
	if addr == "0.0.0.0:13333" || addr == ":13333" {
		addr = "localhost:13333"
	}

	mcpInfo := map[string]interface{}{
		"endpoint": fmt.Sprintf("http://%s/mcp", addr),
		"status":   "active",
		"tools": []map[string]string{
			{
				"name":        "listTunnels",
				"description": "List all available tunnel configurations with their current status",
			},
			{
				"name":        "startTunnel",
				"description": "Start a specific tunnel by ID and get the public URL",
				"parameters":  "tunnel_id (required): The ID of the tunnel to start",
			},
		},
		"config_example": map[string]interface{}{
			"mcpServers": map[string]interface{}{
				"pont": map[string]interface{}{
					"url": fmt.Sprintf("http://%s/mcp", addr),
				},
			},
		},
	}

	s.jsonResponse(w, mcpInfo)
}

func (s *Server) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
