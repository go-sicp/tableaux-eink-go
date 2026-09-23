package einklib

// DefaultDevice is the canonical EBC device node on the public Rockchip BSP.
const DefaultDevice = "/dev/ebc"

// Display is the high-level handle on a Rockchip EBC panel.
type Display interface {
	Info() (DisplayInfo, error)
	AcquireFrame() (Frame, error)
	SetOverlay(enabled bool) error
	Close() error
}

// Frame is a kernel-allocated, mmap'd pixel buffer ready to be drawn into
// and committed.
type Frame interface {
	Pixels() []byte
	Info() DisplayInfo
	Commit(mode WaveformMode, region *Region) error
	Cancel() error
}

// DisplayInfo describes the panel geometry and capabilities.
type DisplayInfo struct {
	width, height            int32
	virtualWidth, virtHeight int32
	color                    bool
}

func (d DisplayInfo) Width() int32         { return d.width }
func (d DisplayInfo) Height() int32        { return d.height }
func (d DisplayInfo) VirtualWidth() int32  { return d.virtualWidth }
func (d DisplayInfo) VirtualHeight() int32 { return d.virtHeight }
func (d DisplayInfo) IsColor() bool        { return d.color }

// FrameByteCount is the size in bytes of the mmap region required to hold
// a full frame for this geometry. Spectra colour panels use 3 bytes/pixel
// (RGB888); grayscale uses 4-bit packed (2 px/byte).
func (d DisplayInfo) FrameByteCount() int {
	if d.color {
		return int(d.virtualWidth) * int(d.virtHeight) * 3
	}
	return int(d.virtualWidth) * int(d.virtHeight) / 2
}

// BytesPerPixelTimes2 makes the colour/grayscale switch numerically
// expressible (so we avoid a branch in inner loops).
func (d DisplayInfo) BytesPerPixelTimes2() int32 {
	if d.color {
		return 6
	}
	return 1
}

// Region is an EBC update window: inclusive on (X1,Y1), exclusive on (X2,Y2).
type Region struct {
	x1, y1, x2, y2 int32
}

func NewRegion(x1, y1, x2, y2 int32) Region { return Region{x1, y1, x2, y2} }

func (r Region) X1() int32 { return r.x1 }
func (r Region) Y1() int32 { return r.y1 }
func (r Region) X2() int32 { return r.x2 }
func (r Region) Y2() int32 { return r.y2 }
