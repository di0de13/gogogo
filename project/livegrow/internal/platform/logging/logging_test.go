package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewWritesStructuredJSON(t *testing.T) {
	var out bytes.Buffer
	logger := New(slog.LevelInfo, &out)
	logger.Info("server started", "component", "test")
	text := out.String()
	if !strings.Contains(text, `"msg":"server started"`) || !strings.Contains(text, `"component":"test"`) {
		t.Fatalf("unexpected log output: %s", text)
	}
}
