package config

import (
	"os"
	"testing"
)

func setEnv(t *testing.T, kvs map[string]string) {
	t.Helper()
	for k, v := range kvs {
		t.Setenv(k, v)
	}
}

func TestLoad_Success(t *testing.T) {
	setEnv(t, map[string]string{
		"NEO4J_URI":      "bolt://localhost:7687",
		"NEO4J_PASSWORD": "secret",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.Neo4jURI != "bolt://localhost:7687" {
		t.Errorf("Neo4jURI = %q", cfg.Neo4jURI)
	}
	if cfg.Neo4jUser != "neo4j" {
		t.Errorf("Neo4jUser = %q, want neo4j", cfg.Neo4jUser)
	}
	if cfg.Neo4jPassword != "secret" {
		t.Errorf("Neo4jPassword = %q", cfg.Neo4jPassword)
	}
	if cfg.Neo4jDatabase != "neo4j" {
		t.Errorf("Neo4jDatabase = %q, want neo4j", cfg.Neo4jDatabase)
	}
}

func TestLoad_CustomPort(t *testing.T) {
	setEnv(t, map[string]string{
		"PORT":           "9090",
		"NEO4J_URI":      "bolt://localhost:7687",
		"NEO4J_PASSWORD": "secret",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q, want :9090", cfg.HTTPAddr)
	}
}

func TestLoad_PortWithColon(t *testing.T) {
	setEnv(t, map[string]string{
		"PORT":           ":3000",
		"NEO4J_URI":      "bolt://localhost:7687",
		"NEO4J_PASSWORD": "secret",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":3000" {
		t.Errorf("HTTPAddr = %q, want :3000", cfg.HTTPAddr)
	}
}

func TestLoad_CustomNeo4jSettings(t *testing.T) {
	setEnv(t, map[string]string{
		"NEO4J_URI":      "bolt://db.example.com:7687",
		"NEO4J_USER":     "admin",
		"NEO4J_PASSWORD": "pass123",
		"NEO4J_DATABASE": "mydb",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Neo4jUser != "admin" {
		t.Errorf("Neo4jUser = %q, want admin", cfg.Neo4jUser)
	}
	if cfg.Neo4jDatabase != "mydb" {
		t.Errorf("Neo4jDatabase = %q, want mydb", cfg.Neo4jDatabase)
	}
}

func TestLoad_MissingURI(t *testing.T) {
	setEnv(t, map[string]string{
		"NEO4J_PASSWORD": "secret",
	})
	os.Unsetenv("NEO4J_URI")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing NEO4J_URI")
	}
}

func TestLoad_MissingPassword(t *testing.T) {
	setEnv(t, map[string]string{
		"NEO4J_URI": "bolt://localhost:7687",
	})
	os.Unsetenv("NEO4J_PASSWORD")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing NEO4J_PASSWORD")
	}
}

func TestGetenv_Fallback(t *testing.T) {
	os.Unsetenv("TEST_GETENV_KEY_XYZZY")
	got := getenv("TEST_GETENV_KEY_XYZZY", "default")
	if got != "default" {
		t.Errorf("getenv = %q, want default", got)
	}
}

func TestGetenv_Set(t *testing.T) {
	t.Setenv("TEST_GETENV_KEY_XYZZY", "value")
	got := getenv("TEST_GETENV_KEY_XYZZY", "default")
	if got != "value" {
		t.Errorf("getenv = %q, want value", got)
	}
}
