// Package loader provides a shared provider-loading contract reused by serve and sync.
package loader

import (
	"context"
	"fmt"

	"github.com/yeisme/taskbridge/internal/provider"
	"github.com/yeisme/taskbridge/internal/provider/feishu"
	"github.com/yeisme/taskbridge/internal/provider/google"
	"github.com/yeisme/taskbridge/internal/provider/microsoft"
	"github.com/yeisme/taskbridge/internal/provider/ticktick"
	"github.com/yeisme/taskbridge/internal/provider/todoist"
)

// ProviderLoadStatus records the outcome of loading a single provider.
type ProviderLoadStatus struct {
	Name          string `json:"name"`
	Loaded        bool   `json:"loaded"`
	Authenticated bool   `json:"authenticated"`
	Error         string `json:"error,omitempty"`
}

// ProviderLoadResult is returned by LoadProviders and consumed by both sync and serve.
type ProviderLoadResult struct {
	Providers map[string]provider.Provider   `json:"-"`
	Statuses  map[string]*ProviderLoadStatus `json:"statuses"`
}

// LoadOption configures provider loading behaviour.
type LoadOption func(*loadConfig)

type loadConfig struct {
	target string // empty means all providers
}

// WithTarget restricts loading to a single named provider.
func WithTarget(name string) LoadOption {
	return func(c *loadConfig) { c.target = name }
}

// allProviders lists every known provider name in canonical order.
var allProviders = []string{"google", "microsoft", "todoist", "feishu", "ticktick", "dida"}

// LoadProviders loads providers according to opts and returns both the
// successfully loaded instances and per-provider status metadata.
func LoadProviders(ctx context.Context, opts ...LoadOption) *ProviderLoadResult {
	cfg := &loadConfig{}
	for _, o := range opts {
		o(cfg)
	}

	result := &ProviderLoadResult{
		Providers: make(map[string]provider.Provider),
		Statuses:  make(map[string]*ProviderLoadStatus),
	}

	targets := allProviders
	if cfg.target != "" {
		resolved := provider.ResolveProviderName(cfg.target)
		targets = []string{resolved}
	}

	for _, name := range targets {
		p, status := loadOne(ctx, name)
		result.Statuses[name] = status
		if p != nil {
			result.Providers[name] = p
		}
	}

	return result
}

// loadOne attempts to create and authenticate a single provider.
func loadOne(ctx context.Context, name string) (provider.Provider, *ProviderLoadStatus) {
	status := &ProviderLoadStatus{Name: name}

	var p provider.Provider
	var err error

	switch name {
	case "google":
		p, err = loadGoogle()
	case "microsoft":
		p, err = loadMicrosoft()
	case "todoist":
		p, err = loadTodoist(ctx)
	case "feishu":
		p, err = loadFeishu()
	case "ticktick":
		p, err = loadTicktick(ctx, "ticktick")
	case "dida":
		p, err = loadTicktick(ctx, "dida")
	default:
		status.Error = fmt.Sprintf("unknown provider: %s", name)
		return nil, status
	}

	if err != nil {
		status.Error = err.Error()
		return nil, status
	}

	status.Loaded = true
	status.Authenticated = true
	return p, status
}

func loadGoogle() (*google.Provider, error) {
	p, err := google.NewProviderFromHome()
	if err != nil {
		return nil, fmt.Errorf("初始化 Google Provider 失败: %w", err)
	}
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("google Provider 未认证，请运行 'taskbridge auth google' 进行认证")
	}
	return p, nil
}

func loadMicrosoft() (*microsoft.Provider, error) {
	p, err := microsoft.NewProviderFromHome()
	if err != nil {
		return nil, fmt.Errorf("初始化 Microsoft Provider 失败: %w", err)
	}
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("microsoft Provider 未认证，请运行 'taskbridge auth microsoft' 进行认证")
	}
	return p, nil
}

func loadTodoist(ctx context.Context) (*todoist.Provider, error) {
	p, err := todoist.NewProviderFromHome()
	if err != nil {
		return nil, fmt.Errorf("初始化 Todoist Provider 失败: %w", err)
	}
	if err := p.Authenticate(ctx, nil); err != nil {
		return nil, fmt.Errorf("todoist Provider 未认证，请运行 'taskbridge auth login todoist' 进行认证")
	}
	return p, nil
}

func loadFeishu() (*feishu.Provider, error) {
	p, err := feishu.NewProviderFromHome()
	if err != nil {
		return nil, fmt.Errorf("初始化 Feishu Provider 失败: %w", err)
	}
	if !p.IsAuthenticated() {
		return nil, fmt.Errorf("feishu Provider 未认证，请运行 'taskbridge auth login feishu' 进行认证")
	}
	return p, nil
}

func loadTicktick(ctx context.Context, providerName string) (*ticktick.Provider, error) {
	p, err := ticktick.NewProviderFromHomeByName(providerName)
	if err != nil {
		return nil, fmt.Errorf("初始化 %s Provider 失败: %w", providerName, err)
	}
	if err := p.Authenticate(ctx, nil); err != nil {
		return nil, fmt.Errorf("%s Provider 未认证: %w", providerName, err)
	}
	return p, nil
}
