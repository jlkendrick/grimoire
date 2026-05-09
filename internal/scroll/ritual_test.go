package scroll_test

import (
	"testing"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

func TestRitualHash_StableAndOrderSensitive(t *testing.T) {
	r1 := scroll.Ritual{
		Command: "pipe",
		Steps:   []scroll.Step{{Spell: "a"}, {Spell: "b"}},
	}
	r2 := scroll.Ritual{
		Command: "pipe",
		Steps:   []scroll.Step{{Spell: "b"}, {Spell: "a"}},
	}

	h1a, err := r1.Hash()
	if err != nil {
		t.Fatalf("Hash r1: %v", err)
	}
	h1b, err := r1.Hash()
	if err != nil {
		t.Fatalf("Hash r1 again: %v", err)
	}
	if h1a != h1b {
		t.Errorf("Ritual.Hash unstable: %q vs %q", h1a, h1b)
	}

	h2, err := r2.Hash()
	if err != nil {
		t.Fatalf("Hash r2: %v", err)
	}
	if h1a == h2 {
		t.Errorf("expected reordered Steps to produce a different hash; both = %q", h1a)
	}
}
