package graphstore

import (
	"testing"
)

func TestValidateLabel_Allowed(t *testing.T) {
	labels := []string{"Application", "Storage", "Network", "IncidentTicket", "ChangeTicket", "RCATicket", "Action", "Anomaly", "Call"}
	for _, l := range labels {
		if err := validateLabel(l); err != nil {
			t.Errorf("validateLabel(%q) = %v, want nil", l, err)
		}
	}
}

func TestValidateLabel_Rejected(t *testing.T) {
	bad := []string{"", "Foo", "application", "NETWORK", "Drop;--"}
	for _, l := range bad {
		if err := validateLabel(l); err == nil {
			t.Errorf("validateLabel(%q) = nil, want error", l)
		}
	}
}

func TestValidateRelType_Allowed(t *testing.T) {
	types := []string{"CALLS", "USES_STORAGE", "CONNECTS_TO", "STORED_ON_NETWORK", "IMPACTS", "AFFECTS", "ROOT_CAUSE_OF", "HAS_ACTION", "DETECTED_ON", "TO", "DEPENDS_ON_TRANSITIVE"}
	for _, rt := range types {
		if err := validateRelType(rt); err != nil {
			t.Errorf("validateRelType(%q) = %v, want nil", rt, err)
		}
	}
}

func TestValidateRelType_Rejected(t *testing.T) {
	bad := []string{"", "calls", "UNKNOWN", "DELETE_ALL"}
	for _, rt := range bad {
		if err := validateRelType(rt); err == nil {
			t.Errorf("validateRelType(%q) = nil, want error", rt)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]struct{}{"c": {}, "a": {}, "b": {}}
	got := sortedKeys(m)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("sortedKeys = %v, want [a b c]", got)
	}
}

func TestSortedKeys_Empty(t *testing.T) {
	got := sortedKeys(map[string]struct{}{})
	if len(got) != 0 {
		t.Errorf("sortedKeys(empty) = %v, want []", got)
	}
}

func TestClampDepth(t *testing.T) {
	tests := []struct {
		d, min, max, want int
	}{
		{5, 1, 20, 5},
		{0, 1, 20, 1},
		{-1, 1, 20, 1},
		{25, 1, 20, 20},
		{1, 1, 1, 1},
		{10, 5, 15, 10},
	}
	for _, tt := range tests {
		got := clampDepth(tt.d, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clampDepth(%d, %d, %d) = %d, want %d", tt.d, tt.min, tt.max, got, tt.want)
		}
	}
}
