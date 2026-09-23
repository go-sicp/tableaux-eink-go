package einkapp

import "errors"

// LogObserver is the gomobile-bind-friendly interface the host UI implements
// to receive log entries pushed from Go.
type LogObserver interface {
	OnEntry(unixMillis int64, level int32, msg string)
}

// ErrNotInit is returned when the API is called before Init has succeeded.
var ErrNotInit = errors.New("einkapp: Init has not been called")

// Init bootstraps the package: builds the foreground-service controller,
// the in-memory log buffer, and attaches slog. filesDir is the writable
// directory where TLS material and tokens are persisted (typically
// ctx.getFilesDir() on Android).
//
// Idempotent. Safe to call multiple times.
func Init(filesDir string) error { return doInit(filesDir) }

// Start opens the EBC display, starts the gRPC server, and brings up the
// Android foreground service so the process is shielded from background
// kills. On Android 13 the host must already have requested
// POST_NOTIFICATIONS at runtime.
func Start() error { return doStart() }

// Stop tears down the gRPC server and asks Android to stop the foreground
// service. Safe to call multiple times.
func Stop() error { return doStop() }

// IsRunning reports the gRPC-server-side state. The Android service may
// still be alive briefly after Stop; query Android directly if you need
// exact lifecycle.
func IsRunning() bool { return doIsRunning() }

// Token returns the operator bearer token (regenerated on Rotate).
func Token() string { return doToken() }

// AdminToken returns the admin bearer token. v0.1 does not enforce a role
// separation; both tokens accept all RPCs.
func AdminToken() string { return doAdminToken() }

// Fingerprint is the SHA-256 of the server cert DER, colon-separated upper
// hex (e.g. "AB:CD:EF:...").
func Fingerprint() string { return doFingerprint() }

// ClientFingerprint is the SHA-256 of the issued client cert.
func ClientFingerprint() string { return doClientFingerprint() }

// Port is the actually-bound TCP port (0 before Start).
func Port() int32 { return int32(doPort()) }

// MdnsName is the instance name advertised on `_tableaux-eink._tcp.`
// (empty when mDNS is disabled or before Start).
func MdnsName() string { return doMdnsName() }

// ConnectURI returns the operator-friendly tableaux:// URL, suitable for
// QR-code generation. Pass the LAN IPv4 you want printed; falls back to
// 127.0.0.1.
func ConnectURI(host string) string { return doConnectURI(host) }

// SetDevicePath overrides the default /dev/ebc node before Start.
func SetDevicePath(p string) { doSetDevicePath(p) }

// DevicePath is the path Start will open. Default: /dev/ebc.
func DevicePath() string { return doDevicePath() }

// RotateNow regenerates every credential (server, client, tokens) and
// restarts the gRPC server. Existing client connections drop.
func RotateNow() error { return doRotate() }

// LogsSnapshot returns the last n lines from the in-memory log buffer as
// newline-joined "TIMESTAMP LEVEL MSG" lines. Pass 0 for everything.
func LogsSnapshot(n int32) string { return doLogsSnapshot(int(n)) }

// RegisterLogObserver pushes log entries as they are produced. Pass nil
// to unsubscribe.
func RegisterLogObserver(o LogObserver) { doRegisterLogObserver(o) }
