//go:build linux

package einklib

/*
#include "ebc_ioctl.h"
#include <fcntl.h>
#include <unistd.h>
#include <stdlib.h>
#include <sys/ioctl.h>
#include <sys/mman.h>
#include <errno.h>
#include <string.h>

// Wrappers (cgo cannot call macros or variadic C functions directly).
static int ebc_get_info(int fd, struct ebc_buf_info *out) { return ioctl(fd, EBC_GET_BUFFER_INFO, out); }
static int ebc_get_buffer(int fd, struct ebc_buf_info *out) { return ioctl(fd, EBC_GET_BUFFER, out); }
static int ebc_send_buffer(int fd, struct ebc_buf_info *in) { return ioctl(fd, EBC_SEND_BUFFER, in); }
static int ebc_overlay_on (int fd) { return ioctl(fd, EBC_ENABLE_OVERLAY); }
static int ebc_overlay_off(int fd) { return ioctl(fd, EBC_DISABLE_OVERLAY); }
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

func unsafePointer(p any) unsafe.Pointer {
	switch v := p.(type) {
	case *byte:
		return unsafe.Pointer(v)
	}
	return nil
}

func newDisplayInfo(w, h, vw, vh int32, color bool) DisplayInfo {
	return DisplayInfo{width: w, height: h, virtualWidth: vw, virtHeight: vh, color: color}
}

// Open opens devicePath (typically /dev/ebc) for refresh control.
func Open(devicePath string) (Display, error) {
	fd, err := syscall.Open(devicePath, syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", devicePath, err)
	}
	return &display{fd: fd}, nil
}

type display struct {
	mu sync.Mutex
	fd int
}

func (d *display) Info() (DisplayInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fd < 0 {
		return DisplayInfo{}, os.ErrClosed
	}
	var info C.struct_ebc_buf_info
	if rc, err := C.ebc_get_info(C.int(d.fd), &info); rc < 0 {
		return DisplayInfo{}, fmt.Errorf("EBC_GET_BUFFER_INFO: %w", err)
	}
	return newDisplayInfo(int32(info.width), int32(info.height),
		int32(info.vir_width), int32(info.vir_height), info.panel_color != 0), nil
}

func (d *display) AcquireFrame() (Frame, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fd < 0 {
		return nil, os.ErrClosed
	}
	var info C.struct_ebc_buf_info
	if rc, err := C.ebc_get_buffer(C.int(d.fd), &info); rc < 0 {
		return nil, fmt.Errorf("EBC_GET_BUFFER: %w", err)
	}

	di := newDisplayInfo(int32(info.width), int32(info.height),
		int32(info.vir_width), int32(info.vir_height), info.panel_color != 0)
	bytes := di.FrameByteCount()

	mapped, err := syscall.Mmap(d.fd, 0, bytes,
		syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("mmap ebc buffer (%d bytes): %w", bytes, err)
	}
	bufferAddr := int64(uintptr(unsafePointer(&mapped[0])))

	return &frame{
		display:    d,
		info:       di,
		pixels:     mapped,
		bufferAddr: bufferAddr,
	}, nil
}

func (d *display) SetOverlay(enabled bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fd < 0 {
		return os.ErrClosed
	}
	var rc C.int
	var err error
	if enabled {
		rc, err = C.ebc_overlay_on(C.int(d.fd))
	} else {
		rc, err = C.ebc_overlay_off(C.int(d.fd))
	}
	if rc < 0 {
		return fmt.Errorf("overlay: %w", err)
	}
	return nil
}

func (d *display) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.fd < 0 {
		return nil
	}
	err := syscall.Close(d.fd)
	d.fd = -1
	return err
}

type frame struct {
	display    *display
	info       DisplayInfo
	pixels     []byte
	bufferAddr int64
	consumed   bool
}

func (f *frame) Pixels() []byte    { return f.pixels }
func (f *frame) Info() DisplayInfo { return f.info }

func (f *frame) Commit(mode WaveformMode, region *Region) error {
	if f.consumed {
		return errors.New("einklib: frame already consumed")
	}
	r := region
	if r == nil {
		full := NewRegion(0, 0, f.info.virtualWidth, f.info.virtHeight)
		r = &full
	}
	var info C.struct_ebc_buf_info
	info.width = C.int32_t(f.info.width)
	info.height = C.int32_t(f.info.height)
	info.vir_width = C.int32_t(f.info.virtualWidth)
	info.vir_height = C.int32_t(f.info.virtHeight)
	if f.info.color {
		info.panel_color = 1
	}
	info.win_x1 = C.int32_t(r.x1)
	info.win_y1 = C.int32_t(r.y1)
	info.win_x2 = C.int32_t(r.x2)
	info.win_y2 = C.int32_t(r.y2)
	info.epd_mode = C.int32_t(mode)
	info.buf_fd = -1
	info.needpic = 0
	info.buffer = C.int64_t(f.bufferAddr)

	if rc, err := C.ebc_send_buffer(C.int(f.display.fd), &info); rc < 0 {
		return fmt.Errorf("EBC_SEND_BUFFER: %w", err)
	}
	f.consumed = true
	_ = syscall.Munmap(f.pixels)
	return nil
}

func (f *frame) Cancel() error {
	if f.consumed {
		return nil
	}
	f.consumed = true
	return syscall.Munmap(f.pixels)
}
