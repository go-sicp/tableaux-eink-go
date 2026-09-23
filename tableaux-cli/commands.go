package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/grandcat/zeroconf"

	pb "github.com/go-sicp/tableaux-eink-go/einkproto"
)

const cliCallTimeout = 30 * time.Second

func cmdInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	c := &commonFlags{}
	registerCommonFlags(fs, c)
	fs.Parse(args)
	if err := c.validate(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cliCallTimeout)
	defer cancel()
	conn, ctx, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := pb.NewDisplayClient(conn)
	info, err := cli.GetInfo(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	fmt.Printf("width:          %d\n", info.Width)
	fmt.Printf("height:         %d\n", info.Height)
	fmt.Printf("virtual_width:  %d\n", info.VirtualWidth)
	fmt.Printf("virtual_height: %d\n", info.VirtualHeight)
	fmt.Printf("color:          %v\n", info.Color)
	fmt.Printf("server_version: %s\n", info.AppVersion)
	return nil
}

func cmdHealth(args []string) error {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	c := &commonFlags{}
	registerCommonFlags(fs, c)
	fs.Parse(args)
	if err := c.validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cliCallTimeout)
	defer cancel()
	conn, ctx, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := pb.NewDisplayClient(conn)
	h, err := cli.Health(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	fmt.Printf("device_open:    %v\n", h.DeviceOpen)
	fmt.Printf("server_version: %s\n", h.ServerVersion)
	fmt.Printf("uptime_seconds: %d\n", h.UptimeSeconds)
	if h.LastError != "" {
		fmt.Printf("last_error:     %s\n", h.LastError)
	}
	return nil
}

func cmdFill(args []string) error {
	fs := flag.NewFlagSet("fill", flag.ExitOnError)
	c := &commonFlags{}
	registerCommonFlags(fs, c)
	var argb, mode string
	fs.StringVar(&argb, "argb", "0xFFFFFF", "0xRRGGBB")
	fs.StringVar(&mode, "mode", "GC16", "WaveformMode (e.g. GC16, SpectraFullGC16)")
	fs.Parse(args)
	if err := c.validate(); err != nil {
		return err
	}
	parsed, err := strconv.ParseUint(strings.TrimPrefix(strings.ToLower(argb), "0x"), 16, 32)
	if err != nil {
		return fmt.Errorf("--argb: %w", err)
	}
	wm, ok := parseWaveform(mode)
	if !ok {
		return fmt.Errorf("--mode: unknown waveform %q", mode)
	}
	ctx, cancel := context.WithTimeout(context.Background(), cliCallTimeout)
	defer cancel()
	conn, ctx, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := pb.NewDisplayClient(conn)
	resp, err := cli.Fill(ctx, &pb.FillRequest{Argb: uint32(parsed), Mode: wm})
	if err != nil {
		return err
	}
	fmt.Printf("elapsed_ms: %d\n", resp.ElapsedMs)
	return nil
}

func cmdRefresh(args []string) error {
	fs := flag.NewFlagSet("refresh", flag.ExitOnError)
	c := &commonFlags{}
	registerCommonFlags(fs, c)
	var pixelsPath, mode string
	var chunkBytes int
	fs.StringVar(&pixelsPath, "pixels", "", "path to RGB888 / 4-bit-gray frame")
	fs.StringVar(&mode, "mode", "SpectraFullGC16", "WaveformMode")
	fs.IntVar(&chunkBytes, "chunk-bytes", 65536, "stream chunk size")
	fs.Parse(args)
	if err := c.validate(); err != nil {
		return err
	}
	if pixelsPath == "" {
		return errors.New("--pixels is required")
	}
	wm, ok := parseWaveform(mode)
	if !ok {
		return fmt.Errorf("--mode: unknown waveform %q", mode)
	}
	pixels, err := os.ReadFile(pixelsPath)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	conn, ctx, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := pb.NewDisplayClient(conn)
	stream, err := cli.Refresh(ctx)
	if err != nil {
		return err
	}
	if err := stream.Send(&pb.RefreshChunk{Payload: &pb.RefreshChunk_Header{
		Header: &pb.RefreshHeader{Mode: wm, TotalBytes: int64(len(pixels))},
	}}); err != nil {
		return err
	}
	for off := 0; off < len(pixels); off += chunkBytes {
		end := off + chunkBytes
		if end > len(pixels) {
			end = len(pixels)
		}
		if err := stream.Send(&pb.RefreshChunk{Payload: &pb.RefreshChunk_Data{Data: pixels[off:end]}}); err != nil {
			return err
		}
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}
	fmt.Printf("elapsed_ms: %d\n", resp.ElapsedMs)
	return nil
}

func cmdOverlay(args []string) error {
	fs := flag.NewFlagSet("overlay", flag.ExitOnError)
	c := &commonFlags{}
	registerCommonFlags(fs, c)
	var enabled bool
	fs.BoolVar(&enabled, "enabled", false, "enable overlay")
	fs.Parse(args)
	if err := c.validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cliCallTimeout)
	defer cancel()
	conn, ctx, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := pb.NewDisplayClient(conn)
	_, err = cli.SetOverlay(ctx, &pb.SetOverlayRequest{Enabled: enabled})
	if err != nil {
		return err
	}
	fmt.Println("ok")
	return nil
}

