package einklib

// WaveformMode selects how the EBC controller flips the panel cells.
// Numeric values mirror the kernel `enum panel_refresh_mode` so the cast
// to int32 in the ioctl is a no-op.
type WaveformMode int32

const (
	WaveformInit                WaveformMode = 0
	WaveformDirectUpdate        WaveformMode = 1
	WaveformGC16                WaveformMode = 2
	WaveformGCC16               WaveformMode = 3
	WaveformAnimation           WaveformMode = 4
	WaveformPartial             WaveformMode = 5
	WaveformFull                WaveformMode = 6
	WaveformAuto                WaveformMode = 7
	WaveformReset               WaveformMode = 8
	WaveformBlackWhite          WaveformMode = 9
	WaveformTransparent         WaveformMode = 10
	WaveformRegal               WaveformMode = 11
	WaveformSpectraFullGC16     WaveformMode = 12
	WaveformSpectraGlareReduced WaveformMode = 13
	WaveformSpectraGhostReduced WaveformMode = 14
	WaveformSpectraFullGCC16    WaveformMode = 15
)

// String returns a stable human-readable name. Useful for logs and CLI parsing.
func (m WaveformMode) String() string {
	switch m {
	case WaveformInit:
		return "Init"
	case WaveformDirectUpdate:
		return "DirectUpdate"
	case WaveformGC16:
		return "GC16"
	case WaveformGCC16:
		return "GCC16"
	case WaveformAnimation:
		return "Animation"
	case WaveformPartial:
		return "Partial"
	case WaveformFull:
		return "Full"
	case WaveformAuto:
		return "Auto"
	case WaveformReset:
		return "Reset"
	case WaveformBlackWhite:
		return "BlackWhite"
	case WaveformTransparent:
		return "Transparent"
	case WaveformRegal:
		return "Regal"
	case WaveformSpectraFullGC16:
		return "SpectraFullGC16"
	case WaveformSpectraGlareReduced:
		return "SpectraGlareReduced"
	case WaveformSpectraGhostReduced:
		return "SpectraGhostReduced"
	case WaveformSpectraFullGCC16:
		return "SpectraFullGCC16"
	}
	return "Unknown"
}

// ParseWaveformMode is the inverse of String. Comparison is case-insensitive.
func ParseWaveformMode(s string) (WaveformMode, bool) {
	for m := WaveformInit; m <= WaveformSpectraFullGCC16; m++ {
		if eqFold(m.String(), s) {
			return m, true
		}
	}
	return 0, false
}

func eqFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}
