package cmd

import (
	"context"
	"fmt"

	"github.com/yeisme/taskbridge/internal/loader"
	"github.com/yeisme/taskbridge/internal/provider"
)

func loadProvidersWithStatus(target string) *loader.ProviderLoadResult {
	if target == "" {
		return loader.LoadProviders(context.Background())
	}
	return loader.LoadProviders(context.Background(), loader.WithTarget(target))
}

func loadAuthenticatedProviders(target string) (map[string]provider.Provider, error) {
	target = provider.ResolveProviderName(target)
	result := loadProvidersWithStatus(target)
	if target == "" {
		return result.Providers, nil
	}

	status, ok := result.Statuses[target]
	if !ok {
		return nil, fmt.Errorf("provider %s not found or not authenticated", target)
	}
	if !status.Authenticated {
		if status.Error != "" {
			return nil, fmt.Errorf("%s", status.Error)
		}
		return nil, fmt.Errorf("provider %s not found or not authenticated", target)
	}

	return result.Providers, nil
}
