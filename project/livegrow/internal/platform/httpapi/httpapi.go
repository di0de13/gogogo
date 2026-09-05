package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"livegrow/internal/platform/apperrors"
)

const RequestIDHeader = "X-Request-ID"

type contextKey string

const requestIDKey contextKey = "livegrow.request_id"

type Response struct {
	Data      any    `json:"data,omitempty"`
	Error     *Error `json:"error,omitempty"`
	RequestID string `json:"request_id"`
}

type Error struct {
	Code    apperrors.Code `json:"code"`
	Message string         `json:"message"`
}

func RequestID(r *http.Request) string {
	if value, ok := r.Context().Value(requestIDKey).(string); ok {
		return value
	}
	return ""
}

func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := normalizeRequestID(r.Header.Get(RequestIDHeader))
		ctx := contextWithRequestID(r, requestID)
		w.Header().Set(RequestIDHeader, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WithAccessLog(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		logger.Info("http request",
			"request_id", RequestID(r),
			"method", r.Method,
			"path", r.URL.Path,
			"status", writer.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func WriteJSON(w http.ResponseWriter, status int, requestID string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{Data: data, RequestID: requestID})
}

func WriteError(w http.ResponseWriter, requestID string, err error) {
	WriteAppError(w, requestID, err)
}

func WriteAppError(w http.ResponseWriter, requestID string, err error) {
	appErr := toAppError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Status)
	_ = json.NewEncoder(w).Encode(Response{
		Error:     &Error{Code: appErr.Code, Message: appErr.ClientMessage()},
		RequestID: requestID,
	})
}

func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return apperrors.New(apperrors.CodeInvalidRequest, "request body is required", http.StatusBadRequest)
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, "invalid JSON request body", http.StatusBadRequest, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return apperrors.New(apperrors.CodeInvalidRequest, "request body must contain one JSON value", http.StatusBadRequest)
	}
	return nil
}

type Validator interface{ Validate() error }

func Validate(v any) error {
	validator, ok := v.(Validator)
	if !ok {
		return nil
	}
	if err := validator.Validate(); err != nil {
		if _, ok := err.(*apperrors.AppError); ok {
			return err
		}
		return apperrors.Wrap(apperrors.CodeInvalidRequest, "request validation failed", http.StatusBadRequest, err)
	}
	return nil
}

func normalizeRequestID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 8 && len(value) <= 128 && isSafeRequestID(value) {
		return value
	}
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

func isSafeRequestID(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func contextWithRequestID(r *http.Request, requestID string) context.Context {
	return context.WithValue(r.Context(), requestIDKey, requestID)
}

func toAppError(err error) *apperrors.AppError {
	if err == nil {
		return apperrors.New(apperrors.CodeInternal, "internal server error", http.StatusInternalServerError)
	}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return apperrors.Wrap(apperrors.CodeInternal, "internal server error", http.StatusInternalServerError, err)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	return w.ResponseWriter.Write(body)
}
