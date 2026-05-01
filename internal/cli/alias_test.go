package cli

import (
	"strings"
	"testing"
)

func TestAliasFunctionWithoutFixedProfile(t *testing.T) {
	got := aliasFunction("")
	if strings.Contains(got, "--profile") {
		t.Fatalf("alias without fixed profile should not include --profile:\n%s", got)
	}
	if !strings.Contains(got, `GH_IMPERSONATE`) {
		t.Fatalf("alias should check GH_IMPERSONATE:\n%s", got)
	}
	if !strings.Contains(got, `impersonate|auth|extension|help|version|--version`) {
		t.Fatalf("alias should include hard bypass commands:\n%s", got)
	}
}

func TestAliasFunctionWithFixedProfile(t *testing.T) {
	got := aliasFunction("work")
	if !strings.Contains(got, `gh impersonate --profile "work" exec -- "$@"`) {
		t.Fatalf("alias should pass fixed profile to exec:\n%s", got)
	}
}
