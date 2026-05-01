package impersonate

import (
	"os"
	"path/filepath"
)

func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "gh-impersonate")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	return filepath.Join(home, ".config", "gh-impersonate")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func CredentialsDir() string {
	return filepath.Join(ConfigDir(), "credentials")
}

func CredentialPath(profile string) string {
	return filepath.Join(CredentialsDir(), profile+".json")
}

func CredentialLockPath(profile string) string {
	return filepath.Join(CredentialsDir(), profile+".lock")
}
