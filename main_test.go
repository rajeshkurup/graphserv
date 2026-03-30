package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthHandler(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["status"] != "ok" {
		t.Errorf("status = %q, want ok", got["status"])
	}
}

func TestWriteJSON_Main(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]int{"count": 5})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	var got map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["count"] != 5 {
		t.Errorf("count = %d, want 5", got["count"])
	}
}

func TestLoggingMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := loggingMiddleware(inner)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestLoggingMiddleware_PassesThrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "passed")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("body"))
	})
	handler := loggingMiddleware(inner)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/foo", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", w.Code)
	}
	if w.Header().Get("X-Test") != "passed" {
		t.Error("middleware did not pass through headers")
	}
	if w.Body.String() != "body" {
		t.Errorf("body = %q", w.Body.String())
	}
}
