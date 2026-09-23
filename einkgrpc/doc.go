// Package einkgrpc hosts the Go port of tableaux-eink/eink-grpc:
// a TLS-secured gRPC server implementing the Display service and a
// bearer-token interceptor.
//
// Differences from the Kotlin original (deferred to a later version):
//   - single role: there is no admin/operator split in v0.1; every token
//     can call every RPC.
//   - no rate limiter, no audit logger — both are operational features
//     not part of the wire-level functional parity.
package einkgrpc
