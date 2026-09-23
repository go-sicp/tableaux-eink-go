// Package einkapp is the entry point exposed to the host Android app via
// gomobile bind. It wires androidsvc (foreground service) with einkgrpc
// (TLS gRPC server) and einklib (EBC display backend).
//
// Every exported function signature is constrained to types gomobile bind
// can translate (bool, int32, int64, string, byte[], plus interfaces with
// such methods). Higher-level Go-only types stay unexported.
package einkapp
