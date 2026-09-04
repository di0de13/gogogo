package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestShutdownBeforeStartIsSafe(t *testing.T) {
	server := New("127.0.0.1:0", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), slog.Default())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
