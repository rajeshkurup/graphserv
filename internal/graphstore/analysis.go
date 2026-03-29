/**
 * @file analysis.go
 * @brief Root-cause traversal and reverse-dependency (impact / blast radius) queries.
 * @auther rajeshkurup@live.com
 */
package graphstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

/**
 * @brief API payload for root-cause analysis: candidate paths and origins at minimum hop depth.
 */
type RootCauseResult struct {
	Origins    []map[string]any `json:"origins"`
	Candidates int              `json:"candidates"`
}

/**
 * @brief From a start node, walks downstream topology edges and finds anomalies by status; origins are min-depth matches.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param startLabel label of the entry node.
 * @param startID business id of the entry node.
 * @param maxDepth max path length (clamped 1–20 internally if needed).
 * @param anomalyStatus Anomaly.status filter (default active).
 * @param limit max candidate rows before filtering to minimum depth.
 * @return RootCauseResult with origins and candidate count, or error.
 */
func (s *Store) RootCauseFromStart(ctx context.Context, startLabel, startID string, maxDepth int, anomalyStatus string, limit int) (*RootCauseResult, error) {
	if err := validateLabel(startLabel); err != nil {
		return nil, err
	}
	if anomalyStatus == "" {
		anomalyStatus = "active"
	}
	depth := clampDepth(maxDepth, 1, 20)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	relPattern := strings.Join(topologyRels, "|")
	cypher := fmt.Sprintf(`
MATCH (start:%s {id: $startId})
MATCH p = (start)-[:%s*1..%d]->(n)
MATCH (anom:Anomaly)-[:DETECTED_ON]->(n)
WHERE anom.status = $status
WITH n, anom, min(length(p)) AS d
RETURN n AS node, anom AS anomaly, d AS depth
ORDER BY d ASC
LIMIT $limit
`, startLabel, relPattern, depth)

	sess := s.session(ctx, false)
	recs, err := readResult(ctx, sess, cypher, map[string]any{
		"startId": startID,
		"status":  anomalyStatus,
		"limit":   limit,
	})
	if err != nil {
		return nil, err
	}

	out := &RootCauseResult{Origins: []map[string]any{}, Candidates: len(recs)}
	minD := -1
	for _, rec := range recs {
		dv, _ := rec.Get("depth")
		d := anyInt(dv)
		if minD < 0 {
			minD = d
		}
		if d != minD {
			continue
		}
		entry := map[string]any{"depth": d}
		if v, ok := rec.Get("node"); ok {
			if n, ok := v.(neo4j.Node); ok {
				entry["node"] = nodeMap(n)
			}
		}
		if v, ok := rec.Get("anomaly"); ok {
			if n, ok := v.(neo4j.Node); ok {
				entry["anomaly"] = nodeMap(n)
			}
		}
		out.Origins = append(out.Origins, entry)
	}
	return out, nil
}

/**
 * @brief Response shape for impact analysis: dependents and aggregate counts.
 */
type ImpactMetrics struct {
	Start              map[string]any   `json:"start"`
	DependentCount     int              `json:"dependentCount"`
	ByRelationship     map[string]int   `json:"byRelationshipType"`
	ByLabel            map[string]int   `json:"byLabel"`
	Dependents         []map[string]any `json:"dependents"`
	UsedTransitiveRels bool             `json:"usedTransitive"`
}

/**
 * @brief Finds nodes that have an incoming edge to the target (reverse dependencies / blast radius inputs).
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param label target node label.
 * @param id target business id.
 * @param useTransitive if true, only DEPENDS_ON_TRANSITIVE incoming edges; else direct topology rel types.
 * @return ImpactMetrics with dependents and counts, or error.
 */
func (s *Store) ImpactAnalysis(ctx context.Context, label, id string, useTransitive bool) (*ImpactMetrics, error) {
	if err := validateLabel(label); err != nil {
		return nil, err
	}

	var relTypes []string
	if useTransitive {
		relTypes = []string{"DEPENDS_ON_TRANSITIVE"}
	} else {
		relTypes = []string{"CALLS", "USES_STORAGE", "CONNECTS_TO", "STORED_ON_NETWORK"}
	}

	typeList := "[" + strings.Join(sQuote(relTypes), ", ") + "]"

	cypher := fmt.Sprintf(`
MATCH (target) WHERE $label IN labels(target) AND target.id = $id
MATCH (target)<-[r]-(d)
WHERE type(r) IN %s
WITH d, collect(DISTINCT type(r)) AS relTypes
RETURN d AS dependent, relTypes AS relationshipTypes
`, typeList)

	sess := s.session(ctx, false)
	recs, err := readResult(ctx, sess, cypher, map[string]any{"label": label, "id": id})
	if err != nil {
		return nil, err
	}

	m := &ImpactMetrics{
		ByRelationship:     make(map[string]int),
		ByLabel:            make(map[string]int),
		Dependents:         []map[string]any{},
		UsedTransitiveRels: useTransitive,
	}

	startRec, err := s.GetNode(ctx, label, id)
	if err != nil {
		startRec = map[string]any{"labels": []string{label}, "id": id}
	}
	m.Start = startRec

	for _, rec := range recs {
		var rts []string
		if rv, ok := rec.Get("relationshipTypes"); ok {
			if sl, ok := rv.([]any); ok {
				for _, x := range sl {
					if s, ok := x.(string); ok {
						rts = append(rts, s)
					}
				}
			}
		}
		for _, rt := range rts {
			m.ByRelationship[rt]++
		}

		raw, _ := rec.Get("dependent")
		n, ok := raw.(neo4j.Node)
		if !ok {
			continue
		}
		nm := nodeMap(n)
		if len(n.Labels) > 0 {
			m.ByLabel[n.Labels[0]]++
		}
		entry := map[string]any{
			"relationshipTypes": rts,
			"node":              nm,
		}
		m.Dependents = append(m.Dependents, entry)
	}
	m.DependentCount = len(recs)
	return m, nil
}

/**
 * @brief Quotes strings for safe inclusion in a Cypher IN [...] literal built from allow-listed types.
 * @param ss slice of relationship type names.
 * @return Single-quoted strings suitable for IN [ 'A', 'B' ].
 */
func sQuote(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = "'" + strings.ReplaceAll(s, "'", "\\'") + "'"
	}
	return out
}
