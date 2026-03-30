package graphstore

import (
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestToJSONValue_Nil(t *testing.T) {
	if got := ToJSONValue(nil); got != nil {
		t.Errorf("ToJSONValue(nil) = %v, want nil", got)
	}
}

func TestToJSONValue_Primitives(t *testing.T) {
	if got := ToJSONValue(42); got != 42 {
		t.Errorf("ToJSONValue(42) = %v", got)
	}
	if got := ToJSONValue("hello"); got != "hello" {
		t.Errorf("ToJSONValue(hello) = %v", got)
	}
	if got := ToJSONValue(3.14); got != 3.14 {
		t.Errorf("ToJSONValue(3.14) = %v", got)
	}
	if got := ToJSONValue(true); got != true {
		t.Errorf("ToJSONValue(true) = %v", got)
	}
}

func TestToJSONValue_Node(t *testing.T) {
	n := neo4j.Node{
		Labels: []string{"Application"},
		Props:  map[string]any{"id": "app-1", "name": "TestApp"},
	}
	got := ToJSONValue(n)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("ToJSONValue(Node) type = %T, want map", got)
	}
	if m["id"] != "app-1" {
		t.Errorf("id = %v", m["id"])
	}
	if m["name"] != "TestApp" {
		t.Errorf("name = %v", m["name"])
	}
	labels, ok := m["labels"].([]string)
	if !ok || len(labels) != 1 || labels[0] != "Application" {
		t.Errorf("labels = %v", m["labels"])
	}
}

func TestToJSONValue_Relationship(t *testing.T) {
	r := neo4j.Relationship{
		Type:           "CALLS",
		ElementId:      "elem-1",
		StartElementId: "start-1",
		EndElementId:   "end-1",
		Props:          map[string]any{"latency": 100},
	}
	got := ToJSONValue(r)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("ToJSONValue(Relationship) type = %T, want map", got)
	}
	if m["type"] != "CALLS" {
		t.Errorf("type = %v", m["type"])
	}
	if m["latency"] != 100 {
		t.Errorf("latency = %v", m["latency"])
	}
}

func TestToJSONValue_Path(t *testing.T) {
	p := neo4j.Path{
		Nodes: []neo4j.Node{
			{Labels: []string{"A"}, Props: map[string]any{"id": "1"}},
			{Labels: []string{"B"}, Props: map[string]any{"id": "2"}},
		},
		Relationships: []neo4j.Relationship{
			{Type: "CALLS", Props: map[string]any{}},
		},
	}
	got := ToJSONValue(p)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("ToJSONValue(Path) type = %T, want map", got)
	}
	nodes, ok := m["nodes"].([]map[string]any)
	if !ok || len(nodes) != 2 {
		t.Errorf("nodes = %v", m["nodes"])
	}
	rels, ok := m["relationships"].([]map[string]any)
	if !ok || len(rels) != 1 {
		t.Errorf("relationships = %v", m["relationships"])
	}
}

func TestToJSONValue_Point2D(t *testing.T) {
	p := neo4j.Point2D{SpatialRefId: 4326, X: 1.0, Y: 2.0}
	got := ToJSONValue(p)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("type = %T", got)
	}
	if m["x"] != 1.0 || m["y"] != 2.0 {
		t.Errorf("point2d = %v", m)
	}
}

func TestToJSONValue_Point3D(t *testing.T) {
	p := neo4j.Point3D{SpatialRefId: 4979, X: 1.0, Y: 2.0, Z: 3.0}
	got := ToJSONValue(p)
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("type = %T", got)
	}
	if m["z"] != 3.0 {
		t.Errorf("point3d = %v", m)
	}
}

func TestNodeMap(t *testing.T) {
	n := neo4j.Node{
		Labels: []string{"Storage"},
		Props:  map[string]any{"id": "db-1", "type": "SQL", "capacity": "500GB"},
	}
	m := nodeMap(n)
	if m["id"] != "db-1" {
		t.Errorf("id = %v", m["id"])
	}
	if m["type"] != "SQL" {
		t.Errorf("type = %v", m["type"])
	}
	labels := m["labels"].([]string)
	if len(labels) != 1 || labels[0] != "Storage" {
		t.Errorf("labels = %v", labels)
	}
}

func TestRelMap(t *testing.T) {
	r := neo4j.Relationship{
		Type:           "USES_STORAGE",
		ElementId:      "e1",
		StartElementId: "s1",
		EndElementId:   "e2",
		Props:          map[string]any{"startTime": "2026-01-01"},
	}
	m := relMap(r)
	if m["type"] != "USES_STORAGE" {
		t.Errorf("type = %v", m["type"])
	}
	if m["startTime"] != "2026-01-01" {
		t.Errorf("startTime = %v", m["startTime"])
	}
	if m["elementId"] != "e1" {
		t.Errorf("elementId = %v", m["elementId"])
	}
}

func TestPathMap(t *testing.T) {
	p := neo4j.Path{
		Nodes:         []neo4j.Node{},
		Relationships: []neo4j.Relationship{},
	}
	m := pathMap(p)
	nodes := m["nodes"].([]map[string]any)
	rels := m["relationships"].([]map[string]any)
	if len(nodes) != 0 || len(rels) != 0 {
		t.Errorf("empty path should have empty slices")
	}
}

func TestNodeMap_NestedNeo4jValue(t *testing.T) {
	inner := neo4j.Point2D{SpatialRefId: 4326, X: 10, Y: 20}
	n := neo4j.Node{
		Labels: []string{"Network"},
		Props:  map[string]any{"id": "net-1", "location": inner},
	}
	m := nodeMap(n)
	loc, ok := m["location"].(map[string]any)
	if !ok {
		t.Fatalf("location type = %T, want map", m["location"])
	}
	if loc["x"] != float64(10) {
		t.Errorf("x = %v", loc["x"])
	}
}
