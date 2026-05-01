package impersonate

import "testing"

func TestResolveDefaultProfileFromConfig(t *testing.T) {
	cfg := Config{
		Profiles: map[string]Profile{
			"default": {ClientID: "Iv1.config"},
		},
	}

	profile, err := ResolveProfile(cfg, "default", "Iv1.shared")
	if err != nil {
		t.Fatal(err)
	}
	if profile.ClientID != "Iv1.config" {
		t.Fatalf("expected configured default client id, got %q", profile.ClientID)
	}
}

func TestResolveDefaultProfileFromSharedFallback(t *testing.T) {
	profile, err := ResolveProfile(Config{}, "default", "Iv1.shared")
	if err != nil {
		t.Fatal(err)
	}
	if profile.ClientID != "Iv1.shared" {
		t.Fatalf("expected shared fallback client id, got %q", profile.ClientID)
	}
}

func TestResolveDefaultProfileWithoutSharedFallback(t *testing.T) {
	_, err := ResolveProfile(Config{}, "default", "")
	if err == nil {
		t.Fatal("expected missing default profile error")
	}
}
