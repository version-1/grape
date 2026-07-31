package color

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestDetectDisablesColorForRedirectedStreams(t *testing.T) {
	policy := Detect(&bytes.Buffer{}, &bytes.Buffer{}, func(string) string { return "" })
	if policy.Stdout || policy.Stderr {
		t.Fatalf("policy = %#v, want both disabled", policy)
	}
}

func TestDetectDisablesColorWhenNoColorIsSet(t *testing.T) {
	policy := Detect(os.Stdout, os.Stderr, func(key string) string {
		if key == "NO_COLOR" {
			return "1"
		}
		return ""
	})
	if policy.Stdout || policy.Stderr {
		t.Fatalf("policy = %#v, want both disabled", policy)
	}
}

func TestPolicyTracksStdoutAndStderrIndependently(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	policy := detect(stdout, stderr, func(string) string { return "" }, func(writer io.Writer) bool {
		return writer == stdout
	})
	if !policy.Stdout || policy.Stderr {
		t.Fatalf("policy = %#v", policy)
	}
}

func TestDetectDoesNotTreatNonTerminalCharacterDeviceAsTerminal(t *testing.T) {
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open null device: %v", err)
	}
	defer null.Close()

	policy := Detect(null, null, func(string) string { return "" })
	if policy.Stdout || policy.Stderr {
		t.Fatalf("policy = %#v, want non-terminal device disabled", policy)
	}
}
