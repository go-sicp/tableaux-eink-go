//go:build !android

package einkapp

func doInit(string) error               { return ErrNotInit }
func doStart() error                    { return ErrNotInit }
func doStop() error                     { return nil }
func doIsRunning() bool                 { return false }
func doToken() string                   { return "" }
func doAdminToken() string              { return "" }
func doFingerprint() string             { return "" }
func doClientFingerprint() string       { return "" }
func doPort() int                       { return 0 }
func doMdnsName() string                { return "" }
func doConnectURI(host string) string   { return "" }
func doSetDevicePath(string)            {}
func doDevicePath() string              { return "" }
func doRotate() error                   { return ErrNotInit }
func doLogsSnapshot(int) string         { return "" }
func doRegisterLogObserver(LogObserver) {}
