/**
 * @file relationships.go
 * @brief Store methods for relationship ingest, query, patch, and delete between endpoints.
 * @auther rajeshkurup@live.com
 */
package graphstore

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

/**
 * @brief Identifies an endpoint node by Neo4j label and business id for relationship APIs.
 */
type Endpoint struct {
	Label string `json:"label"`
	ID    string `json:"id"`
}

/**
 * @brief MERGEs a typed directed edge from→to with optional relationship properties.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param relType validated relationship type.
 * @param from source endpoint (match by label+id).
 * @param to target endpoint.
 * @param properties relationship property map (may be empty).
 * @return JSON-friendly relationship map, or error if endpoints missing.
 */
func (s *Store) IngestRelationship(ctx context.Context, relType string, from, to Endpoint, properties map[string]any) (map[string]any, error) {
	if err := validateRelType(relType); err != nil {
		return nil, err
	}
	if err := validateLabel(from.Label); err != nil {
		return nil, fmt.Errorf("from: %w", err)
	}
	if err := validateLabel(to.Label); err != nil {
		return nil, fmt.Errorf("to: %w", err)
	}
	if properties == nil {
		properties = map[string]any{}
	}
	cypher := fmt.Sprintf(
		"MATCH (a:%s {id: $fromId}), (b:%s {id: $toId}) MERGE (a)-[r:%s]->(b) SET r += $props RETURN r AS rel",
		from.Label, to.Label, relType,
	)
	sess := s.session(ctx, true)
	recs, err := writeResult(ctx, sess, cypher, map[string]any{
		"fromId": from.ID,
		"toId":   to.ID,
		"props":  properties,
	})
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("not found or merge produced no record")
	}
	raw, _ := recs[0].Get("rel")
	if r, ok := raw.(neo4j.Relationship); ok {
		return relMap(r), nil
	}
	return nil, fmt.Errorf("unexpected result")
}

/**
 * @brief Lists outgoing relationships from a node by type; optional exact target filter.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param from source endpoint (required).
 * @param relType relationship type (required).
 * @param to if non-nil, constrains the target node label+id.
 * @param limit max rows (bounded internally).
 * @return Slice of maps with from, to, relationship keys.
 */
func (s *Store) ListRelationships(ctx context.Context, from Endpoint, relType string, to *Endpoint, limit int) ([]map[string]any, error) {
	if err := validateLabel(from.Label); err != nil {
		return nil, err
	}
	if err := validateRelType(relType); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	var cypher string
	params := map[string]any{"fromId": from.ID, "limit": limit}

	if to != nil {
		if err := validateLabel(to.Label); err != nil {
			return nil, err
		}
		cypher = fmt.Sprintf(
			"MATCH (a:%s {id: $fromId})-[r:%s]->(b:%s {id: $toId}) RETURN a AS fromNode, b AS toNode, r AS rel LIMIT $limit",
			from.Label, relType, to.Label,
		)
		params["toId"] = to.ID
	} else {
		cypher = fmt.Sprintf(
			"MATCH (a:%s {id: $fromId})-[r:%s]->(b) RETURN a AS fromNode, b AS toNode, r AS rel LIMIT $limit",
			from.Label, relType,
		)
	}

	sess := s.session(ctx, false)
	recs, err := readResult(ctx, sess, cypher, params)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(recs))
	for _, rec := range recs {
		item := map[string]any{}
		if v, ok := rec.Get("fromNode"); ok {
			if n, ok := v.(neo4j.Node); ok {
				item["from"] = nodeMap(n)
			}
		}
		if v, ok := rec.Get("toNode"); ok {
			if n, ok := v.(neo4j.Node); ok {
				item["to"] = nodeMap(n)
			}
		}
		if v, ok := rec.Get("rel"); ok {
			if r, ok := v.(neo4j.Relationship); ok {
				item["relationship"] = relMap(r)
			}
		}
		out = append(out, item)
	}
	return out, nil
}

/**
 * @brief SETs merged properties on an existing relationship identified by endpoints and type.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param relType relationship type.
 * @param from source endpoint.
 * @param to target endpoint.
 * @param patch properties to merge onto the relationship.
 * @return Updated relationship map, or "not found".
 */
func (s *Store) UpdateRelationship(ctx context.Context, relType string, from, to Endpoint, patch map[string]any) (map[string]any, error) {
	if err := validateRelType(relType); err != nil {
		return nil, err
	}
	if err := validateLabel(from.Label); err != nil {
		return nil, err
	}
	if err := validateLabel(to.Label); err != nil {
		return nil, err
	}
	if patch == nil {
		patch = map[string]any{}
	}
	cypher := fmt.Sprintf(
		"MATCH (a:%s {id: $fromId})-[r:%s]->(b:%s {id: $toId}) SET r += $props RETURN r AS rel",
		from.Label, relType, to.Label,
	)
	sess := s.session(ctx, true)
	recs, err := writeResult(ctx, sess, cypher, map[string]any{
		"fromId": from.ID,
		"toId":   to.ID,
		"props":  patch,
	})
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("not found")
	}
	raw, _ := recs[0].Get("rel")
	if r, ok := raw.(neo4j.Relationship); ok {
		return relMap(r), nil
	}
	return nil, fmt.Errorf("unexpected result")
}

/**
 * @brief Deletes the single relationship matching type and endpoints.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param relType relationship type.
 * @param from source endpoint.
 * @param to target endpoint.
 * @return Error if Cypher fails; nil when DELETE completes.
 */
func (s *Store) DeleteRelationship(ctx context.Context, relType string, from, to Endpoint) error {
	if err := validateRelType(relType); err != nil {
		return err
	}
	cypher := fmt.Sprintf(
		"MATCH (a:%s {id: $fromId})-[r:%s]->(b:%s {id: $toId}) DELETE r",
		from.Label, relType, to.Label,
	)
	sess := s.session(ctx, true)
	_, err := writeResult(ctx, sess, cypher, map[string]any{"fromId": from.ID, "toId": to.ID})
	return err
}
