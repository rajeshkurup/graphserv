package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadJSON_Valid(t *testing.T) {
	body := bytes.NewBufferString(`{"label":"Application","properties":{"id":"app-1"}}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	var req ingestNodeReq
	if err := readJSON(r, &req); err != nil {
		t.Fatalf("readJSON error: %v", err)
	}
	if req.Label != "Application" {
		t.Errorf("Label = %q, want Application", req.Label)
	}
	id, ok := req.Properties["id"]
	if !ok {
		t.Fatal("missing properties.id")
	}
	if id != "app-1" {
		t.Errorf("id = %v, want app-1", id)
	}
}

func TestReadJSON_InvalidJSON(t *testing.T) {
	body := bytes.NewBufferString(`{not json}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	var req ingestNodeReq
	if err := readJSON(r, &req); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReadJSON_EmptyBody(t *testing.T) {
	body := bytes.NewBufferString(``)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	var req ingestNodeReq
	if err := readJSON(r, &req); err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestReadJSON_UseNumber(t *testing.T) {
	body := bytes.NewBufferString(`{"label":"Call","properties":{"id":"c1","latency":250}}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	var req ingestNodeReq
	if err := readJSON(r, &req); err != nil {
		t.Fatalf("readJSON error: %v", err)
	}
	lat := req.Properties["latency"]
	if _, ok := lat.(json.Number); !ok {
		t.Errorf("latency type = %T, want json.Number (UseNumber enabled)", lat)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
		t.Errorf("status = %q", got["status"])
	}
}

func TestWriteJSON_StatusCreated(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"id": "1"})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

func TestWriteErr(t *testing.T) {
	w := httptest.NewRecorder()
	writeErr(w, http.StatusBadRequest, "invalid input")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != "invalid input" {
		t.Errorf("error = %q", got["error"])
	}
}

func TestWriteStoreErr_Nil(t *testing.T) {
	w := httptest.NewRecorder()
	writeStoreErr(w, nil)
	if w.Code != http.StatusOK {
		t.Errorf("nil error should not write status, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("nil error should not write body, got %q", w.Body.String())
	}
}

func TestWriteStoreErr_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	writeStoreErr(w, fmt.Errorf("not found"))
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestWriteStoreErr_Generic(t *testing.T) {
	w := httptest.NewRecorder()
	writeStoreErr(w, fmt.Errorf("something bad"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestWriteStoreErr_ContextCanceled(t *testing.T) {
	w := httptest.NewRecorder()
	writeStoreErr(w, context.Canceled)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestWriteStoreErr_DeadlineExceeded(t *testing.T) {
	w := httptest.NewRecorder()
	writeStoreErr(w, context.DeadlineExceeded)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
}

func TestRoot_Handler(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	s.root(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["service"] != "graphserv" {
		t.Errorf("service = %v", got["service"])
	}
	endpoints, ok := got["endpoints"].([]any)
	if !ok || len(endpoints) == 0 {
		t.Error("expected non-empty endpoints list")
	}
}

func TestIngestNode_BadJSON(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/nodes", bytes.NewBufferString(`{bad`))
	s.ingestNode(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestIngestRel_BadJSON(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/relationships", bytes.NewBufferString(`{bad`))
	s.ingestRel(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestPostAnomaly_BadJSON(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/anomalies", bytes.NewBufferString(`{bad`))
	s.postAnomaly(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRootCause_BadJSON(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/analysis/root-cause", bytes.NewBufferString(`{bad`))
	s.rootCause(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRootCause_MissingFields(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"startLabel":"","startId":""}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/analysis/root-cause", body)
	s.rootCause(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestListRels_MissingParams(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/relationships", nil)
	s.listRels(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestImpact_MissingParams(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/analysis/impact", nil)
	s.impact(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestPatchRel_BadJSON(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/relationships", bytes.NewBufferString(`{bad`))
	s.patchRel(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestDeleteRel_BadJSON(t *testing.T) {
	s := &Server{Store: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/relationships", bytes.NewBufferString(`{bad`))
	s.deleteRel(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
