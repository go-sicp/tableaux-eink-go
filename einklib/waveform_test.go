package einklib_test

import (
	"testing"

	"github.com/go-sicp/tableaux-eink-go/einklib"
)

func TestWaveformStringRoundtrip(t *testing.T) {
	for m := einklib.WaveformInit; m <= einklib.WaveformSpectraFullGCC16; m++ {
		s := m.String()
		if s == "Unknown" {
			t.Fatalf("mode %d: String() returned Unknown", m)
		}
		got, ok := einklib.ParseWaveformMode(s)
		if !ok || got != m {
			t.Fatalf("ParseWaveformMode(%q) = (%d, %v), want (%d, true)", s, got, ok, m)
		}
	}
}

func TestWaveformCaseInsensitive(t *testing.T) {
	got, ok := einklib.ParseWaveformMode("gc16")
	if !ok || got != einklib.WaveformGC16 {
		t.Fatalf("lowercase parse: %v %v", got, ok)
	}
	got, ok = einklib.ParseWaveformMode("SPECTRAFULLGC16")
	if !ok || got != einklib.WaveformSpectraFullGC16 {
		t.Fatalf("uppercase parse: %v %v", got, ok)
	}
}

func TestWaveformUnknown(t *testing.T) {
	if _, ok := einklib.ParseWaveformMode("nope"); ok {
		t.Fatal("expected ok=false for unknown mode")
	}
}

func TestUnknownModeStringStable(t *testing.T) {
	m := einklib.WaveformMode(999)
	if got := m.String(); got != "Unknown" {
		t.Fatalf("got %q, want Unknown", got)
	}
}
