# tableaux-eink-go

Go port of the Kotlin project [`tableaux-eink/`](../tableaux-eink/), built
to **exercise [`androidsvc`](../androidsvc/)** on a realistic workload: an
Android 13 foreground service hosting a TLS gRPC server that drives the
Spectra 6 panel of a Philips Tableaux 32BDL5150I/00 via `/dev/ebc`.

## v0.1 status

| Module | Parity with Kotlin | Notes |
|---|---|---|
| `einkproto/`     | ✓ identical proto | Generated via protoc-gen-go / -go-grpc |
| `einklib/`       | ✓ ioctl, mmap, FillSolid, PaintRGB888 | cgo, linux-only build |
| `einkgrpc/`      | ✓ TLS + mTLS + bearer + mDNS + every RPC | No operator/admin role split (single token covers all RPCs), no rate limit, no audit log |
| `einkapp/`       | ✓ gomobile-bind entry points | Uses androidsvc to host the foreground service |
| `tableaux-cli/`  | ✓ info / health / fill / refresh / overlay / discover / connect / rotate / reissue / metrics | Uses stdlib `flag` (no cobra) |
| `android-host/`  | ✓ minimal Compose Activity | Material3, no QR rendering |

**Deferred to v0.2**: operator/admin role differentiation (two tokens),
per-principal SHA-256 rate limiter, audit logger, Compose QR-code render.
All flagged as `TODO` in the source.

## Architecture

```
android-host/   (Kotlin Compose)
       │  Einkapp.start()  ← via gomobile bind on einkapp/
       ▼
einkapp/   ─── androidsvc ────► foreground service (notification)
       │
       ▼
einkgrpc/  ── TLS + bearer ──► clients (tableaux-cli)
       │                        mDNS:_tableaux-eink._tcp.
       ▼
einklib/   ── cgo ioctl ─────► /dev/ebc (Spectra 6)
```

The gRPC server runs inside the Android process; its survival is guaranteed
by the foreground service driven by `androidsvc` (see Android 13
limitations documented in [`../androidsvc/README.md`](../androidsvc/README.md)).

## Build

### Prerequisites (developer workstation)

- Go ≥ 1.22
- `protoc` ≥ 25 plus `protoc-gen-go` and `protoc-gen-go-grpc` on `$PATH`
- `task` ≥ 3.40 ([taskfile.dev](https://taskfile.dev))
- For the APK: Android NDK r26+, Android SDK 33, `gomobile`,
  `gradle` ≥ 8.5

```sh
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

### Toolchain environment

`gomobile bind` and `gradle` need a few env vars set. On macOS with the
Homebrew-installed Android command-line tools, this works:

```sh
export ANDROID_HOME=/opt/homebrew/share/android-commandlinetools
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/27.0.12077973
export JAVA_HOME=/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home
export PATH=$JAVA_HOME/bin:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools:$HOME/go/bin:$PATH
```

Adjust the NDK and JDK paths to match your installed versions
(`ls $ANDROID_HOME/ndk/`).

### CLI

```sh
task build:cli
./bin/tableaux-cli help
```

### Android AAR

```sh
task build:aar       # gomobile bind einkapp -> android-host/app/libs/einkapp.aar
task build:apk       # gradle :app:assembleDebug
task adb:install     # adb install -r ...
task adb:run         # launch MainActivity
task adb:logcat      # filtered tail
```

### Regenerate proto stubs

```sh
task proto
```

### Go tests

```sh
task test
task vet
```

## Local validation (no NDK)

The developer workstation builds in stub mode:

```sh
go build ./...        # from tableaux-eink-go/
go vet ./...
```

`einklib/display_linux.go` is only compiled on `GOOS=linux`. On darwin
the `display_stub.go` returns "not supported", which is enough for `go vet`.

## Runtime walkthrough

1. On the device: install the APK, grant `POST_NOTIFICATIONS`.
2. Click **Start** in the Compose UI.
3. The server listens on `:50051` with TLS+mTLS+bearer.
4. From the workstation:

```sh
task adb:port-forward
task pull:tls
TOKEN=$(cat tls/token.txt)

bin/tableaux-cli info \
   -H 127.0.0.1 -t $TOKEN \
   --cert tls/cert.pem \
   --client-cert tls/client-cert.pem \
   --client-key tls/client-key.pem
```

5. From a LAN host: `bin/tableaux-cli discover`.

## APK permissions and deployment

`/dev/ebc` is `crw-rw---- root:graphics` on the Rockchip BSP. Same options
as the Kotlin project:

1. **`shell` over adb**: not viable — UID `shell` is not in `graphics`.
2. **Privileged system app**: `adb root && adb remount`, push under
   `/system/priv-app/TableauxEinkGo/`.
3. **Platform-key signing**: add `android:sharedUserId="android.uid.system"`.

The debug APK is enough to validate the foreground service boot and the
gRPC server. `Fill` and `Refresh` RPCs only succeed once one of the three
options above has unlocked `/dev/ebc`.

## Layout

```
tableaux-eink-go/
├── go.mod                    (module github.com/go-sicp/tableaux-eink-go)
├── Taskfile.yaml
├── README.md
├── einkproto/
│   ├── display.proto
│   ├── display.pb.go         (generated)
│   └── display_grpc.pb.go    (generated)
├── einklib/
│   ├── doc.go
│   ├── waveform.go
│   ├── display.go
│   ├── display_linux.go      (cgo + ioctl)
│   ├── display_stub.go
│   ├── bitmap.go
│   └── ebc_ioctl.h
├── einkgrpc/
│   ├── server.go
│   ├── displayservice.go
│   ├── certmanager.go
│   ├── tokenauth.go
│   ├── mdns.go
│   └── mdns_zeroconf.go
├── einkapp/                  (gomobile bind target)
│   ├── doc.go
│   ├── app.go
│   ├── app_android.go
│   └── app_stub.go
├── tableaux-cli/
│   ├── main.go
│   ├── dial.go
│   └── commands.go
└── android-host/
    ├── settings.gradle.kts
    ├── build.gradle.kts
    └── app/
        ├── build.gradle.kts
        └── src/main/
            ├── AndroidManifest.xml
            ├── res/values/strings.xml
            └── kotlin/io/gosicp/tableauxeink/
                └── MainActivity.kt
```

## Notable differences from the Kotlin version

- **Tokens**: v0.1 generates an operator token and an admin token, but the
  server does not check the role (any valid token is accepted on every
  RPC). Sufficient for `androidsvc` validation.
- **Crypto**: RSA-2048 self-signed, strict mTLS (the server's TrustManager
  is pinned to the single issued client cert), base64-url tokens
  (24 random bytes).
- **mDNS**: backed by `github.com/grandcat/zeroconf`. TXT records mirror
  the Kotlin original (`version`, `panel`, truncated `fingerprint`).
- **Compose UI**: minimal (Start/Stop, status, logs). No QR. The QR
  rendering can be delegated to a third-party Compose library (such as
  zxing-android-embedded) without touching the Go code.
- **No Kotlin coroutines**: the Go side uses goroutines + sync primitives.

## License

MIT (to be confirmed).
