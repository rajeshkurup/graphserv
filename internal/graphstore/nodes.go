/**
 * @file nodes.go
 * @brief Store methods for node CRUD: merge ingest, read, list, patch, detach-delete.
 * @auther rajeshkurup@live.com
 */
package graphstore

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

/**
 * @brief MERGEs a node by label and id then SETs properties; id must exist in properties.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param label validated Neo4j label.
 * @param properties node properties including required id.
 * @return JSON-friendly map of the node, or error.
 */
func (s *Store) IngestNode(ctx context.Context, label string, properties map[string]any) (map[string]any, error) {
	if err := validateLabel(label); err != nil {
		return nil, err
	}
	if properties == nil {
		return nil, fmt.Errorf("properties required")
	}
	idVal, ok := properties["id"]
	if !ok {
		return nil, fmt.Errorf("properties.id is required")
	}

	props := make(map[string]any, len(properties))
	for k, v := range properties {
		props[k] = v
	}
	idStr := fmt.Sprint(idVal)

	cypher := fmt.Sprintf("MERGE (n:%s {id: $id}) SET n += $props RETURN n AS node", label)
	sess := s.session(ctx, true)
	recs, err := writeResult(ctx, sess, cypher, map[string]any{"id": idStr, "props": props})
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("no record returned")
	}
	raw, _ := recs[0].Get("node")
	switch n := raw.(type) {
	case neo4j.Node:
		return nodeMap(n), nil
	default:
		return map[string]any{"node": ToJSONValue(raw)}, nil
	}
}

/**
 * @brief Loads a single node by business id and label.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param label validated label.
 * @param id business key on the node.
 * @return Map of properties and labels, or "not found" error.
 */
func (s *Store) GetNode(ctx context.Context, label, id string) (map[string]any, error) {
	if err := validateLabel(label); err != nil {
		return nil, err
	}
	cypher := fmt.Sprintf("MATCH (n:%s {id: $id}) RETURN n AS node LIMIT 1", label)
	sess := s.session(ctx, false)
	recs, err := readResult(ctx, sess, cypher, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("not found")
	}
	raw, _ := recs[0].Get("node")
	if n, ok := raw.(neo4j.Node); ok {
		return nodeMap(n), nil
	}
	return nil, fmt.Errorf("unexpected result")
}

/**
 * @brief Returns up to limit nodes for a label (default and max bounds applied).
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param label validated label.
 * @param limit desired max rows (0 triggers defaults).
 * @return Slice of JSON-friendly node maps.
 */
func (s *Store) ListNodes(ctx context.Context, label string, limit int) ([]map[string]any, error) {
	if err := validateLabel(label); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	cypher := fmt.Sprintf("MATCH (n:%s) RETURN n AS node LIMIT $limit", label)
	sess := s.session(ctx, false)
	recs, err := readResult(ctx, sess, cypher, map[string]any{"limit": limit})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(recs))
	for _, rec := range recs {
		raw, _ := rec.Get("node")
		if n, ok := raw.(neo4j.Node); ok {
			out = append(out, nodeMap(n))
		}
	}
	return out, nil
}

/**
 * @brief MATCH + SET merges patch onto an existing node; does not clear missing keys.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param label validated label.
 * @param id business id.
 * @param patch property map (id key ignored if present).
 * @return Updated node map, or "not found" error.
 */
func (s *Store) UpdateNode(ctx context.Context, label, id string, patch map[string]any) (map[string]any, error) {
	if err := validateLabel(label); err != nil {
		return nil, err
	}
	if patch == nil {
		return nil, fmt.Errorf("patch required")
	}
	props := make(map[string]any, len(patch))
	for k, v := range patch {
		if k == "id" {
			continue
		}
		props[k] = v
	}
	cypher := fmt.Sprintf("MATCH (n:%s {id: $id}) SET n += $props RETURN n AS node", label)
	sess := s.session(ctx, true)
	recs, err := writeResult(ctx, sess, cypher, map[string]any{"id": id, "props": props})
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("not found")
	}
	raw, _ := recs[0].Get("node")
	if n, ok := raw.(neo4j.Node); ok {
		return nodeMap(n), nil
	}
	return nil, fmt.Errorf("unexpected result")
}

/**
 * @brief DETACH DELETE removes the node and all attached relationships.
 * @param s the graph store (receiver).
 * @param ctx request context.
 * @param label validated label.
 * @param id business id.
 * @return Error if Cypher fails; nil on success.
 */
func (s *Store) DeleteNode(ctx context.Context, label, id string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	cypher := fmt.Sprintf("MATCH (n:%s {id: $id}) DETACH DELETE n", label)
	sess := s.session(ctx, true)
	recs, err := writeResult(ctx, sess, cypher, map[string]any{"id": id})
	if err != nil {
		return err
	}
	_ = recs
	return nil
}
