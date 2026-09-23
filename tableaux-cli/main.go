// Command tableaux-cli is the Go port of tableaux-eink/cli. It speaks
// Display gRPC over TLS+bearer to the device, and browses _tableaux-eink._tcp.
package main

import (
	"fmt"
	"os"
)

const usage = `tableaux-cli — Go port of tableaux-eink/cli

Usage:
  tableaux-cli <command> [flags...]

Commands:
  info                 Print panel geometry.
  health               Print server health.
  fill   --argb 0xRRGGBB --mode <Mode>
  refresh --pixels <file> --mode <Mode> [--chunk-bytes 65536]
  overlay --enabled
  rotate               Force-rotate every credential (admin).
  reissue --out ./tls  Reissue client cert+key (admin).
  metrics              Print Prometheus metrics.
  discover [--timeout 5] [--first]
  connect tableaux://...

Common flags (all RPC commands):
  -H, --host        Server host (required)
  -p, --port        Server port (default 50051)
  -t, --token       Bearer token (required unless --insecure-anon)
      --cert        Server cert PEM to trust (skip if --insecure)
      --client-cert Client cert PEM for mTLS
      --client-key  Client key PEM for mTLS
      --insecure    Skip TLS (dev only)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "info":
		err = cmdInfo(args)
	case "health":
		err = cmdHealth(args)
	case "fill":
		err = cmdFill(args)
	case "refresh":
		err = cmdRefresh(args)
	case "overlay":
		err = cmdOverlay(args)
	case "rotate":
		err = cmdRotate(args)
	case "reissue":
		err = cmdReissue(args)
	case "metrics":
		err = cmdMetrics(args)
	case "discover":
		err = cmdDiscover(args)
	case "connect":
		err = cmdConnect(args)
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", cmd, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
