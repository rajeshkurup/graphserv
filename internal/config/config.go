/**
 * @file config.go
 * @brief Environment-based configuration for the HTTP server and Neo4j connection.
 * @auther rajeshkurup@live.com
 */
package config

import (
	"fmt"
	"os"
)

/**
 * @brief Runtime settings: listen address, Neo4j URI, credentials, and target database name.
 */
type Config struct {
	HTTPAddr      string
	Neo4jURI      string
	Neo4jUser     string
	Neo4jPassword string
	Neo4jDatabase string
}

/**
 * @brief Reads process environment variables and builds a Config, validating required Neo4j settings.
 * @return A populated *Config, or an error if NEO4J_URI or NEO4J_PASSWORD is missing.
 */
func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:      getenv("PORT", "8080"),
		Neo4jURI:      os.Getenv("NEO4J_URI"),
		Neo4jUser:     getenv("NEO4J_USER", "neo4j"),
		Neo4jPassword: os.Getenv("NEO4J_PASSWORD"),
		Neo4jDatabase: getenv("NEO4J_DATABASE", "neo4j"),
	}
	if cfg.HTTPAddr[0] != ':' {
		cfg.HTTPAddr = ":" + cfg.HTTPAddr
	}
	if cfg.Neo4jURI == "" {
		return nil, fmt.Errorf("NEO4J_URI is required (e.g. bolt://localhost:7687)")
	}
	if cfg.Neo4jPassword == "" {
		return nil, fmt.Errorf("NEO4J_PASSWORD is required")
	}
	return cfg, nil
}

/**
 * @brief Returns the environment value for key when set and non-empty; otherwise returns fallback.
 * @param key the environment variable name.
 * @param fallback default string when key is unset or empty.
 * @return The chosen string value.
 */
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
