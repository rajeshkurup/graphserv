/**
 * @file validate.go
 * @brief Allow-lists for node labels and relationship types; small validation helpers.
 * @auther rajeshkurup@live.com
 */
package graphstore

import (
	"fmt"
	"strings"
)

var (
	allowedLabels = map[string]struct{}{
		"Application":    {},
		"Storage":        {},
		"Network":        {},
		"IncidentTicket": {},
		"ChangeTicket":   {},
		"RCATicket":      {},
		"Action":         {},
		"Anomaly":        {},
		"Call":           {},
	}

	allowedRelTypes = map[string]struct{}{
		"CALLS":                 {},
		"USES_STORAGE":          {},
		"CONNECTS_TO":           {},
		"STORED_ON_NETWORK":     {},
		"IMPACTS":               {},
		"AFFECTS":               {},
		"ROOT_CAUSE_OF":         {},
		"HAS_ACTION":            {},
		"DETECTED_ON":           {},
		"TO":                    {},
		"DEPENDS_ON_TRANSITIVE": {},
	}

	topologyRels = []string{
		"CALLS",
		"USES_STORAGE",
		"CONNECTS_TO",
		"STORED_ON_NETWORK",
	}
)

/**
 * @brief Ensures label is an allowed Neo4j node label for safe dynamic Cypher fragments.
 * @param label user-supplied label string.
 * @return nil if allowed, or an error listing allowed labels.
 */
func validateLabel(label string) error {
	if _, ok := allowedLabels[label]; !ok {
		return fmt.Errorf("unknown label %q (allowed: %s)", label, strings.Join(sortedKeys(allowedLabels), ", "))
	}
	return nil
}

/**
 * @brief Ensures rt is an allowed relationship type for ingest/query operations.
 * @param rt user-supplied relationship type name.
 * @return nil if allowed, or an error.
 */
func validateRelType(rt string) error {
	if _, ok := allowedRelTypes[rt]; !ok {
		return fmt.Errorf("unknown relationship type %q", rt)
	}
	return nil
}

/**
 * @brief Returns map keys in a stable sorted order (bubble sort) for error messages.
 * @param m set-like map from string to empty struct.
 * @return Sorted slice of keys.
 */
func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// deterministic enough for error messages
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

/**
 * @brief Clamps d to the inclusive [min,max] range for traversal depth.
 * @param d requested depth (may be zero).
 * @param min minimum allowed depth.
 * @param max maximum allowed depth.
 * @return The clamped integer.
 */
func clampDepth(d, min, max int) int {
	if d < min {
		return min
	}
	if d > max {
		return max
	}
	return d
}
