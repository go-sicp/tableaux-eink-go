package einklib

// FillSolid paints every pixel of frame with the given 0xRRGGBB colour.
// The alpha byte is ignored. On grayscale panels the luminance is computed
// per Rec. 709.
func FillSolid(f Frame, argb uint32) {
	r := byte((argb >> 16) & 0xFF)
	g := byte((argb >> 8) & 0xFF)
	b := byte(argb & 0xFF)
	if f.Info().IsColor() {
		px := f.Pixels()
		for i := 0; i+2 < len(px); i += 3 {
			px[i], px[i+1], px[i+2] = r, g, b
		}
		return
	}
	// 4-bit grayscale, 2 pixels per byte. luma in [0..255] -> nibble [0..15]
	luma := byte((uint32(r)*54 + uint32(g)*183 + uint32(b)*19) >> 8)
	nib := luma >> 4
	pair := (nib << 4) | nib
	px := f.Pixels()
	for i := range px {
		px[i] = pair
	}
}

// PaintRGB888 copies a flat RGB888 buffer into the frame. Length must
// match Frame.Info().FrameByteCount() for colour panels.
func PaintRGB888(f Frame, rgb []byte) {
	dst := f.Pixels()
	n := len(rgb)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst[:n], rgb[:n])
}

// PaintGrayscale copies a flat 4-bit grayscale buffer (2 px/byte, low
// nibble first) into the frame.
func PaintGrayscale(f Frame, gray []byte) {
	dst := f.Pixels()
	n := len(gray)
	if n > len(dst) {
		n = len(dst)
	}
	copy(dst[:n], gray[:n])
}
