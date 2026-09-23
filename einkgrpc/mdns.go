package einkgrpc

import (
	"fmt"
	"sync"
)

// MdnsRegistrar advertises the gRPC server as `_tableaux-eink._tcp.` on
// the LAN with TXT records `version`, `panel`, and `fingerprint`.
//
// v0.1 ships an interface but the concrete advertiser is gated to systems
// where the third-party zeroconf package is available; otherwise it is a
// no-op (see mdns_zeroconf.go vs mdns_stub.go).
type MdnsRegistrar interface {
	Register(port int, panelIsColor bool, certFingerprint, version string) error
	Unregister()
	RegisteredName() string
}

// SERVICE_TYPE follows Bonjour conventions.
const ServiceType = "_tableaux-eink._tcp."

type mdnsBase struct {
	mu   sync.Mutex
	name string
}

func (m *mdnsBase) RegisteredName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.name
}

// HostnameInstance composes the instance name advertised on mDNS.
func HostnameInstance(prefix string) string {
	if prefix == "" {
		prefix = "tableaux-eink"
	}
	return fmt.Sprintf("%s-go", prefix)
}
