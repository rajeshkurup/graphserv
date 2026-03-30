package graphstore

import (
	"testing"
)

func TestAnyInt(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int
	}{
		{"int", 42, 42},
		{"int32", int32(10), 10},
		{"int64", int64(99), 99},
		{"float64", float64(7.9), 7},
		{"string", "hello", 0},
		{"nil", nil, 0},
		{"bool", true, 0},
		{"negative_int", -5, -5},
		{"negative_int64", int64(-100), -100},
		{"zero_float", float64(0.0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := anyInt(tt.in)
			if got != tt.want {
				t.Errorf("anyInt(%v) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
