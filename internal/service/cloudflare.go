package service

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"pont/internal/config"
	"pont/internal/logger"
	"regexp"
	"strings"
	"sync"

	"github.com/cloudflare/cloudflared/cmd/cloudflared/cliutil"
	"github.com/cloudflare/cloudflared/cmd/cloudflared/tunnel"
	"github.com/cloudflare/cloudflared/cmd/cloudflared/updater"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
)

// safeRegisterer wraps a Prometheus registry and gracefully handles duplicate registrations
type safeRegisterer struct {
	prometheus.Registerer
}

func newSafeRegisterer(reg prometheus.Registerer) prometheus.Registerer {
	return &safeRegisterer{Registerer: reg}
}

func (s *safeRegisterer) Register(c prometheus.Collector) error {
	err := s.Registerer.Register(c)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "already registered") {
			return nil
		}
		return err
	}
	return nil
}

func (s *safeRegisterer) MustRegister(cs ...prometheus.Collector) {
	for _, c := range cs {
		if err := s.Register(c); err != nil {
			errStr := err.Error()
			if !strings.Contains(errStr, "duplicate") && !strings.Contains(errStr, "already registered") {
				panic(err)
			}
		}
	}
}

var urlPattern = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

type urlCapture struct {
	cs      *CloudflareService
	wrapped io.Writer
}

func (u *urlCapture) Write(p []byte) (n int, err error) {
	if u.wrapped != nil {
		u.wrapped.Write(p)
	}
	n = len(p)
	if u.cs.GetPublicURL() != "" {
		return
	}
	if match := urlPattern.Find(p); match != nil {
		u.cs.mu.Lock()
		if u.cs.publicURL == "" {
			u.cs.publicURL = string(match)
			u.cs.status = "running"
		}
		u.cs.mu.Unlock()
	}
	return
}

type CloudflareService struct {
	config            *config.TunnelConfig
	publicURL         string
	status            string
	lastError         error
	mu                sync.RWMutex
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	initOnce          sync.Once
	metricsRegistry   *prometheus.Registry
	gracefulShutdownC chan struct{}
}

func NewCloudflareService(cfg *config.TunnelConfig) *CloudflareService {
	return &CloudflareService{
		config:            cfg,
		status:            "stopped",
		gracefulShutdownC: make(chan struct{}, 1),
	}
}

func (cs *CloudflareService) initTunnel() {
	cs.initOnce.Do(func() {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Sugar.Errorf("Panic during tunnel initialization: %v", rec)
			}
		}()

		buildInfo := cliutil.GetBuildInfo("pont", "1.0.0")
		updater.Init(buildInfo)
		tunnel.Init(buildInfo, cs.gracefulShutdownC)
		logger.Sugar.Info("Cloudflared tunnel initialized")
	})
}

func (cs *CloudflareService) Start(ctx context.Context) error {
	defer func() {
		if rec := recover(); rec != nil {
			logger.Sugar.Errorf("Panic during tunnel start: %v", rec)
		}
	}()

	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.status == "running" || cs.status == "starting" {
		return fmt.Errorf("tunnel already running")
	}

	targetURL, err := url.Parse(cs.config.Target)
	if err != nil {
		return fmt.Errorf("invalid target URL: %w", err)
	}

	cs.initTunnel()

	cs.metricsRegistry = prometheus.NewRegistry()
	prometheus.DefaultRegisterer = newSafeRegisterer(cs.metricsRegistry)

	if cs.cancel != nil {
		cs.cancel()
	}

	tunnelCtx, cancel := context.WithCancel(ctx)
	cs.cancel = cancel
	cs.status = "starting"
	cs.lastError = nil

	cs.wg.Add(1)
	go cs.runTunnel(tunnelCtx, targetURL.String())

	return nil
}

func (cs *CloudflareService) runTunnel(ctx context.Context, targetURL string) {
	defer cs.wg.Done()
	defer func() {
		if rec := recover(); rec != nil {
			logger.Sugar.Errorf("Panic in tunnel: %v", rec)
			cs.mu.Lock()
			cs.lastError = fmt.Errorf("tunnel panic: %v", rec)
			cs.status = "error"
			cs.mu.Unlock()
		}
	}()

	defer func() {
		cs.mu.Lock()
		cs.status = "stopped"
		cs.publicURL = ""
		cs.mu.Unlock()
	}()

	// Redirect stdout/stderr to capture URL
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Start URL capture goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		capture := &urlCapture{cs: cs, wrapped: oldStdout}
		io.Copy(capture, r)
	}()

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		w.Close()
		<-done
	}()

	app := &cli.App{
		Name:     "cloudflared",
		Commands: tunnel.Commands(),
		ExitErrHandler: func(c *cli.Context, err error) {
			if err != nil {
				logger.Sugar.Errorf("CLI error: %v", err)
			}
		},
	}

	cli.OsExiter = func(exitCode int) {
		if exitCode != 0 {
			panic(fmt.Sprintf("CLI exit with code %d", exitCode))
		}
	}

	args := []string{"cloudflared", "tunnel", "--no-autoupdate", "--url", targetURL}

	logger.Sugar.Infof("Starting cloudflared tunnel: %s", targetURL)

	err := app.RunContext(ctx, args)

	if ctx.Err() != nil {
		logger.Sugar.Info("Tunnel stopped by user")
		return
	}

	if err != nil {
		logger.Sugar.Errorf("Tunnel error: %v", err)
		cs.mu.Lock()
		cs.lastError = err
		cs.status = "error"
		cs.mu.Unlock()
	}
}

func (cs *CloudflareService) Stop() error {
	cs.mu.Lock()
	if cs.status == "stopped" {
		cs.mu.Unlock()
		return nil
	}

	if cs.cancel != nil {
		cs.cancel()
	}

	select {
	case cs.gracefulShutdownC <- struct{}{}:
	default:
	}
	cs.mu.Unlock()

	cs.wg.Wait()
	return nil
}

func (cs *CloudflareService) GetPublicURL() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.publicURL
}

func (cs *CloudflareService) GetStatus() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.status
}

func (cs *CloudflareService) GetError() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	if cs.lastError != nil {
		return cs.lastError.Error()
	}
	return ""
}
