package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"livegrow/internal/platform/apperrors"
)

type testRequest struct {
	Name string `json:"name"`
}

func (r testRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name must not be empty")
	}
	return nil
}

func TestWithRequestIDPreservesSafeIDAndGeneratesMissingID(t *testing.T) {
	handler := WithRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestID(r) == "" {
			t.Fatal("request ID missing from context")
		}
		WriteJSON(w, http.StatusOK, RequestID(r), map[string]string{"ok": "true"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(RequestIDHeader, "client-123")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Header().Get(RequestIDHeader) != "client-123" {
		t.Fatalf("request ID = %q", res.Header().Get(RequestIDHeader))
	}

	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	generated := res.Header().Get(RequestIDHeader)
	if len(generated) != 32 {
		t.Fatalf("generated request ID length = %d, value=%q", len(generated), generated)
	}
}

func TestWithRequestIDRejectsUnsafeID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(RequestIDHeader, "bad value")
	res := httptest.NewRecorder()
	WithRequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestID(r) == "bad value" {
			t.Fatal("unsafe request ID was preserved")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(res, req)
	if res.Header().Get(RequestIDHeader) == "bad value" {
		t.Fatal("unsafe request ID was returned")
	}
}

func TestWriteAppErrorHidesInternalCause(t *testing.T) {
	res := httptest.NewRecorder()
	WriteError(res, "req-123456", apperrors.Wrap(apperrors.CodeUnavailable, "storage temporarily unavailable", http.StatusServiceUnavailable, errors.New("password=secret")))
	if res.Code != http.StatusServiceUnavailable || strings.Contains(res.Body.String(), "password=secret") {
		t.Fatalf("unexpected error response: code=%d body=%s", res.Code, res.Body.String())
	}
	var body Response
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error == nil || body.Error.Code != apperrors.CodeUnavailable || body.RequestID != "req-123456" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestDecodeJSONAndValidate(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/test", bytes.NewBufferString(`{"name":"alice"}`))
	var payload testRequest
	if err := DecodeJSON(req, &payload); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if err := Validate(payload); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/test", bytes.NewBufferString(`{"unknown":1}`))
	if err := DecodeJSON(req, &payload); err == nil {
		t.Fatal("DecodeJSON() expected unknown-field error")
	}

	if err := Validate(testRequest{}); err == nil {
		t.Fatal("Validate() expected error")
	}
}

func TestWithAccessLogRecordsStatus(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := WithAccessLog(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest(http.MethodPost, "/v1/test", nil)
	res := httptest.NewRecorder()
	WithRequestID(handler).ServeHTTP(res, req)
	if res.Code != http.StatusCreated || !strings.Contains(logs.String(), `"status":201`) {
		t.Fatalf("unexpected result: code=%d logs=%s", res.Code, logs.String())
	}
}
