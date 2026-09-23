//go:build !linux

package einklib

import "errors"

var errNotLinux = errors.New("einklib: only available on linux/android (cgo + /dev/ebc)")

func Open(devicePath string) (Display, error) { return nil, errNotLinux }
