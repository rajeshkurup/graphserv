/**
 * @file convert.go
 * @brief Converts Neo4j driver types (nodes, rels, paths, temporal, spatial) to JSON-friendly maps.
 * @auther rajeshkurup@live.com
 */
package graphstore

import (
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

/**
 * @brief Recursively maps Neo4j-specific values to strings/maps or passes through primitives.
 * @param v any value from a Record, Node props, etc.
 * @return A value safe to marshal as JSON in typical cases.
 */
func ToJSONValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case neo4j.Node:
		return nodeMap(x)
	case neo4j.Relationship:
		return relMap(x)
	case neo4j.Path:
		return pathMap(x)
	case neo4j.Point2D:
		return map[string]any{"srid": x.SpatialRefId, "x": x.X, "y": x.Y}
	case neo4j.Point3D:
		return map[string]any{"srid": x.SpatialRefId, "x": x.X, "y": x.Y, "z": x.Z}
	case neo4j.Duration:
		return x.String()
	case neo4j.Date:
		return x.String()
	case neo4j.LocalTime:
		return x.String()
	case neo4j.LocalDateTime:
		return x.String()
	case neo4j.OffsetTime:
		return x.String()
	default:
		return x
	}
}

/**
 * @brief Converts a Neo4j Node to a flat map including labels and property keys.
 * @param n driver node value.
 * @return Map with "labels" and all property keys.
 */
func nodeMap(n neo4j.Node) map[string]any {
	m := map[string]any{
		"labels": n.Labels,
	}
	for k, v := range n.Props {
		m[k] = ToJSONValue(v)
	}
	return m
}

/**
 * @brief Converts a Neo4j Relationship to a map including type, ids, and properties.
 * @param r driver relationship value.
 * @return Map suitable for JSON encoding.
 */
func relMap(r neo4j.Relationship) map[string]any {
	m := map[string]any{
		"type":      r.Type,
		"elementId": r.ElementId,
		"startId":   r.StartId,
		"endId":     r.EndId,
		"startElId": r.StartElementId,
		"endElId":   r.EndElementId,
	}
	for k, v := range r.Props {
		m[k] = ToJSONValue(v)
	}
	return m
}

/**
 * @brief Converts a Path to nested node and relationship maps.
 * @param p Neo4j path value.
 * @return Map with "nodes" and "relationships" arrays.
 */
func pathMap(p neo4j.Path) map[string]any {
	nodes := make([]map[string]any, 0, len(p.Nodes))
	for _, n := range p.Nodes {
		nodes = append(nodes, nodeMap(n))
	}
	rels := make([]map[string]any, 0, len(p.Relationships))
	for _, r := range p.Relationships {
		rels = append(rels, relMap(r))
	}
	return map[string]any{"nodes": nodes, "relationships": rels}
}
