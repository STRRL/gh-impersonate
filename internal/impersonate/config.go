package impersonate

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/viper"
)

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type Config struct {
	Profiles map[string]Profile `mapstructure:"profiles"`
}

type Profile struct {
	ClientID string `mapstructure:"client_id"`
}

func LoadConfig() (Config, error) {
	v := viper.New()
	v.SetConfigFile(ConfigPath())
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !errors.Is(err, os.ErrNotExist) {
			return Config{}, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	for name, profile := range cfg.Profiles {
		if !profileNamePattern.MatchString(name) {
			return Config{}, fmt.Errorf("invalid profile name %q", name)
		}
		if profile.ClientID == "" {
			return Config{}, fmt.Errorf("profile %q is missing client_id", name)
		}
	}
	return cfg, nil
}

func ResolveProfile(cfg Config, name string, sharedClientID string) (Profile, error) {
	if name == "" {
		name = "default"
	}
	if !profileNamePattern.MatchString(name) {
		return Profile{}, fmt.Errorf("invalid profile name %q", name)
	}
	if profile, ok := cfg.Profiles[name]; ok {
		return profile, nil
	}
	if name == "default" && sharedClientID != "" {
		return Profile{ClientID: sharedClientID}, nil
	}
	if name == "default" {
		return Profile{}, fmt.Errorf("profile %q is not configured and this build has no shared GitHub App client_id; add profiles.default.client_id to %s", name, ConfigPath())
	}
	return Profile{}, fmt.Errorf("profile %q is not configured in %s", name, ConfigPath())
}
