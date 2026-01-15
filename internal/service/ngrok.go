package service

import (
	"context"
	"errors"
	"fmt"
	"pont/internal/config"
	"pont/internal/logger"
	"strings"
	"time"

	"golang.ngrok.com/ngrok/v2"
)

// NgrokService implements ngrok tunnel
type NgrokService struct {
	config    *config.TunnelConfig
	agent     ngrok.Agent
	forwarder ngrok.EndpointForwarder
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

	// Create agent with authtoken
	var agentOpts []ngrok.AgentOption
	if ns.config.NgrokAuthtoken != "" {
		agentOpts = append(agentOpts, ngrok.WithAuthtoken(ns.config.NgrokAuthtoken))
	}

	agent, err := ngrok.NewAgent(agentOpts...)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create agent: %v", err)
		ns.lastError = errMsg
		ns.status = "error"
		return fmt.Errorf("%s", errMsg)
	}
	ns.agent = agent

	// Check protocol
	if strings.HasPrefix(ns.config.Target, "tcp://") {
		target := strings.TrimPrefix(ns.config.Target, "tcp://")
		return ns.startTCP(target)
	}
	if strings.HasPrefix(ns.config.Target, "tls://") {
		target := strings.TrimPrefix(ns.config.Target, "tls://")
		return ns.startTLS(target)
	}
	return ns.startHTTP()
}

func (ns *NgrokService) startHTTP() error {
	// Build endpoint options
	var opts []ngrok.EndpointOption
	if ns.config.NgrokDomain != "" {
		opts = append(opts, ngrok.WithURL(ns.config.NgrokDomain))
	}

	logger.Sugar.Infof("Connecting to ngrok...")

	// Create a channel to receive the result
	type result struct {
		forwarder ngrok.EndpointForwarder
		err       error
	}
	resultCh := make(chan result, 1)

	// Start connection in a goroutine with timeout
	go func() {
		forwarder, err := ns.agent.Forward(ns.ctx, ngrok.WithUpstream(ns.config.Target), opts...)
		resultCh <- result{forwarder: forwarder, err: err}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultCh:
		if res.err != nil {
			errMsg := fmt.Sprintf("Failed to start tunnel: %v", res.err)
			// Check if it's ngrok error with code
			var ngrokErr ngrok.Error
			if errors.As(res.err, &ngrokErr) && ngrokErr.Code() == "ERR_NGROK_108" {
				errMsg = "Free ngrok accounts can only run one tunnel at a time. Please stop other tunnels first."
			}
			ns.lastError = errMsg
			ns.status = "error"
			logger.Sugar.Errorf("Ngrok connection failed: %v", res.err)
			return fmt.Errorf("%s", errMsg)
		}
		ns.forwarder = res.forwarder
		ns.publicURL = res.forwarder.URL().String()
		ns.status = "running"
		logger.Sugar.Infof("Ngrok tunnel created: %s -> %s", ns.publicURL, ns.config.Target)
	case <-time.After(30 * time.Second):
		errMsg := "Ngrok connection timeout. Possible causes: 1) Network issue 2) Invalid authtoken 3) Free account limit: only 1 endpoint allowed, please stop other tunnels first"
		ns.lastError = errMsg
		ns.status = "error"
		logger.Sugar.Error(errMsg)
		if ns.cancel != nil {
			ns.cancel()
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

func (ns *NgrokService) startTCP(target string) error {
	logger.Sugar.Infof("Connecting to ngrok (TCP)...")

	// Create a channel to receive the result
	type result struct {
		forwarder ngrok.EndpointForwarder
		err       error
	}
	resultCh := make(chan result, 1)

	// Start connection in a goroutine with timeout
	go func() {
		forwarder, err := ns.agent.Forward(ns.ctx, ngrok.WithUpstream("tcp://"+target), ngrok.WithURL("tcp://"))
		resultCh <- result{forwarder: forwarder, err: err}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultCh:
		if res.err != nil {
			errMsg := fmt.Sprintf("Failed to start TCP tunnel: %v", res.err)
			var ngrokErr ngrok.Error
			if errors.As(res.err, &ngrokErr) && ngrokErr.Code() == "ERR_NGROK_108" {
				errMsg = "Free ngrok accounts can only run one tunnel at a time. Please stop other tunnels first."
			}
			ns.lastError = errMsg
			ns.status = "error"
			logger.Sugar.Errorf("Ngrok TCP connection failed: %v", res.err)
			return fmt.Errorf("%s", errMsg)
		}
		ns.forwarder = res.forwarder
		ns.publicURL = res.forwarder.URL().String()
		ns.status = "running"
		logger.Sugar.Infof("Ngrok TCP tunnel created: %s -> %s", ns.publicURL, target)
	case <-time.After(30 * time.Second):
		errMsg := "Ngrok TCP connection timeout. Possible causes: 1) Network issue 2) Invalid authtoken 3) Free account limit: only 1 endpoint allowed, please stop other tunnels first"
		ns.lastError = errMsg
		ns.status = "error"
		logger.Sugar.Error(errMsg)
		if ns.cancel != nil {
			ns.cancel()
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

func (ns *NgrokService) startTLS(target string) error {
	logger.Sugar.Infof("Connecting to ngrok (TLS)...")

	type result struct {
		forwarder ngrok.EndpointForwarder
		err       error
	}
	resultCh := make(chan result, 1)

	go func() {
		forwarder, err := ns.agent.Forward(ns.ctx, ngrok.WithUpstream("tls://"+target), ngrok.WithURL("tls://"))
		resultCh <- result{forwarder: forwarder, err: err}
	}()

	select {
	case res := <-resultCh:
		if res.err != nil {
			errMsg := fmt.Sprintf("Failed to start TLS tunnel: %v", res.err)
			var ngrokErr ngrok.Error
			if errors.As(res.err, &ngrokErr) && ngrokErr.Code() == "ERR_NGROK_108" {
				errMsg = "Free ngrok accounts can only run one tunnel at a time. Please stop other tunnels first."
			}
			ns.lastError = errMsg
			ns.status = "error"
			logger.Sugar.Errorf("Ngrok TLS connection failed: %v", res.err)
			return fmt.Errorf("%s", errMsg)
		}
		ns.forwarder = res.forwarder
		ns.publicURL = res.forwarder.URL().String()
		ns.status = "running"
		logger.Sugar.Infof("Ngrok TLS tunnel created: %s -> %s", ns.publicURL, target)
	case <-time.After(30 * time.Second):
		errMsg := "Ngrok TLS connection timeout. Possible causes: 1) Network issue 2) Invalid authtoken 3) Free account limit: only 1 endpoint allowed, please stop other tunnels first"
		ns.lastError = errMsg
		ns.status = "error"
		logger.Sugar.Error(errMsg)
		if ns.cancel != nil {
			ns.cancel()
		}
		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

// Stop stops the ngrok tunnel
func (ns *NgrokService) Stop() error {
	if ns.cancel != nil {
		ns.cancel()
	}

	ns.status = "stopped"
	ns.publicURL = ""

	if ns.forwarder != nil {
		ns.forwarder.Close()
	}

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
