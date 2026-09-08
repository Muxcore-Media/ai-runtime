package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Muxcore-Media/core/pkg/contracts"
	"google.golang.org/grpc"
)

const moduleVersion = "0.1.0"

type Module struct {
	id, grpcAddr, httpAddr string
	preferred              string
	openaiKey              string
	openaiBase             string
	openaiModel            string
	cfgMu                  sync.RWMutex
	router                 Router

	grpcSrv *grpc.Server
	lis     net.Listener
	httpSrv *http.Server
}

type Config struct {
	ID, GRPCAddr, HTTPAddr string
	Preferred              string
	OpenAIKey              string
	OpenAIBase             string
	OpenAIModel            string
}

func NewModule() *Module { return New(Config{}) }

func New(cfg Config) *Module {
	if cfg.ID == "" {
		cfg.ID = "ai-runtime"
	}
	if cfg.GRPCAddr == "" {
		cfg.GRPCAddr = "127.0.0.1:9760"
	}
	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = "127.0.0.1:9761"
	}
	if cfg.Preferred == "" {
		cfg.Preferred = os.Getenv("AI_RUNTIME_PROVIDER")
	}
	if cfg.OpenAIKey == "" {
		cfg.OpenAIKey = firstEnv("AI_RUNTIME_API_KEY", "OPENAI_API_KEY")
	}
	if cfg.OpenAIBase == "" {
		cfg.OpenAIBase = os.Getenv("AI_RUNTIME_BASE_URL")
	}
	if cfg.OpenAIModel == "" {
		cfg.OpenAIModel = os.Getenv("AI_RUNTIME_MODEL")
	}
	if v := os.Getenv("AI_RUNTIME_GRPC_ADDR"); v != "" {
		cfg.GRPCAddr = v
	}
	if v := os.Getenv("MUXCORE_HTTP_ADDR"); v != "" {
		cfg.HTTPAddr = v
	}
	m := &Module{
		id: cfg.ID, grpcAddr: cfg.GRPCAddr, httpAddr: cfg.HTTPAddr,
		preferred: cfg.Preferred, openaiKey: cfg.OpenAIKey,
		openaiBase: cfg.OpenAIBase, openaiModel: cfg.OpenAIModel,
	}
	m.rebuildRouter()
	return m
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func (m *Module) rebuildRouter() {
	r := Router{Default: HeuristicProvider{}, Preferred: m.preferred}
	if m.openaiKey != "" {
		r.OpenAI = OpenAIProvider{BaseURL: m.openaiBase, APIKey: m.openaiKey, Model: m.openaiModel}
	}
	m.router = r
}

func (m *Module) Info() contracts.ModuleInfo {
	return contracts.ModuleInfo{
		ID: m.id, Name: "AI Runtime", Version: moduleVersion,
		Roles:          []string{"ai"},
		Description:    "Shared AI inference gateway (complete, embed, transcribe, classify)",
		Author:         "Muxcore-Media",
		Capabilities:   []string{"ai.runtime", "settings"},
		MinCoreVersion: MinCoreVersion,
		HTTPAddr:       m.grpcAddr,
	}
}

func (m *Module) Init(context.Context) error { return nil }

func (m *Module) Start(ctx context.Context) error {
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", m.grpcAddr)
	if err != nil {
		return fmt.Errorf("listen gRPC %s: %w", m.grpcAddr, err)
	}
	m.lis = lis
	m.grpcAddr = lis.Addr().String()
	m.grpcSrv = grpc.NewServer()
	registerAIMesh(m.grpcSrv, m.id, m)
	go func() {
		slog.Info("ai-runtime gRPC listening", "addr", m.grpcAddr)
		if serveErr := m.grpcSrv.Serve(lis); serveErr != nil {
			slog.Error("gRPC serve", "error", serveErr)
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", m.handleHealth)
	mux.HandleFunc("GET /healthz", m.handleHealth)
	m.registerHTTP(mux)
	httpLis, err := lc.Listen(ctx, "tcp", m.httpAddr)
	if err != nil {
		return fmt.Errorf("listen HTTP %s: %w", m.httpAddr, err)
	}
	m.httpAddr = httpLis.Addr().String()
	m.httpSrv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		slog.Info("ai-runtime HTTP listening", "addr", m.httpAddr)
		if serveErr := m.httpSrv.Serve(httpLis); serveErr != nil && serveErr != http.ErrServerClosed {
			slog.Error("HTTP serve", "error", serveErr)
		}
	}()
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	if m.grpcSrv != nil {
		m.grpcSrv.GracefulStop()
	}
	if m.httpSrv != nil {
		return m.httpSrv.Shutdown(ctx)
	}
	return nil
}

func (m *Module) Health(context.Context) error { return nil }

func (m *Module) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := m.Health(r.Context()); err != nil {
		http.Error(w, `{"error":"unhealthy"}`, http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (m *Module) GRPCListenAddr() string { return m.grpcAddr }
func (m *Module) HTTPListenAddr() string { return m.httpAddr }

func (m *Module) Settings() []contracts.SettingDef {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return []contracts.SettingDef{
		{
			Key: "provider", Label: "Preferred provider", Type: contracts.SettingTypeString,
			Value: m.preferred, Default: "heuristic",
			Description: "heuristic (offline) or openai (requires API key)", Group: "AI",
		},
		{
			Key: "openai_model", Label: "OpenAI model", Type: contracts.SettingTypeString,
			Value: m.openaiModel, Default: "gpt-4o-mini", Group: "AI",
		},
	}
}

func (m *Module) UpdateSetting(key, value string) error {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	switch key {
	case "provider":
		m.preferred = value
	case "openai_model":
		m.openaiModel = value
	default:
		return fmt.Errorf("unknown setting %q", key)
	}
	m.rebuildRouter()
	return nil
}

func (m *Module) provider(name string) Provider {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return m.router.pick(name)
}
