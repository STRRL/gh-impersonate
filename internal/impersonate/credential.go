package impersonate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Credential struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	ExpiresAt             time.Time `json:"expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	TokenType             string    `json:"token_type,omitempty"`
	Scope                 string    `json:"scope,omitempty"`
}

func LoadCredential(profile string) (Credential, error) {
	data, err := os.ReadFile(CredentialPath(profile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credential{}, ErrCredentialMissing
		}
		return Credential{}, err
	}
	var credential Credential
	if err := json.Unmarshal(data, &credential); err != nil {
		return Credential{}, err
	}
	if credential.AccessToken == "" || credential.RefreshToken == "" {
		return Credential{}, fmt.Errorf("credential for profile %q is incomplete", profile)
	}
	return credential, nil
}

func SaveCredential(profile string, credential Credential) error {
	if err := os.MkdirAll(CredentialsDir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(credential, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	path := CredentialPath(profile)
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func DeleteCredential(profile string) error {
	err := os.Remove(CredentialPath(profile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

var ErrCredentialMissing = errors.New("credential missing")
