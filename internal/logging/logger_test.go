package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerFormatsSeveritiesWithoutColor(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := New(buffer, false)

	logger.Info("branch: %s", "feature/test")
	logger.Warning("missing config")
	logger.Error("push failed")

	want := "grape: branch: feature/test\n" +
		"grape: warning: missing config\n" +
		"grape: push failed\n"
	if got := buffer.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestLoggerColorsOnlySeverityPrefix(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := New(buffer, true)

	logger.Warning("missing config")
	logger.Error("push failed")

	got := buffer.String()
	if !strings.Contains(got, "\033[33mgrape: warning:\033[0m missing config") {
		t.Fatalf("warning = %q, want colored prefix", got)
	}
	if !strings.Contains(got, "\033[31mgrape:\033[0m push failed") {
		t.Fatalf("error = %q, want colored prefix", got)
	}
	if strings.Contains(got, "\033[33mmissing config") || strings.Contains(got, "\033[31mpush failed") {
		t.Fatalf("message body was colored: %q", got)
	}
}
