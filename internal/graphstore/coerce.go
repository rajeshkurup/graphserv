/**
 * @file coerce.go
 * @brief Numeric coercion helpers for loosely-typed values returned from Neo4j records.
 * @auther rajeshkurup@live.com
 */
package graphstore

/**
 * @brief Converts numeric driver/JSON values to int (non-numeric types yield 0).
 * @param v any value (int, int32, int64, float64 supported).
 * @return Integer representation suitable for hop depth etc.
 */
func anyInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}
