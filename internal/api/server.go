/**
 * @file server.go
 * @brief HTTP REST API: routes, JSON request parsing, and responses delegating to graphstore.
 * @auther rajeshkurup@live.com
 */
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"graphserv/internal/graphstore"
)

// maxBody caps JSON request body size (1 MiB).
const maxBody = 1 << 20

/**
 * @brief Holds dependencies for REST handlers—primarily the Neo4j-backed graph store.
 */
type Server struct {
	Store *graphstore.Store
}

/**
 * @brief Registers all /api/v1 routes on mux (nodes, relationships, anomalies, analysis).
 * @param mux the HTTP serve mux receiving method+path patterns for Go 1.22+.
 * @return None.
 */
func (s *Server) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1", http.HandlerFunc(s.root))

	mux.Handle("POST /api/v1/nodes", http.HandlerFunc(s.ingestNode))
	mux.Handle("GET /api/v1/nodes/{label}", http.HandlerFunc(s.listNodes))
	mux.Handle("GET /api/v1/nodes/{label}/{id}", http.HandlerFunc(s.getNode))
	mux.Handle("PATCH /api/v1/nodes/{label}/{id}", http.HandlerFunc(s.patchNode))
	mux.Handle("DELETE /api/v1/nodes/{label}/{id}", http.HandlerFunc(s.deleteNode))

	mux.Handle("POST /api/v1/relationships", http.HandlerFunc(s.ingestRel))
	mux.Handle("GET /api/v1/relationships", http.HandlerFunc(s.listRels))
	mux.Handle("PATCH /api/v1/relationships", http.HandlerFunc(s.patchRel))
	mux.Handle("DELETE /api/v1/relationships", http.HandlerFunc(s.deleteRel))

	mux.Handle("POST /api/v1/anomalies", http.HandlerFunc(s.postAnomaly))

	mux.Handle("POST /api/v1/analysis/root-cause", http.HandlerFunc(s.rootCause))
	mux.Handle("GET /api/v1/analysis/impact", http.HandlerFunc(s.impact))
}

/**
 * @brief GET /api/v1 — returns service name and a list of available endpoint paths.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r incoming HTTP request.
 * @return None; writes JSON 200 to w.
 */
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "graphserv",
		"endpoints": []string{
			"POST /api/v1/nodes",
			"GET /api/v1/nodes/{label}",
			"GET /api/v1/nodes/{label}/{id}",
			"PATCH /api/v1/nodes/{label}/{id}",
			"DELETE /api/v1/nodes/{label}/{id}",
			"POST /api/v1/relationships",
			"GET /api/v1/relationships",
			"PATCH /api/v1/relationships",
			"DELETE /api/v1/relationships",
			"POST /api/v1/anomalies",
			"POST /api/v1/analysis/root-cause",
			"GET /api/v1/analysis/impact",
		},
	})
}

// ingestNodeReq is the JSON body for POST /api/v1/nodes.
type ingestNodeReq struct {
	Label      string         `json:"label"`
	Properties map[string]any `json:"properties"`
}

/**
 * @brief POST /api/v1/nodes — ingests (merge) a node with label and properties (must include id).
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with JSON body (ingestNodeReq).
 * @return None; responds 201 with node map or an error JSON status.
 */
func (s *Server) ingestNode(w http.ResponseWriter, r *http.Request) {
	var req ingestNodeReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.Store.IngestNode(r.Context(), req.Label, req.Properties)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

/**
 * @brief GET /api/v1/nodes/{label} — lists nodes for a label with optional ?limit=.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request; path label from route; query limit.
 * @return None; writes JSON { "nodes": [...] } or error.
 */
func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.Store.ListNodes(r.Context(), label, limit)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": items})
}

/**
 * @brief GET /api/v1/nodes/{label}/{id} — returns a single node by business id.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with path label and id.
 * @return None; writes JSON node or 404/error.
 */
func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	id := r.PathValue("id")
	n, err := s.Store.GetNode(r.Context(), label, id)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

/**
 * @brief PATCH /api/v1/nodes/{label}/{id} — merges JSON properties onto the node (id cannot be changed).
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with path label/id and JSON object body.
 * @return None; writes updated node JSON or error.
 */
