package mcp

import (
	"context"
	"fmt"
	"pont/internal/config"
	"pont/internal/logger"
	"pont/internal/service"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server represents the MCP server for tunnel management
type Server struct {
	cfgMgr *config.Manager
	svcMgr *service.Manager
	server *mcp.Server
}

// TunnelInfo represents tunnel information for MCP responses
type TunnelInfo struct {
	Index     int    `json:"index"`
	Name      string `json:"name"`
	ID        string `json:"id"`
	Type      string `json:"type"`
	Target    string `json:"target"`
	Status    string `json:"status"`
	PublicURL string `json:"public_url,omitempty"`
}

// TunnelListResponse represents the response for listing tunnels
type TunnelListResponse struct {
	Count   int          `json:"count"`
	Tunnels []TunnelInfo `json:"tunnels"`
}

// TunnelStartResponse represents the response for starting a tunnel
type TunnelStartResponse struct {
	Success   bool   `json:"success"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Target    string `json:"target"`
	Status    string `json:"status"`
	PublicURL string `json:"public_url,omitempty"`
	Message   string `json:"message"`
}

// NewServer creates a new MCP server instance
func NewServer(cfgMgr *config.Manager, svcMgr *service.Manager) *Server {
	impl := &mcp.Implementation{
		Name:    "pont-tunnel-manager",
		Version: "1.0.0",
	}

	mcpServer := mcp.NewServer(impl, nil)

	s := &Server{
		cfgMgr: cfgMgr,
		svcMgr: svcMgr,
		server: mcpServer,
	}

	// Register tools
	s.registerTools()

	return s
}

// registerTools registers all MCP tools
func (s *Server) registerTools() {
	// Tool 1: List available tunnels
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "listTunnels",
		Description: "List all available tunnel configurations with their details",
	}, s.listTunnels)

	// Tool 2: Start a tunnel and get public URL
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "startTunnel",
		Description: "Start a specific tunnel by ID and return the public URL for external access",
	}, s.startTunnel)
}

// GetServer returns the underlying MCP server
func (s *Server) GetServer() *mcp.Server {
	return s.server
}

// ListTunnelsParams defines parameters for listing tunnels
type ListTunnelsParams struct{}

// listTunnels implements the tool to list all available tunnels
func (s *Server) listTunnels(
	ctx context.Context,
	req *mcp.CallToolRequest,
	params *ListTunnelsParams,
) (*mcp.CallToolResult, any, error) {
	tunnels, err := s.cfgMgr.GetAllTunnels()
	if err != nil {
		logger.Sugar.Errorf("MCP: Failed to list tunnels: %v", err)
		return nil, nil, fmt.Errorf("failed to list tunnels: %w", err)
	}

	// Build structured response
	response := TunnelListResponse{
		Count:   0,
		Tunnels: make([]TunnelInfo, 0),
	}

	for i, t := range tunnels {
		// Skip tunnels that are not MCP-enabled
		if !t.MCPEnabled {
			continue
		}

		status, _ := s.svcMgr.GetStatus(t.ID)
		tunnelInfo := TunnelInfo{
			Index:     i + 1,
			Name:      t.Name,
			ID:        t.ID,
			Type:      string(t.Type),
			Target:    t.Target,
			Status:    status.Status,
			PublicURL: status.PublicURL,
		}
		response.Tunnels = append(response.Tunnels, tunnelInfo)
	}

	response.Count = len(response.Tunnels)

	// Format as readable text
	var textResponse string
	if response.Count == 0 {
		textResponse = "No tunnels configured."
	} else {
		textResponse = fmt.Sprintf("Found %d tunnel(s):\n\n", response.Count)
		for _, t := range response.Tunnels {
			textResponse += fmt.Sprintf("%d. %s (ID: %s)\n", t.Index, t.Name, t.ID)
			textResponse += fmt.Sprintf("   Type: %s\n", t.Type)
			textResponse += fmt.Sprintf("   Target: %s\n", t.Target)
			textResponse += fmt.Sprintf("   Status: %s\n", t.Status)
			if t.PublicURL != "" {
				textResponse += fmt.Sprintf("   Public URL: %s\n", t.PublicURL)
			}
			textResponse += "\n"
		}
	}

	logger.Sugar.Infof("MCP: Listed %d tunnels", response.Count)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: textResponse},
		},
	}, response, nil
}

// StartTunnelParams defines parameters for starting a tunnel
type StartTunnelParams struct {
	TunnelID string `json:"tunnel_id" jsonschema:"required,The ID of the tunnel to start"`
}

// startTunnel implements the tool to start a tunnel and return its public URL
func (s *Server) startTunnel(
	ctx context.Context,
	req *mcp.CallToolRequest,
	params *StartTunnelParams,
) (*mcp.CallToolResult, any, error) {
	if params.TunnelID == "" {
		return nil, nil, fmt.Errorf("tunnel_id is required")
	}

	// Get tunnel configuration
	tunnelCfg, err := s.cfgMgr.GetTunnel(params.TunnelID)
	if err != nil {
		logger.Sugar.Errorf("MCP: Failed to get tunnel %s: %v", params.TunnelID, err)
		return nil, nil, fmt.Errorf("tunnel not found: %w", err)
	}

	// Check if MCP is enabled for this tunnel
	if !tunnelCfg.MCPEnabled {
		logger.Sugar.Warnf("MCP: Tunnel %s (%s) is not MCP-enabled", tunnelCfg.Name, params.TunnelID)
		return nil, TunnelStartResponse{
			Success: false,
			Name:    tunnelCfg.Name,
			Type:    string(tunnelCfg.Type),
			Target:  tunnelCfg.Target,
			Message: "This tunnel is not enabled for MCP management",
		}, fmt.Errorf("tunnel is not MCP-enabled")
	}

	// Start the tunnel
	if err := s.svcMgr.Start(params.TunnelID); err != nil {
		logger.Sugar.Errorf("MCP: Failed to start tunnel %s: %v", params.TunnelID, err)
		return nil, TunnelStartResponse{
			Success: false,
			Name:    tunnelCfg.Name,
			Type:    string(tunnelCfg.Type),
			Target:  tunnelCfg.Target,
			Message: fmt.Sprintf("Failed to start tunnel: %v", err),
		}, fmt.Errorf("failed to start tunnel: %w", err)
	}

	logger.Sugar.Infof("MCP: Started tunnel %s (%s)", tunnelCfg.Name, params.TunnelID)

	// Get the status with public URL
	status, err := s.svcMgr.GetStatus(params.TunnelID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tunnel status: %w", err)
	}

	// Build structured response
	response := TunnelStartResponse{
		Success:   true,
		Name:      tunnelCfg.Name,
		Type:      string(tunnelCfg.Type),
		Target:    tunnelCfg.Target,
		Status:    status.Status,
		PublicURL: status.PublicURL,
	}

	// Format as readable text
	textResponse := fmt.Sprintf("Tunnel '%s' started successfully!\n\n", response.Name)
	textResponse += fmt.Sprintf("Type: %s\n", response.Type)
	textResponse += fmt.Sprintf("Target: %s\n", response.Target)
	textResponse += fmt.Sprintf("Status: %s\n", response.Status)

	if response.PublicURL != "" {
		textResponse += fmt.Sprintf("\nPublic URL: %s\n", response.PublicURL)
		textResponse += "\nYou can now access your local service through this public URL."
		response.Message = "Tunnel started and public URL is available"
	} else {
		textResponse += "\nNote: Public URL will be available once the tunnel is fully established."
		response.Message = "Tunnel started, waiting for public URL"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: textResponse},
		},
	}, response, nil
}
