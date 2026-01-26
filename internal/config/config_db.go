package config

import (
	"context"
	"fmt"
	"pont/ent"
	"pont/ent/setting"
	"pont/ent/tunnel"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TunnelType represents the type of tunnel
type TunnelType string

const (
	TunnelTypeCloudflare TunnelType = "cloudflare"
	TunnelTypeNgrok      TunnelType = "ngrok"
)

// TunnelConfig represents a single tunnel configuration
type TunnelConfig struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       TunnelType `json:"type"`
	Target     string     `json:"target"`
	Enabled    bool       `json:"enabled"`
	MCPEnabled bool       `json:"mcp_enabled"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Ngrok-specific fields
	NgrokAuthtoken string `json:"ngrok_authtoken,omitempty"`
	NgrokDomain    string `json:"ngrok_domain,omitempty"`
}

// Settings represents global application settings
type Settings struct {
	AutoStart bool   `json:"auto_start"`
	LogLevel  string `json:"log_level"`
}

// Manager manages configuration with database storage
type Manager struct {
	mu     sync.RWMutex
	client *ent.Client
}

// NewManager creates a new configuration manager
func NewManager(client *ent.Client) *Manager {
	return &Manager{client: client}
}

// GetAllTunnels returns all tunnel configurations
func (m *Manager) GetAllTunnels() ([]TunnelConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnels, err := m.client.Tunnel.Query().
		Order(ent.Desc(tunnel.FieldCreatedAt)).
		All(context.Background())
	if err != nil {
		return nil, err
	}

	configs := make([]TunnelConfig, len(tunnels))
	for i, t := range tunnels {
		configs[i] = TunnelConfig{
			ID:             t.ID.String(),
			Name:           t.Name,
			Type:           TunnelType(t.Type),
			Target:         t.Target,
			Enabled:        t.Enabled,
			MCPEnabled:     t.McpEnabled,
			CreatedAt:      t.CreatedAt,
			UpdatedAt:      t.UpdatedAt,
			NgrokAuthtoken: stringPtrToString(t.NgrokAuthtoken),
			NgrokDomain:    stringPtrToString(t.NgrokDomain),
		}
	}

	return configs, nil
}

// GetTunnel returns a specific tunnel configuration
func (m *Manager) GetTunnel(id string) (*TunnelConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid tunnel id: %w", err)
	}

	t, err := m.client.Tunnel.Get(context.Background(), uid)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("tunnel not found: %s", id)
		}
		return nil, err
	}

	return &TunnelConfig{
		ID:             t.ID.String(),
		Name:           t.Name,
		Type:           TunnelType(t.Type),
		Target:         t.Target,
		Enabled:        t.Enabled,
		MCPEnabled:     t.McpEnabled,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		NgrokAuthtoken: stringPtrToString(t.NgrokAuthtoken),
		NgrokDomain:    stringPtrToString(t.NgrokDomain),
	}, nil
}

// AddTunnel adds a new tunnel configuration
func (m *Manager) AddTunnel(tunnelCfg *TunnelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.validateTunnel(tunnelCfg); err != nil {
		return err
	}

	var uid uuid.UUID
	if tunnelCfg.ID == "" {
		uid = uuid.New()
		tunnelCfg.ID = uid.String()
	} else {
		var err error
		uid, err = uuid.Parse(tunnelCfg.ID)
		if err != nil {
			return fmt.Errorf("invalid tunnel id: %w", err)
		}
	}

	builder := m.client.Tunnel.Create().
		SetID(uid).
		SetName(tunnelCfg.Name).
		SetType(tunnel.Type(tunnelCfg.Type)).
		SetTarget(tunnelCfg.Target).
		SetEnabled(tunnelCfg.Enabled).
		SetMcpEnabled(tunnelCfg.MCPEnabled)

	if tunnelCfg.NgrokAuthtoken != "" {
		builder.SetNillableNgrokAuthtoken(&tunnelCfg.NgrokAuthtoken)
	}
	if tunnelCfg.NgrokDomain != "" {
		builder.SetNillableNgrokDomain(&tunnelCfg.NgrokDomain)
	}

	t, err := builder.Save(context.Background())
	if err != nil {
		return err
	}

	tunnelCfg.CreatedAt = t.CreatedAt
	tunnelCfg.UpdatedAt = t.UpdatedAt

	return nil
}

// UpdateTunnel updates an existing tunnel configuration
func (m *Manager) UpdateTunnel(id string, tunnelCfg *TunnelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.validateTunnel(tunnelCfg); err != nil {
		return err
	}

	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid tunnel id: %w", err)
	}

	builder := m.client.Tunnel.UpdateOneID(uid).
		SetName(tunnelCfg.Name).
		SetType(tunnel.Type(tunnelCfg.Type)).
		SetTarget(tunnelCfg.Target).
		SetEnabled(tunnelCfg.Enabled).
		SetMcpEnabled(tunnelCfg.MCPEnabled)

	if tunnelCfg.NgrokAuthtoken != "" {
		builder.SetNillableNgrokAuthtoken(&tunnelCfg.NgrokAuthtoken)
	} else {
		builder.ClearNgrokAuthtoken()
	}

	if tunnelCfg.NgrokDomain != "" {
		builder.SetNillableNgrokDomain(&tunnelCfg.NgrokDomain)
	} else {
		builder.ClearNgrokDomain()
	}

	t, err := builder.Save(context.Background())
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("tunnel not found: %s", id)
		}
		return err
	}

	tunnelCfg.UpdatedAt = t.UpdatedAt

	return nil
}

// DeleteTunnel deletes a tunnel configuration
func (m *Manager) DeleteTunnel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid tunnel id: %w", err)
	}

	err = m.client.Tunnel.DeleteOneID(uid).Exec(context.Background())
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("tunnel not found: %s", id)
		}
		return err
	}

	return nil
}

// GetSettings returns global settings
func (m *Manager) GetSettings() (*Settings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	settings := &Settings{
		AutoStart: false,
		LogLevel:  "info",
	}

	settingsList, err := m.client.Setting.Query().All(context.Background())
	if err != nil {
		return settings, nil
	}

	for _, s := range settingsList {
		switch s.Key {
		case "auto_start":
			settings.AutoStart = s.Value == "true"
		case "log_level":
			settings.LogLevel = s.Value
		}
	}

	return settings, nil
}

// UpdateSettings updates global settings
func (m *Manager) UpdateSettings(settings *Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx := context.Background()

	autoStart := "false"
	if settings.AutoStart {
		autoStart = "true"
	}

	// Update or create auto_start
	existing, err := m.client.Setting.Query().Where(setting.KeyEQ("auto_start")).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if existing != nil {
		_, err = m.client.Setting.UpdateOne(existing).SetValue(autoStart).Save(ctx)
		if err != nil {
			return err
		}
	} else {
		_, err = m.client.Setting.Create().SetKey("auto_start").SetValue(autoStart).Save(ctx)
		if err != nil {
			return err
		}
	}

	// Update or create log_level
	existing, err = m.client.Setting.Query().Where(setting.KeyEQ("log_level")).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}
	if existing != nil {
		_, err = m.client.Setting.UpdateOne(existing).SetValue(settings.LogLevel).Save(ctx)
		if err != nil {
			return err
		}
	} else {
		_, err = m.client.Setting.Create().SetKey("log_level").SetValue(settings.LogLevel).Save(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// validateTunnel validates a tunnel configuration
func (m *Manager) validateTunnel(tunnel *TunnelConfig) error {
	if tunnel.Name == "" {
		return fmt.Errorf("tunnel name is required")
	}

	if tunnel.Type != TunnelTypeCloudflare && tunnel.Type != TunnelTypeNgrok {
		return fmt.Errorf("invalid tunnel type: %s", tunnel.Type)
	}

	if tunnel.Target == "" {
		return fmt.Errorf("tunnel target is required")
	}

	return nil
}

func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
