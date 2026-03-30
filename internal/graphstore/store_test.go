package graphstore

import (
	"context"
	"testing"
)

func TestClose_NilStore(t *testing.T) {
	var s *Store
	if err := s.Close(context.Background()); err != nil {
		t.Errorf("Close(nil store) = %v, want nil", err)
	}
}

func TestClose_NilDriver(t *testing.T) {
	s := &Store{driver: nil, database: "neo4j"}
	if err := s.Close(context.Background()); err != nil {
		t.Errorf("Close(nil driver) = %v, want nil", err)
	}
}
