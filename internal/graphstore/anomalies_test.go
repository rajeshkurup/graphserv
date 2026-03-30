package graphstore

import (
	"context"
	"testing"
)

func TestUpsertAnomalyOnNodes_NilProps(t *testing.T) {
	s := &Store{}
	_, err := s.UpsertAnomalyOnNodes(context.Background(), nil,
		[]Endpoint{{Label: "Application", ID: "1"}}, nil,
	)
	if err == nil {
		t.Fatal("expected error for nil anomaly properties")
	}
}

func TestUpsertAnomalyOnNodes_MissingID(t *testing.T) {
	s := &Store{}
	_, err := s.UpsertAnomalyOnNodes(context.Background(),
		map[string]any{"type": "latency_spike"},
		[]Endpoint{{Label: "Application", ID: "1"}}, nil,
	)
	if err == nil {
		t.Fatal("expected error for missing anomaly id")
	}
}

func TestUpsertAnomalyOnNodes_EmptyTargets(t *testing.T) {
	s := &Store{}
	_, err := s.UpsertAnomalyOnNodes(context.Background(),
		map[string]any{"id": "ANOM-1"},
		[]Endpoint{}, nil,
	)
	if err == nil {
		t.Fatal("expected error for empty targets")
	}
}

func TestUpsertAnomalyOnNodes_NilTargets(t *testing.T) {
	s := &Store{}
	_, err := s.UpsertAnomalyOnNodes(context.Background(),
		map[string]any{"id": "ANOM-1"},
		nil, nil,
	)
	if err == nil {
		t.Fatal("expected error for nil targets")
	}
}

func TestUpsertAnomalyOnNodes_InvalidTargetLabel(t *testing.T) {
	s := &Store{}
	_, err := s.UpsertAnomalyOnNodes(context.Background(),
		map[string]any{"id": "ANOM-1"},
		[]Endpoint{{Label: "BadLabel", ID: "1"}}, nil,
	)
	if err == nil {
		t.Fatal("expected error for invalid target label")
	}
}

func TestUpsertAnomalyOnNodes_MultipleTargets_OneInvalid(t *testing.T) {
	s := &Store{}
	_, err := s.UpsertAnomalyOnNodes(context.Background(),
		map[string]any{"id": "ANOM-1"},
		[]Endpoint{
			{Label: "Application", ID: "1"},
			{Label: "InvalidLabel", ID: "2"},
		}, nil,
	)
	if err == nil {
		t.Fatal("expected error when one target label is invalid")
	}
}
