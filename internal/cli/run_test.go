package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	oldVersion := Version
	Version = "test"
	t.Cleanup(func() { Version = oldVersion })

	var stdout bytes.Buffer
	err := Run(context.Background(), []string{"version"}, &stdout, &stdout)
	if err != nil {
		t.Fatalf("Run(version) error = %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "nssmr test" {
		t.Fatalf("stdout = %q, want %q", got, "nssmr test")
	}
}

func TestUsageOnUnknownCommand(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"bogus"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("Run(bogus) error = %v, want unknown command", err)
	}
}
