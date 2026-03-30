package graphstore

import (
	"context"
	"testing"
)

func TestIngestNode_InvalidLabel(t *testing.T) {
	s := &Store{}
	_, err := s.IngestNode(context.Background(), "BadLabel", map[string]any{"id": "1"})
	if err == nil {
		t.Fatal("expected error for invalid label")
	}
}

func TestIngestNode_NilProperties(t *testing.T) {
	s := &Store{}
	_, err := s.IngestNode(context.Background(), "Application", nil)
	if err == nil {
		t.Fatal("expected error for nil properties")
	}
}

func TestIngestNode_MissingID(t *testing.T) {
	s := &Store{}
	_, err := s.IngestNode(context.Background(), "Application", map[string]any{"name": "test"})
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestIngestNode_AllLabelsValidated(t *testing.T) {
	s := &Store{}
	badLabels := []string{"", "unknown", "application", "STORAGE", "drop;--"}
	for _, label := range badLabels {
		_, err := s.IngestNode(context.Background(), label, map[string]any{"id": "1"})
		if err == nil {
			t.Errorf("IngestNode(%q) = nil, want error", label)
		}
	}
}

func TestGetNode_InvalidLabel(t *testing.T) {
	s := &Store{}
	_, err := s.GetNode(context.Background(), "BadLabel", "id-1")
	if err == nil {
		t.Fatal("expected error for invalid label")
	}
}

func TestGetNode_AllLabelsValidated(t *testing.T) {
	s := &Store{}
	for _, label := range []string{"", "foo", "network"} {
		_, err := s.GetNode(context.Background(), label, "id-1")
		if err == nil {
			t.Errorf("GetNode(%q) = nil, want error", label)
		}
	}
}

func TestListNodes_InvalidLabel(t *testing.T) {
	s := &Store{}
	_, err := s.ListNodes(context.Background(), "BadLabel", 10)
	if err == nil {
		t.Fatal("expected error for invalid label")
	}
}

func TestUpdateNode_InvalidLabel(t *testing.T) {
	s := &Store{}
	_, err := s.UpdateNode(context.Background(), "BadLabel", "id-1", map[string]any{"name": "x"})
	if err == nil {
		t.Fatal("expected error for invalid label")
	}
}

func TestUpdateNode_NilPatch(t *testing.T) {
	s := &Store{}
	_, err := s.UpdateNode(context.Background(), "Application", "id-1", nil)
	if err == nil {
		t.Fatal("expected error for nil patch")
	}
}

func TestDeleteNode_InvalidLabel(t *testing.T) {
	s := &Store{}
	err := s.DeleteNode(context.Background(), "BadLabel", "id-1")
	if err == nil {
		t.Fatal("expected error for invalid label")
	}
}
