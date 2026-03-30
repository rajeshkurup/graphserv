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

// root godoc
// @Summary API root
// @Description Returns service name and list of available endpoints
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]any
// @Router /api/v1 [get]
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
	Label      string         `json:"label" example:"Application"`
	Properties map[string]any `json:"properties"`
}

// ingestNode godoc
// @Summary Ingest (merge) a node
// @Description Creates or merges a node with the given label and properties. Properties must include an "id" field.
// @Tags Nodes
// @Accept json
// @Produce json
// @Param body body ingestNodeReq true "Node label and properties"
// @Success 201 {object} map[string]any
// @Failure 400 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/nodes [post]
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

// listNodes godoc
// @Summary List nodes by label
// @Description Returns all nodes matching the given label with optional limit
// @Tags Nodes
// @Produce json
// @Param label path string true "Node label (e.g. Application, Storage, Network, IncidentTicket, ChangeTicket, RCATicket, Action, Anomaly, Call)"
// @Param limit query int false "Max results" default(100)
// @Success 200 {object} map[string]any
// @Failure 502 {object} errorResponse
// @Router /api/v1/nodes/{label} [get]
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

// getNode godoc
// @Summary Get a single node
// @Description Returns a node by its label and business id
// @Tags Nodes
// @Produce json
// @Param label path string true "Node label"
// @Param id path string true "Node business id"
// @Success 200 {object} map[string]any
// @Failure 404 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/nodes/{label}/{id} [get]
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

// patchNode godoc
// @Summary Update node properties
// @Description Merges the given JSON properties onto the node (id cannot be changed)
// @Tags Nodes
// @Accept json
// @Produce json
// @Param label path string true "Node label"
// @Param id path string true "Node business id"
// @Param body body map[string]any true "Properties to merge"
// @Success 200 {object} map[string]any
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/nodes/{label}/{id} [patch]
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

// deleteNode godoc
// @Summary Delete a node
// @Description Detach-deletes the node and all its relationships
// @Tags Nodes
// @Param label path string true "Node label"
// @Param id path string true "Node business id"
// @Success 204 "Node deleted"
// @Failure 404 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/nodes/{label}/{id} [delete]
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
	Type       string              `json:"type" example:"CALLS"`
	From       graphstore.Endpoint `json:"from"`
	To         graphstore.Endpoint `json:"to"`
	Properties map[string]any      `json:"properties"`
}

// ingestRel godoc
// @Summary Ingest (merge) a relationship
// @Description Creates or merges a typed relationship between two endpoint nodes
// @Tags Relationships
// @Accept json
// @Produce json
// @Param body body ingestRelReq true "Relationship type, endpoints, and properties"
// @Success 201 {object} map[string]any
// @Failure 400 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/relationships [post]
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

// listRels godoc
// @Summary List relationships from a node
// @Description Returns relationships of a given type from a source node, with optional target filter
// @Tags Relationships
// @Produce json
// @Param fromLabel query string true "Source node label"
// @Param fromId query string true "Source node id"
// @Param type query string true "Relationship type (e.g. CALLS, USES_STORAGE, CONNECTS_TO, STORED_ON_NETWORK, IMPACTS, AFFECTS, ROOT_CAUSE_OF, HAS_ACTION, TO, DEPENDS_ON_TRANSITIVE)"
// @Param toLabel query string false "Target node label"
// @Param toId query string false "Target node id"
// @Param limit query int false "Max results" default(100)
// @Success 200 {object} map[string]any
// @Failure 400 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/relationships [get]
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
	Type  string              `json:"type" example:"CALLS"`
	From  graphstore.Endpoint `json:"from"`
	To    graphstore.Endpoint `json:"to"`
	Patch map[string]any      `json:"properties"`
}

// patchRel godoc
// @Summary Update relationship properties
// @Description Merges properties onto an existing relationship identified by type and endpoints
// @Tags Relationships
// @Accept json
// @Produce json
// @Param body body relPatchReq true "Relationship identifier and properties to merge"
// @Success 200 {object} map[string]any
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/relationships [patch]
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

// deleteRelReq identifies a relationship for deletion.
type deleteRelReq struct {
	Type string              `json:"type" example:"CALLS"`
	From graphstore.Endpoint `json:"from"`
	To   graphstore.Endpoint `json:"to"`
}

// deleteRel godoc
// @Summary Delete a relationship
// @Description Deletes a relationship identified by type and endpoints
// @Tags Relationships
// @Accept json
// @Param body body deleteRelReq true "Relationship identifier"
// @Success 204 "Relationship deleted"
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/relationships [delete]
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

// postAnomaly godoc
// @Summary Upsert anomaly with DETECTED_ON edges
// @Description Creates or updates an anomaly node and links it to topology nodes via DETECTED_ON relationships
// @Tags Anomalies
// @Accept json
// @Produce json
// @Param body body anomalyReq true "Anomaly properties, target nodes, and relationship properties"
// @Success 200 {object} map[string]any
// @Failure 400 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/anomalies [post]
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
	StartLabel    string `json:"startLabel" example:"Application"`
	StartID       string `json:"startId" example:"app-123"`
	MaxDepth      int    `json:"maxDepth" example:"5"`
	AnomalyStatus string `json:"anomalyStatus" example:"active"`
	Limit         int    `json:"limit" example:"50"`
}

// rootCause godoc
// @Summary Root cause analysis
// @Description Traverses downstream topology edges from a start node to find active anomalies
// @Tags Analysis
// @Accept json
// @Produce json
// @Param body body rootCauseReq true "Start node and traversal parameters"
// @Success 200 {object} graphstore.RootCauseResult
// @Failure 400 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/analysis/root-cause [post]
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

// impact godoc
// @Summary Blast radius / impact analysis
// @Description Returns reverse-dependency (blast radius) metrics for a node — how many other nodes depend on it
// @Tags Analysis
// @Produce json
// @Param label query string true "Node label"
// @Param id query string true "Node business id"
// @Param useTransitive query string false "Use precomputed DEPENDS_ON_TRANSITIVE edges (true/false)" default(false)
// @Success 200 {object} graphstore.ImpactMetrics
// @Failure 400 {object} errorResponse
// @Failure 502 {object} errorResponse
// @Router /api/v1/analysis/impact [get]
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

// errorResponse is the standard error JSON shape for Swagger docs.
type errorResponse struct {
	Error string `json:"error" example:"bad request"`
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	body := io.LimitReader(r.Body, maxBody)
	dec := json.NewDecoder(body)
	dec.UseNumber()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

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
