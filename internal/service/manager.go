package service

import (
	"context"
	"fmt"
	"pont/internal/config"
	"pont/internal/logger"
	"sync"
	"time"
)

// TunnelService interface for different tunnel implementations
type TunnelService interface {
	Start(ctx context.Context) error
	Stop() error
	GetPublicURL() string
	GetStatus() string
	GetError() string
}

// TunnelState represents the runtime state of a tunnel
type TunnelState struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"` // "stopped", "starting", "running", "error"
	PublicURL string    `json:"public_url"`
	StartedAt time.Time `json:"started_at"`
	Error     string    `json:"error,omitempty"`
	ctx       context.Context `json:"-"`
	cancel    context.CancelFunc `json:"-"`
	service   TunnelService `json:"-"`
}

// Manager manages multiple tunnel instances
type Manager struct {
	mu      sync.RWMutex
	tunnels map[string]*TunnelState
	cfgMgr  *config.Manager
}

// NewManager creates a new tunnel service manager
func NewManager(cfgMgr *config.Manager) *Manager {
	return &Manager{
		tunnels: make(map[string]*TunnelState),
		cfgMgr:  cfgMgr,
	}
}

// Start starts a tunnel
func (m *Manager) Start(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if state, exists := m.tunnels[id]; exists && state.Status == "running" {
		return fmt.Errorf("tunnel already running")
	}

	// Get tunnel configuration
	tunnelCfg, err := m.cfgMgr.GetTunnel(id)
	if err != nil {
		return err
	}

	// Create tunnel service based on type
	var service TunnelService
	switch tunnelCfg.Type {
	case config.TunnelTypeCloudflare:
		service = NewCloudflareService(tunnelCfg)
	case config.TunnelTypeNgrok:
		service = NewNgrokService(tunnelCfg)
	default:
		return fmt.Errorf("unsupported tunnel type: %s", tunnelCfg.Type)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())

	// Create state
	state := &TunnelState{
		ID:        id,
		Status:    "starting",
		StartedAt: time.Now(),
		ctx:       ctx,
		cancel:    cancel,
		service:   service,
	}

	m.tunnels[id] = state

	// Start tunnel in goroutine
	go func() {
		logger.Sugar.Infof("Starting tunnel: %s (%s)", tunnelCfg.Name, tunnelCfg.Type)

		if err := service.Start(ctx); err != nil {
			m.mu.Lock()
			state.Status = "error"
			state.Error = err.Error()
			m.mu.Unlock()
			logger.Sugar.Errorf("Tunnel error: %v", err)
			return
		}

		m.mu.Lock()
		state.Status = "running"
		state.PublicURL = service.GetPublicURL()
		m.mu.Unlock()

		logger.Sugar.Infof("Tunnel running: %s -> %s", tunnelCfg.Name, state.PublicURL)

		// Wait for context cancellation
		<-ctx.Done()

		m.mu.Lock()
		state.Status = "stopped"
		m.mu.Unlock()

		logger.Sugar.Infof("Tunnel stopped: %s", tunnelCfg.Name)
	}()

	return nil
}

// Stop stops a tunnel
func (m *Manager) Stop(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.tunnels[id]
	if !exists {
		return fmt.Errorf("tunnel not found")
	}

	// Check actual service status instead of cached status
	if state.service != nil && state.service.GetStatus() == "stopped" {
		return nil
	}

	logger.Sugar.Infof("Stopping tunnel: %s", id)

	// Cancel context
	if state.cancel != nil {
		state.cancel()
	}

	// Stop service
	if state.service != nil {
		if err := state.service.Stop(); err != nil {
			logger.Sugar.Warnf("Error stopping tunnel service: %v", err)
		}
	}

	state.Status = "stopped"
	return nil
}

// GetStatus returns the status of a tunnel
func (m *Manager) GetStatus(id string) (*TunnelState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.tunnels[id]
	if !exists {
		return &TunnelState{
			ID:     id,
			Status: "stopped",
		}, nil
	}

	// Return a copy with current service status
	return &TunnelState{
		ID:        state.ID,
		Status:    state.service.GetStatus(),
		PublicURL: state.service.GetPublicURL(),
		StartedAt: state.StartedAt,
		Error:     state.service.GetError(),
	}, nil
}

// GetAllStatuses returns the status of all tunnels
func (m *Manager) GetAllStatuses() map[string]*TunnelState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*TunnelState)
	for id, state := range m.tunnels {
		result[id] = &TunnelState{
			ID:        state.ID,
			Status:    state.service.GetStatus(),
			PublicURL: state.service.GetPublicURL(),
			StartedAt: state.StartedAt,
			Error:     state.service.GetError(),
		}
	}

	return result
}

// StopAll stops all running tunnels
func (m *Manager) StopAll() error {
	m.mu.RLock()
	ids := make([]string, 0, len(m.tunnels))
	for id := range m.tunnels {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		if err := m.Stop(id); err != nil {
			logger.Sugar.Warnf("Error stopping tunnel %s: %v", id, err)
		}
	}

	return nil
}
