package nomad

import "testing"

func TestPickSSHTarget(t *testing.T) {
	if got := pickSSHTarget("node-a", "10.0.0.1"); got != "node-a" {
		t.Fatalf("prefer name: got %q", got)
	}
	if got := pickSSHTarget("", "10.0.0.1"); got != "10.0.0.1" {
		t.Fatalf("fallback address: got %q", got)
	}
	if got := pickSSHTarget("  ", ""); got != "" {
		t.Fatalf("empty: got %q", got)
	}
}

func TestHostFromHTTPAddr(t *testing.T) {
	if got := hostFromHTTPAddr("10.0.0.5:4646"); got != "10.0.0.5" {
		t.Fatalf("got %q", got)
	}
	if got := hostFromHTTPAddr("10.0.0.5"); got != "10.0.0.5" {
		t.Fatalf("got %q", got)
	}
}

func TestRunSSHCommandRejectsPlaceholder(t *testing.T) {
	_, err := RunSSHCommand("localhost", "PLACEHOLDER: nope")
	if err == nil {
		t.Fatal("expected error for PLACEHOLDER command")
	}
}
