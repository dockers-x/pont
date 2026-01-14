package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"pont/internal/config"
	"pont/internal/logger"
	"time"

	"golang.ngrok.com/ngrok"
	ngrokconfig "golang.ngrok.com/ngrok/config"
)

// NgrokService implements ngrok tunnel
type NgrokService struct {
	config    *config.TunnelConfig
	tunnel    ngrok.Tunnel
	publicURL string
	status    string
	lastError string
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewNgrokService creates a new ngrok tunnel service
func NewNgrokService(cfg *config.TunnelConfig) *NgrokService {
	return &NgrokService{
		config: cfg,
		status: "stopped",
	}
}

// Start starts the ngrok tunnel
func (ns *NgrokService) Start(ctx context.Context) error {
	ns.ctx, ns.cancel = context.WithCancel(ctx)

	// Parse target URL
	targetURL, err := url.Parse(ns.config.Target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	// Build tunnel config options
	tunnelOpts := []ngrokconfig.HTTPEndpointOption{}

	// Add custom domain if provided
	if ns.config.NgrokDomain != "" {
		tunnelOpts = append(tunnelOpts, ngrokconfig.WithDomain(ns.config.NgrokDomain))
	}

	// Create tunnel configuration
	tunnelConfig := ngrokconfig.HTTPEndpoint(tunnelOpts...)

	// Build connect options
	connectOpts := []ngrok.ConnectOption{}

	// Add authtoken if provided
	if ns.config.NgrokAuthtoken != "" {
		connectOpts = append(connectOpts, ngrok.WithAuthtoken(ns.config.NgrokAuthtoken))
	}

	logger.Sugar.Infof("Connecting to ngrok with authtoken: %s...", ns.config.NgrokAuthtoken[:min(10, len(ns.config.NgrokAuthtoken))])

	// Create a channel to receive the result
	type result struct {
		tunnel ngrok.Tunnel
		err    error
	}
	resultCh := make(chan result, 1)

	// Start connection in a goroutine with timeout
	go func() {
		tunnel, err := ngrok.Listen(ns.ctx, tunnelConfig, connectOpts...)
		resultCh <- result{tunnel: tunnel, err: err}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultCh:
		if res.err != nil {
			errMsg := fmt.Sprintf("Failed to start tunnel: %v", res.err)
			ns.lastError = errMsg
			ns.status = "error"
			logger.Sugar.Errorf("Ngrok connection failed: %v", res.err)
			return fmt.Errorf(errMsg)
		}
		ns.tunnel = res.tunnel
		ns.publicURL = res.tunnel.URL()
		ns.status = "running"
		logger.Sugar.Infof("Ngrok tunnel created: %s -> %s", ns.publicURL, ns.config.Target)
	case <-time.After(30 * time.Second):
		errMsg := "Ngrok connection timeout after 30 seconds. Please check your network connection and authtoken."
		ns.lastError = errMsg
		ns.status = "error"
		logger.Sugar.Error(errMsg)
		if ns.cancel != nil {
			ns.cancel()
		}
		return fmt.Errorf(errMsg)
	}

	// Start HTTP server to forward requests
	go func() {
		// Create HTTP client to forward requests to target
		client := &http.Client{}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Build target URL
			targetReq := r.Clone(ns.ctx)
			targetReq.URL.Scheme = targetURL.Scheme
			targetReq.URL.Host = targetURL.Host
			targetReq.RequestURI = ""

			// Forward request
			resp, err := client.Do(targetReq)
			if err != nil {
				logger.Sugar.Errorf("Error forwarding request: %v", err)
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()

			// Copy response headers
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}

			// Copy status code
			w.WriteHeader(resp.StatusCode)

			// Copy body
			if _, err := w.Write([]byte{}); err != nil {
				logger.Sugar.Errorf("Error writing response: %v", err)
			}
		})

		if err := http.Serve(ns.tunnel, handler); err != nil {
			if ns.ctx.Err() == nil {
				logger.Sugar.Errorf("Ngrok HTTP server error: %v", err)
			}
		}
	}()

	return nil
}

// Stop stops the ngrok tunnel
func (ns *NgrokService) Stop() error {
	if ns.cancel != nil {
		ns.cancel()
	}

	if ns.tunnel != nil {
		if err := ns.tunnel.Close(); err != nil {
			return fmt.Errorf("failed to close ngrok tunnel: %w", err)
		}
	}

	ns.status = "stopped"
	return nil
}

// GetPublicURL returns the public URL
func (ns *NgrokService) GetPublicURL() string {
	return ns.publicURL
}

// GetStatus returns the current status
func (ns *NgrokService) GetStatus() string {
	return ns.status
}

// GetError returns the last error message
func (ns *NgrokService) GetError() string {
	return ns.lastError
}
