//go:build android

package einkapp

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/go-sicp/androidsvc"
	asvclog "github.com/go-sicp/androidsvc/log"
	"github.com/go-sicp/androidsvc/notif"
	"github.com/go-sicp/tableaux-eink-go/einkgrpc"
	"github.com/go-sicp/tableaux-eink-go/einklib"
)

const ServerVersion = "0.1.0"

var (
	mu         sync.Mutex
	inited     bool
	filesDir   string
	devPath    = einklib.DefaultDevice
	display    einklib.Display
	server     *einkgrpc.Server
	asvcSvc    androidsvc.Service
	logBuf     asvclog.Buffer
	channel    notif.Channel
	observer   LogObserver
	stopFanout func()
)

func doInit(dir string) error {
	mu.Lock()
	defer mu.Unlock()
	if inited {
		return nil
	}
	if dir == "" {
		return errors.New("einkapp: filesDir is required")
	}
	filesDir = dir

	channel = notif.NewChannel(
		notif.WithChannelID("tableaux-eink"),
		notif.WithChannelName("Tableaux ePaper server"),
		notif.WithImportance(notif.ImportanceLow),
	)
	asvcSvc = androidsvc.New(
		androidsvc.WithChannel(channel),
		androidsvc.WithLogCapacity(2000),
		androidsvc.WithNotificationContent("Tableaux ePaper server", "starting…", 0),
		androidsvc.WithForegroundServiceType(1), // dataSync
	)
	androidsvc.Register(asvcSvc)
	logBuf = asvcSvc.Logs()

	slog.SetDefault(slog.New(asvclog.NewHandler(logBuf, slog.LevelInfo)))
	startObserverFanout()

	slog.Info("einkapp init", "filesDir", filesDir)
	inited = true
	return nil
}

func doStart() error {
	mu.Lock()
	defer mu.Unlock()
	if !inited {
		return ErrNotInit
	}
	if server != nil {
		return nil
	}

	d, err := einklib.Open(devPath)
	if err != nil {
		slog.Error("einklib.Open failed", "err", err.Error(), "device", devPath)
		// Continue: server can run without display for tests; RPCs that
		// need it will return FailedPrecondition.
	} else {
		display = d
	}

	srv, err := einkgrpc.New(einkgrpc.Config{
		Display:       display,
		FilesDir:      filesDir,
		Port:          einkgrpc.DefaultPort,
		ServerVersion: ServerVersion,
		EnableMTLS:    true,
		EnableMDNS:    true,
	})
	if err != nil {
		return err
	}
	if err := srv.Start(); err != nil {
		return err
	}
	server = srv

	n := notif.New(channel,
		notif.WithTitle("Tableaux ePaper server"),
		notif.WithText(fmt.Sprintf("listening on :%d", srv.ListeningPort())),
		notif.WithOngoing(true),
	)
	if err := asvcSvc.Start(n); err != nil {
		_ = server.Stop()
		server = nil
		return err
	}
	slog.Info("server started", "port", srv.ListeningPort(), "fingerprint", srv.CertManager().Fingerprint())
	return nil
}

func doStop() error {
	mu.Lock()
	srv := server
	d := display
	server = nil
	display = nil
	mu.Unlock()

	if srv != nil {
		if err := srv.Stop(); err != nil {
			slog.Warn("server stop", "err", err.Error())
		}
	}
	if asvcSvc != nil {
		if err := asvcSvc.Stop(); err != nil {
			slog.Warn("asvc stop", "err", err.Error())
		}
	}
	if d != nil {
		_ = d.Close()
	}
	return nil
}

func doIsRunning() bool {
	mu.Lock()
	defer mu.Unlock()
	return server != nil && server.IsRunning()
}

func doToken() string {
	mu.Lock()
	defer mu.Unlock()
	if server == nil {
		return ""
	}
	return server.CertManager().Token()
}

func doAdminToken() string {
	mu.Lock()
	defer mu.Unlock()
	if server == nil {
		return ""
	}
	return server.CertManager().AdminToken()
}

func doFingerprint() string {
	mu.Lock()
	defer mu.Unlock()
	if server == nil {
		return ""
	}
	return server.CertManager().Fingerprint()
}

func doClientFingerprint() string {
	mu.Lock()
	defer mu.Unlock()
	if server == nil {
		return ""
	}
	return server.CertManager().ClientFingerprint()
}

func doPort() int {
	mu.Lock()
	defer mu.Unlock()
	if server == nil {
		return 0
	}
	return server.ListeningPort()
}

func doMdnsName() string {
	// MdnsRegistrar.RegisteredName is reachable only via the Server's
	// internal handle; v0.1 returns "" until that handle is exposed.
	return ""
}

func doConnectURI(host string) string {
	mu.Lock()
	defer mu.Unlock()
	if server == nil {
		return ""
	}
	return server.ConnectURI(host)
}

func doSetDevicePath(p string) {
	mu.Lock()
	defer mu.Unlock()
	if p == "" {
		devPath = einklib.DefaultDevice
		return
	}
	devPath = p
}

func doDevicePath() string {
	mu.Lock()
	defer mu.Unlock()
	return devPath
}

func doRotate() error {
	mu.Lock()
	srv := server
	mu.Unlock()
	if srv == nil {
		return errors.New("einkapp: server not running")
	}
	if err := srv.CertManager().Rotate(); err != nil {
		return err
	}
	if err := srv.Stop(); err != nil {
		return err
	}
	mu.Lock()
	server = nil
	mu.Unlock()
	return doStart()
}

func doLogsSnapshot(n int) string {
	if logBuf == nil {
		return ""
	}
	entries := logBuf.Snapshot(n)
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "%d %s %s\n", e.Time().UnixMilli(), e.Level().String(), e.Message())
	}
	return b.String()
}

func doRegisterLogObserver(o LogObserver) {
	mu.Lock()
	observer = o
	mu.Unlock()
}

func startObserverFanout() {
	if logBuf == nil {
		return
	}
	if stopFanout != nil {
		return
	}
	ch, cancel := logBuf.Subscribe(64)
	stopFanout = cancel
	go func() {
		for e := range ch {
			mu.Lock()
			obs := observer
			mu.Unlock()
			if obs != nil {
				obs.OnEntry(e.Time().UnixMilli(), int32(e.Level()), e.Message())
			}
		}
	}()
}
