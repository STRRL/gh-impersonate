package impersonate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	githubDeviceCodeURL  = "https://github.com/login/device/code"
	githubAccessTokenURL = "https://github.com/login/oauth/access_token"
	githubUserURL        = "https://api.github.com/user"
)

type GitHubClient struct {
	httpClient *http.Client
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken           string `json:"access_token"`
	ExpiresIn             int    `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	TokenType             string `json:"token_type"`
	Scope                 string `json:"scope"`
	Error                 string `json:"error"`
	ErrorDescription      string `json:"error_description"`
}

type GitHubUser struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

func (c *GitHubClient) RequestDeviceCode(ctx context.Context, clientID string) (DeviceCode, error) {
	values := url.Values{}
	values.Set("client_id", clientID)

	var device DeviceCode
	if err := c.postForm(ctx, githubDeviceCodeURL, values, &device); err != nil {
		return DeviceCode{}, err
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.VerificationURI == "" {
		return DeviceCode{}, fmt.Errorf("GitHub returned an incomplete device code response")
	}
	if device.Interval <= 0 {
		device.Interval = 5
	}
	return device, nil
}

func (c *GitHubClient) PollDeviceToken(ctx context.Context, clientID string, device DeviceCode) (Credential, error) {
	interval := time.Duration(device.Interval) * time.Second
	deadline := time.Now().Add(time.Duration(device.ExpiresIn) * time.Second)

	for {
		if time.Now().After(deadline) {
			return Credential{}, fmt.Errorf("device code expired")
		}

		values := url.Values{}
		values.Set("client_id", clientID)
		values.Set("device_code", device.DeviceCode)
		values.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

		var token TokenResponse
		if err := c.postForm(ctx, githubAccessTokenURL, values, &token); err != nil {
			return Credential{}, err
		}

		switch token.Error {
		case "":
			return credentialFromToken(token, time.Now()), nil
		case "authorization_pending":
			time.Sleep(interval)
		case "slow_down":
			interval += 5 * time.Second
			time.Sleep(interval)
		default:
			if token.ErrorDescription != "" {
				return Credential{}, fmt.Errorf("%s: %s", token.Error, token.ErrorDescription)
			}
			return Credential{}, fmt.Errorf("%s", token.Error)
		}
	}
}

func (c *GitHubClient) RefreshCredential(ctx context.Context, clientID string, credential Credential) (Credential, error) {
	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", credential.RefreshToken)

	var token TokenResponse
	if err := c.postForm(ctx, githubAccessTokenURL, values, &token); err != nil {
		return Credential{}, err
	}
	if token.Error != "" {
		if token.ErrorDescription != "" {
			return Credential{}, fmt.Errorf("%s: %s", token.Error, token.ErrorDescription)
		}
		return Credential{}, fmt.Errorf("%s", token.Error)
	}
	return credentialFromToken(token, time.Now()), nil
}

func (c *GitHubClient) CurrentUser(ctx context.Context, accessToken string) (GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUserURL, nil)
	if err != nil {
		return GitHubUser{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GitHubUser{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return GitHubUser{}, fmt.Errorf("GitHub /user returned %s: %s", resp.Status, string(body))
	}
	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return GitHubUser{}, err
	}
	return user, nil
}

func (c *GitHubClient) postForm(ctx context.Context, endpoint string, values url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub returned %s: %s", resp.Status, string(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

func credentialFromToken(token TokenResponse, now time.Time) Credential {
	credential := Credential{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Scope:        token.Scope,
	}
	if token.ExpiresIn > 0 {
		credential.ExpiresAt = now.Add(time.Duration(token.ExpiresIn) * time.Second)
	}
	if token.RefreshTokenExpiresIn > 0 {
		credential.RefreshTokenExpiresAt = now.Add(time.Duration(token.RefreshTokenExpiresIn) * time.Second)
	}
	return credential
}

func FormatExpiry(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.UTC().Format(time.RFC3339)
}
