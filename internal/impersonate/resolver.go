package impersonate

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Resolver struct {
	Client *GitHubClient
}

func NewResolver() *Resolver {
	return &Resolver{Client: NewGitHubClient()}
}

func (r *Resolver) Resolve(ctx context.Context, profileName string, profile Profile) (Credential, error) {
	lock, err := LockCredential(profileName)
	if err != nil {
		return Credential{}, err
	}
	defer lock.Unlock()

	credential, err := LoadCredential(profileName)
	if err != nil {
		if errors.Is(err, ErrCredentialMissing) {
			return Credential{}, LoginRequiredError{Profile: profileName}
		}
		return Credential{}, err
	}
	if credential.ExpiresAt.IsZero() || time.Now().Before(credential.ExpiresAt) {
		return credential, nil
	}
	if credential.RefreshToken == "" {
		return Credential{}, LoginRequiredError{Profile: profileName}
	}
	if !credential.RefreshTokenExpiresAt.IsZero() && time.Now().After(credential.RefreshTokenExpiresAt) {
		return Credential{}, LoginRequiredError{Profile: profileName}
	}

	refreshed, err := r.Client.RefreshCredential(ctx, profile.ClientID, credential)
	if err != nil {
		return Credential{}, fmt.Errorf("refresh credential for profile %q: %w", profileName, err)
	}
	if err := SaveCredential(profileName, refreshed); err != nil {
		return Credential{}, err
	}
	return refreshed, nil
}

type LoginRequiredError struct {
	Profile string
}

func (e LoginRequiredError) Error() string {
	return fmt.Sprintf("profile %q is not logged in; run: gh impersonate --profile %s auth login", e.Profile, e.Profile)
}
