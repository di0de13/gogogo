package apperrors

import (
	"errors"
	"testing"
)

func TestWrapPreservesCauseAndSafeMessage(t *testing.T) {
	cause := errors.New("database password=secret")
	err := Wrap(CodeUnavailable, "storage temporarily unavailable", 503, cause)
	if !errors.Is(err, cause) {
		t.Fatal("wrapped cause was not preserved")
	}
	if err.ClientMessage() != "storage temporarily unavailable" {
		t.Fatalf("unexpected client message: %q", err.ClientMessage())
	}
}
