package config

import (
	"testing"
	"time"
)

func TestParseAffinityMask(t *testing.T) {
	t.Parallel()

	mask, err := ParseAffinityMask("0-2,4,6-7")
	if err != nil {
		t.Fatalf("ParseAffinityMask() error = %v", err)
	}

	if got, want := FormatAffinityMask(mask), "0-2,4,6,7"; got != want {
		t.Fatalf("FormatAffinityMask() = %q, want %q", got, want)
	}
}

func TestParseAffinityMaskRejectsInvalidCPU(t *testing.T) {
	t.Parallel()

	if _, err := ParseAffinityMask("64"); err == nil {
		t.Fatal("ParseAffinityMask() succeeded, want error")
	}
}

func TestPriorityClassWindowsValueRoundTrip(t *testing.T) {
	t.Parallel()

	priority, ok := PriorityClassFromWindowsValue(PriorityAboveNormal.WindowsValue())
	if !ok {
		t.Fatal("PriorityClassFromWindowsValue() = !ok, want ok")
	}
	if priority != PriorityAboveNormal {
		t.Fatalf("PriorityClassFromWindowsValue() = %q, want %q", priority, PriorityAboveNormal)
	}
}

func TestStopWaitHint(t *testing.T) {
	t.Parallel()

	service := Default("svc")
	service.StopMethodSkip = StopMethodWindow | StopMethodTerminate
	service.StopConsoleDelay = 2 * time.Second
	service.StopThreadsDelay = 3 * time.Second

	if got, want := service.StopWaitHint(), 5*time.Second; got != want {
		t.Fatalf("StopWaitHint() = %v, want %v", got, want)
	}
}
