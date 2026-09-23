// Package einklib drives the Rockchip EBC kernel device exposed at
// /dev/ebc on Philips Tableaux 32BDL5150I and compatible RK3566 panels.
//
// The Go API mirrors the Kotlin facade of tableaux-eink/eink-lib:
//
//	d, err := einklib.Open(einklib.DefaultDevice)
//	defer d.Close()
//	info, _ := d.Info()
//	fmt.Println(info.Width(), info.Height(), info.IsColor())
//
//	f, err := d.AcquireFrame()
//	einklib.FillSolid(f, 0xFFFFFF)
//	f.Commit(einklib.WaveformGC16, nil)
//
// All struct fields are unexported; access is via interface methods so the
// underlying ioctl layout can change without breaking callers.
package einklib
