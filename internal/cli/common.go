package cli

import (
	"context"

	"github.com/spf13/viper"
	"github.com/strrl/gh-impersonate/internal/impersonate"
)

func loadSelectedProfile(v *viper.Viper) (string, impersonate.Profile, error) {
	profileName := selectedProfile(v)
	cfg, err := impersonate.LoadConfig()
	if err != nil {
		return "", impersonate.Profile{}, err
	}
	profile, err := impersonate.ResolveProfile(cfg, profileName, SharedClientID)
	if err != nil {
		return "", impersonate.Profile{}, err
	}
	return profileName, profile, nil
}

func resolveCredential(ctx context.Context, v *viper.Viper) (string, impersonate.Credential, error) {
	profileName, profile, err := loadSelectedProfile(v)
	if err != nil {
		return "", impersonate.Credential{}, err
	}
	resolver := impersonate.NewResolver()
	credential, err := resolver.Resolve(ctx, profileName, profile)
	if err != nil {
		return "", impersonate.Credential{}, err
	}
	return profileName, credential, nil
}