func (s *Server) patchNode(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	id := r.PathValue("id")
	var patch map[string]any
	if err := readJSON(r, &patch); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	n, err := s.Store.UpdateNode(r.Context(), label, id, patch)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

/**
 * @brief DELETE /api/v1/nodes/{label}/{id} — detach-deletes the node.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with path label and id.
 * @return None; 204 on success or error JSON.
 */
func (s *Server) deleteNode(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	id := r.PathValue("id")
	if err := s.Store.DeleteNode(r.Context(), label, id); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ingestRelReq is the JSON body for POST /api/v1/relationships.
type ingestRelReq struct {
	Type       string              `json:"type"`
	From       graphstore.Endpoint `json:"from"`
	To         graphstore.Endpoint `json:"to"`
	Properties map[string]any      `json:"properties"`
}

/**
 * @brief POST /api/v1/relationships — merges a typed relationship between two endpoints.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with ingestRelReq JSON.
 * @return None; 201 with relationship map or error.
 */
func (s *Server) ingestRel(w http.ResponseWriter, r *http.Request) {
	var req ingestRelReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.Store.IngestRelationship(r.Context(), req.Type, req.From, req.To, req.Properties)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

/**
 * @brief GET /api/v1/relationships — lists relationships from a node; requires fromLabel, fromId, type; optional toLabel/toId, limit.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with query parameters.
 * @return None; JSON { "relationships": [...] } or 400/error.
 */
func (s *Server) listRels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := graphstore.Endpoint{Label: q.Get("fromLabel"), ID: q.Get("fromId")}
	relType := q.Get("type")
	if from.Label == "" || from.ID == "" || relType == "" {
		writeErr(w, http.StatusBadRequest, "fromLabel, fromId, and type query parameters required")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	var to *graphstore.Endpoint
	if tl, tid := q.Get("toLabel"), q.Get("toId"); tl != "" && tid != "" {
		to = &graphstore.Endpoint{Label: tl, ID: tid}
	}
	items, err := s.Store.ListRelationships(r.Context(), from, relType, to, limit)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"relationships": items})
}

// relPatchReq identifies a relationship and optional property patch for PATCH/DELETE.
type relPatchReq struct {
	Type  string              `json:"type"`
	From  graphstore.Endpoint `json:"from"`
	To    graphstore.Endpoint `json:"to"`
	Patch map[string]any      `json:"properties"`
}

/**
 * @brief PATCH /api/v1/relationships — merges properties onto an existing relationship.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with relPatchReq JSON.
 * @return None; updated relationship JSON or error.
 */
func (s *Server) patchRel(w http.ResponseWriter, r *http.Request) {
	var req relPatchReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.Store.UpdateRelationship(r.Context(), req.Type, req.From, req.To, req.Patch)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

/**
 * @brief DELETE /api/v1/relationships — deletes a relationship identified by type and endpoints.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with relPatchReq JSON (type, from, to).
 * @return None; 204 or error.
 */
func (s *Server) deleteRel(w http.ResponseWriter, r *http.Request) {
	var req relPatchReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.DeleteRelationship(r.Context(), req.Type, req.From, req.To); err != nil {
		writeStoreErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// anomalyReq is the JSON body for POST /api/v1/anomalies.
type anomalyReq struct {
	Anomaly                map[string]any        `json:"anomaly"`
	DetectedOn             []graphstore.Endpoint `json:"detectedOn"`
	RelationshipProperties map[string]any        `json:"relationshipProperties"`
}

/**
 * @brief POST /api/v1/anomalies — upserts an anomaly and DETECTED_ON edges to given topology nodes.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with anomalyReq JSON.
 * @return None; anomaly node JSON or error.
 */
func (s *Server) postAnomaly(w http.ResponseWriter, r *http.Request) {
	var req anomalyReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.Store.UpsertAnomalyOnNodes(r.Context(), req.Anomaly, req.DetectedOn, req.RelationshipProperties)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// rootCauseReq is the JSON body for POST /api/v1/analysis/root-cause.
type rootCauseReq struct {
	StartLabel    string `json:"startLabel"`
	StartID       string `json:"startId"`
	MaxDepth      int    `json:"maxDepth"`
	AnomalyStatus string `json:"anomalyStatus"`
	Limit         int    `json:"limit"`
}

/**
 * @brief POST /api/v1/analysis/root-cause — runs downstream topology traversal to nearest active anomalies.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with rootCauseReq JSON (startLabel, startId required).
 * @return None; RootCauseResult JSON or error.
 */
func (s *Server) rootCause(w http.ResponseWriter, r *http.Request) {
	var req rootCauseReq
	if err := readJSON(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.StartLabel == "" || req.StartID == "" {
		writeErr(w, http.StatusBadRequest, "startLabel and startId required")
		return
	}
	out, err := s.Store.RootCauseFromStart(r.Context(), req.StartLabel, req.StartID, req.MaxDepth, req.AnomalyStatus, req.Limit)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

/**
 * @brief GET /api/v1/analysis/impact — reverse-dependency (blast radius) metrics for a node.
 * @param s the API server (receiver).
 * @param w HTTP response writer.
 * @param r request with query label, id, optional useTransitive=true|1.
 * @return None; ImpactMetrics JSON or error.
 */
func (s *Server) impact(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	label, id := q.Get("label"), q.Get("id")
	if label == "" || id == "" {
		writeErr(w, http.StatusBadRequest, "label and id query parameters required")
		return
	}
	useTransitive := q.Get("useTransitive") == "true" || q.Get("useTransitive") == "1"
	out, err := s.Store.ImpactAnalysis(r.Context(), label, id, useTransitive)
	if err != nil {
		writeStoreErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

/**
 * @brief Decodes a JSON request body (capped by maxBody) into v and closes the body.
 * @param r HTTP request whose Body is read.
 * @param v destination for json.Unmarshal-compatible decode (pointer).
 * @return nil on success, or a decode/io error.
 */
func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	body := io.LimitReader(r.Body, maxBody)
	dec := json.NewDecoder(body)
	dec.UseNumber()
	return dec.Decode(v)
}

/**
 * @brief Writes JSON response with Content-Type application/json.
 * @param w HTTP response writer.
 * @param status HTTP status code.
 * @param v value to JSON-encode.
 * @return None.
 */
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

/**
 * @brief Writes a JSON object {"error": msg} with the given HTTP status.
 * @param w HTTP response writer.
 * @param status HTTP status code (e.g. 400).
 * @param msg error message string for clients.
 * @return None.
 */
func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

/**
 * @brief Maps store/driver errors to appropriate HTTP status and JSON error body.
 * @param w HTTP response writer.
 * @param err error from graphstore or Neo4j (nil returns immediately with no write).
 * @return None.
 */
func writeStoreErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		writeErr(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if neo4j.IsNeo4jError(err) {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	msg := err.Error()
	switch msg {
	case "not found":
		writeErr(w, http.StatusNotFound, msg)
	default:
		writeErr(w, http.StatusBadRequest, msg)
	}
}
