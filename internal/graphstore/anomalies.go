/**
 * @file anomalies.go
 * @brief Anomaly upsert and DETECTED_ON attachment to topology nodes.
 * @auther rajeshkurup@live.com
 */
package graphstore

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

/**
 * @brief MERGEs an Anomaly node by id, SETs properties, and MERGEs DETECTED_ON to each target.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param anomalyProps must include id; other keys stored on the node.
 * @param targets nodes (label+id) the anomaly is detected on.
 * @param relProps optional properties merged onto each DETECTED_ON edge.
 * @return JSON-friendly anomaly node map, or error.
 */
func (s *Store) UpsertAnomalyOnNodes(ctx context.Context, anomalyProps map[string]any, targets []Endpoint, relProps map[string]any) (map[string]any, error) {
	if anomalyProps == nil {
		return nil, fmt.Errorf("anomaly properties required")
	}
	idVal, ok := anomalyProps["id"]
	if !ok {
		return nil, fmt.Errorf("anomaly id required")
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("at least one detectedOn target required")
	}
	if err := validateLabel("Anomaly"); err != nil {
		return nil, err
	}
	for _, t := range targets {
		if err := validateLabel(t.Label); err != nil {
			return nil, err
		}
	}
	if relProps == nil {
		relProps = map[string]any{}
	}

	aid := fmt.Sprint(idVal)
	props := make(map[string]any, len(anomalyProps))
	for k, v := range anomalyProps {
		props[k] = v
	}

	pairs := make([]map[string]any, 0, len(targets))
	for _, t := range targets {
		pairs = append(pairs, map[string]any{"label": t.Label, "id": t.ID})
	}

	cypher := `
MERGE (anom:Anomaly {id: $aid})
SET anom += $props
WITH anom
UNWIND $pairs AS pair
MATCH (n) WHERE pair.label IN labels(n) AND n.id = pair.id
MERGE (anom)-[r:DETECTED_ON]->(n)
SET r += $relProps
WITH DISTINCT anom
RETURN anom AS anomaly
`
	sess := s.session(ctx, true)
	recs, err := writeResult(ctx, sess, cypher, map[string]any{
		"aid":      aid,
		"props":    props,
		"pairs":    pairs,
		"relProps": relProps,
	})
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("no anomaly returned (check target node ids)")
	}
	raw, _ := recs[0].Get("anomaly")
	if n, ok := raw.(neo4j.Node); ok {
		return nodeMap(n), nil
	}
	return nil, fmt.Errorf("unexpected result")
}
