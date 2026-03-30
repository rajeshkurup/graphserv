package graphstore

import (
	"testing"
)

func TestSQuote(t *testing.T) {
	got := sQuote([]string{"CALLS", "USES_STORAGE"})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != "'CALLS'" {
		t.Errorf("got[0] = %q, want 'CALLS'", got[0])
	}
	if got[1] != "'USES_STORAGE'" {
		t.Errorf("got[1] = %q, want 'USES_STORAGE'", got[1])
	}
}

func TestSQuote_Empty(t *testing.T) {
	got := sQuote([]string{})
	if len(got) != 0 {
		t.Errorf("sQuote(empty) = %v, want []", got)
	}
}

func TestSQuote_EscapesSingleQuote(t *testing.T) {
	got := sQuote([]string{"it's"})
	if got[0] != "'it\\'s'" {
		t.Errorf("got = %q", got[0])
	}
}