func cmdRotate(args []string) error {
	fs := flag.NewFlagSet("rotate", flag.ExitOnError)
	c := &commonFlags{}
	registerCommonFlags(fs, c)
	fs.Parse(args)
	if err := c.validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cliCallTimeout)
	defer cancel()
	conn, ctx, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := pb.NewDisplayClient(conn)
	resp, err := cli.Rotate(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	fmt.Printf("scheduled_at_ms: %d\n", resp.ScheduledAtMs)
	fmt.Println("warning: server is restarting; pull fresh tls/ from device.")
	return nil
}

func cmdReissue(args []string) error {
	fs := flag.NewFlagSet("reissue", flag.ExitOnError)
	c := &commonFlags{}
	registerCommonFlags(fs, c)
	var outDir string
	fs.StringVar(&outDir, "out", "./tls", "output directory for client cert/key")
	fs.Parse(args)
	if err := c.validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cliCallTimeout)
	defer cancel()
	conn, ctx, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := pb.NewDisplayClient(conn)
	resp, err := cli.ReissueClientCert(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "client-cert.pem"), []byte(resp.ClientCertPem), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "client-key.pem"), []byte(resp.ClientKeyPem), 0o600); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", filepath.Join(outDir, "client-cert.pem"))
	fmt.Printf("wrote %s\n", filepath.Join(outDir, "client-key.pem"))
	fmt.Printf("fingerprint: %s\n", resp.ClientCertFingerprint)
	fmt.Printf("server restart in: %d ms\n", resp.ServerRestartInMs)
	return nil
}

func cmdMetrics(args []string) error {
	fs := flag.NewFlagSet("metrics", flag.ExitOnError)
	c := &commonFlags{}
	registerCommonFlags(fs, c)
	fs.Parse(args)
	if err := c.validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cliCallTimeout)
	defer cancel()
	conn, ctx, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := pb.NewDisplayClient(conn)
	r, err := cli.GetMetrics(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	fmt.Print(r.PromText)
	return nil
}

func cmdDiscover(args []string) error {
	fs := flag.NewFlagSet("discover", flag.ExitOnError)
	timeout := fs.Int("timeout", 5, "browse window in seconds")
	first := fs.Bool("first", false, "exit after first answer")
	fs.Parse(args)

	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry, 16)
	if err := resolver.Browse(ctx, "_tableaux-eink._tcp.", "local.", entries); err != nil {
		return err
	}
	for e := range entries {
		fmt.Printf("%s@%s\n", e.Instance, e.AddrIPv4)
		fmt.Printf("  port:        %d\n", e.Port)
		fmt.Printf("  addresses:   %v\n", e.AddrIPv4)
		for _, t := range e.Text {
			fmt.Printf("  %s\n", t)
		}
		if *first {
			return nil
		}
	}
	return nil
}

func cmdConnect(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: tableaux-cli connect <tableaux://...>")
	}
	u, err := url.Parse(args[0])
	if err != nil {
		return err
	}
	if u.Scheme != "tableaux" {
		return fmt.Errorf("scheme: %q (expected tableaux://)", u.Scheme)
	}
	q := u.Query()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "50051"
	}
	fmt.Printf("host:        %s\n", host)
	fmt.Printf("port:        %s\n", port)
	fmt.Printf("token:       %s\n", q.Get("token"))
	fmt.Printf("fingerprint: %s\n", q.Get("fp"))
	fmt.Println()
	fmt.Println("Equivalent flags:")
	fmt.Printf("  -H %s -p %s -t %s\n", host, port, q.Get("token"))
	return nil
}

// parseWaveform maps a CLI string (case-insensitive) to a proto enum value.
func parseWaveform(s string) (pb.WaveformMode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "init":
		return pb.WaveformMode_WAVEFORM_MODE_INIT, true
	case "directupdate", "direct_update":
		return pb.WaveformMode_WAVEFORM_MODE_DIRECT_UPDATE, true
	case "gc16":
		return pb.WaveformMode_WAVEFORM_MODE_GC16, true
	case "gcc16":
		return pb.WaveformMode_WAVEFORM_MODE_GCC16, true
	case "animation":
		return pb.WaveformMode_WAVEFORM_MODE_ANIMATION, true
	case "partial":
		return pb.WaveformMode_WAVEFORM_MODE_PARTIAL, true
	case "full":
		return pb.WaveformMode_WAVEFORM_MODE_FULL, true
	case "auto":
		return pb.WaveformMode_WAVEFORM_MODE_AUTO, true
	case "reset":
		return pb.WaveformMode_WAVEFORM_MODE_RESET, true
	case "blackwhite", "black_white":
		return pb.WaveformMode_WAVEFORM_MODE_BLACK_WHITE, true
	case "transparent":
		return pb.WaveformMode_WAVEFORM_MODE_TRANSPARENT, true
	case "regal":
		return pb.WaveformMode_WAVEFORM_MODE_REGAL, true
	case "spectrafullgc16", "spectra_full_gc16":
		return pb.WaveformMode_WAVEFORM_MODE_SPECTRA_FULL_GC16, true
	case "spectraglarereduced", "spectra_glare_reduced":
		return pb.WaveformMode_WAVEFORM_MODE_SPECTRA_GLARE_REDUCED, true
	case "spectraghostreduced", "spectra_ghost_reduced":
		return pb.WaveformMode_WAVEFORM_MODE_SPECTRA_GHOST_REDUCED, true
	case "spectrafullgcc16", "spectra_full_gcc16":
		return pb.WaveformMode_WAVEFORM_MODE_SPECTRA_FULL_GCC16, true
	}
	return pb.WaveformMode_WAVEFORM_MODE_UNSPECIFIED, false
}
