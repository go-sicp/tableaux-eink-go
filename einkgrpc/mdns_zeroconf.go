package einkgrpc

import (
	"fmt"
	"os"

	"github.com/grandcat/zeroconf"
)

// NewMdnsRegistrar returns a registrar backed by github.com/grandcat/zeroconf.
func NewMdnsRegistrar() MdnsRegistrar {
	return &zcRegistrar{}
}

type zcRegistrar struct {
	mdnsBase
	server *zeroconf.Server
}

func (z *zcRegistrar) Register(port int, panelIsColor bool, certFingerprint, version string) error {
	host, _ := os.Hostname()
	if host == "" {
		host = "device"
	}
	instance := HostnameInstance(host)
	panel := "grayscale"
	if panelIsColor {
		panel = "color"
	}
	txt := []string{
		"version=" + version,
		"panel=" + panel,
		"fingerprint=" + truncatedFingerprint(certFingerprint),
	}
	srv, err := zeroconf.Register(instance, ServiceType, "local.", port, txt, nil)
	if err != nil {
		return fmt.Errorf("zeroconf register: %w", err)
	}
	z.mu.Lock()
	z.name = instance
	z.server = srv
	z.mu.Unlock()
	return nil
}

func (z *zcRegistrar) Unregister() {
	z.mu.Lock()
	srv := z.server
	z.server = nil
	z.name = ""
	z.mu.Unlock()
	if srv != nil {
		srv.Shutdown()
	}
}

// truncatedFingerprint keeps only the first 16 hex bytes (32 chars) so the
// TXT record stays under the 255-byte mDNS limit even with many fields.
func truncatedFingerprint(fp string) string {
	if len(fp) <= 47 {
		return fp
	}
	return fp[:47]
}
