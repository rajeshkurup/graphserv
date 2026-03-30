package graphstore

import (
	"context"
	"testing"
)

func TestIngestRelationship_InvalidRelType(t *testing.T) {
	s := &Store{}
	_, err := s.IngestRelationship(context.Background(), "INVALID",
		Endpoint{Label: "Application", ID: "1"},
		Endpoint{Label: "Application", ID: "2"},
		nil,
	)
	if err == nil {
		t.Fatal("expected error for invalid rel type")
	}
}

func TestIngestRelationship_InvalidFromLabel(t *testing.T) {
	s := &Store{}
	_, err := s.IngestRelationship(context.Background(), "CALLS",
		Endpoint{Label: "Bad", ID: "1"},
		Endpoint{Label: "Application", ID: "2"},
		nil,
	)
	if err == nil {
		t.Fatal("expected error for invalid from label")
	}
}

func TestIngestRelationship_InvalidToLabel(t *testing.T) {
	s := &Store{}
	_, err := s.IngestRelationship(context.Background(), "CALLS",
		Endpoint{Label: "Application", ID: "1"},
		Endpoint{Label: "Bad", ID: "2"},
		nil,
	)
	if err == nil {
		t.Fatal("expected error for invalid to label")
	}
}

func TestIngestRelationship_AllRelTypesValidated(t *testing.T) {
	s := &Store{}
	badTypes := []string{"", "calls", "UNKNOWN", "delete"}
	for _, rt := range badTypes {
		_, err := s.IngestRelationship(context.Background(), rt,
			Endpoint{Label: "Application", ID: "1"},
			Endpoint{Label: "Application", ID: "2"},
			nil,
		)
		if err == nil {
			t.Errorf("IngestRelationship(%q) = nil, want error", rt)
		}
	}
}

func TestListRelationships_InvalidFromLabel(t *testing.T) {
	s := &Store{}
	_, err := s.ListRelationships(context.Background(),
		Endpoint{Label: "Bad", ID: "1"}, "CALLS", nil, 10,
	)
	if err == nil {
		t.Fatal("expected error for invalid from label")
	}
}

func TestListRelationships_InvalidRelType(t *testing.T) {
	s := &Store{}
	_, err := s.ListRelationships(context.Background(),
		Endpoint{Label: "Application", ID: "1"}, "INVALID", nil, 10,
	)
	if err == nil {
		t.Fatal("expected error for invalid rel type")
	}
}

func TestListRelationships_InvalidToLabel(t *testing.T) {
	s := &Store{}
	to := &Endpoint{Label: "Bad", ID: "2"}
	_, err := s.ListRelationships(context.Background(),
		Endpoint{Label: "Application", ID: "1"}, "CALLS", to, 10,
	)
	if err == nil {
		t.Fatal("expected error for invalid to label")
	}
}

func TestUpdateRelationship_InvalidRelType(t *testing.T) {
	s := &Store{}
	_, err := s.UpdateRelationship(context.Background(), "INVALID",
		Endpoint{Label: "Application", ID: "1"},
		Endpoint{Label: "Application", ID: "2"},
		map[string]any{"key": "val"},
	)
	if err == nil {
		t.Fatal("expected error for invalid rel type")
	}
}

func TestUpdateRelationship_InvalidFromLabel(t *testing.T) {
	s := &Store{}
	_, err := s.UpdateRelationship(context.Background(), "CALLS",
		Endpoint{Label: "Bad", ID: "1"},
		Endpoint{Label: "Application", ID: "2"},
		map[string]any{"key": "val"},
	)
	if err == nil {
		t.Fatal("expected error for invalid from label")
	}
}

func TestUpdateRelationship_InvalidToLabel(t *testing.T) {
	s := &Store{}
	_, err := s.UpdateRelationship(context.Background(), "CALLS",
		Endpoint{Label: "Application", ID: "1"},
		Endpoint{Label: "Bad", ID: "2"},
		map[string]any{"key": "val"},
	)
	if err == nil {
		t.Fatal("expected error for invalid to label")
	}
}

func TestDeleteRelationship_InvalidRelType(t *testing.T) {
	s := &Store{}
	err := s.DeleteRelationship(context.Background(), "INVALID",
		Endpoint{Label: "Application", ID: "1"},
		Endpoint{Label: "Application", ID: "2"},
	)
	if err == nil {
		t.Fatal("expected error for invalid rel type")
	}
}
