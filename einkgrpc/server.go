package einkgrpc

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/go-sicp/tableaux-eink-go/einklib"
	pb "github.com/go-sicp/tableaux-eink-go/einkproto"
)

// Config is the construction-time settings of a Server.
type Config struct {
	Display       einklib.Display
	FilesDir      string
	Port          int
	MaxRecvBytes  int
	ServerVersion string
	EnableMTLS    bool
	EnableMDNS    bool
}

// Default port and message size match the Kotlin server.
const (
	DefaultPort         = 50051
	DefaultMaxRecvBytes = 16 * 1024 * 1024
)

// Server hosts a gRPC implementation backed by an einklib.Display.
type Server struct {
	cfg       Config
	cm        *CertManager
	mdns      MdnsRegistrar
	mu        sync.Mutex
	srv       *grpc.Server
	lis       net.Listener
	running   atomic.Bool
	port      int
	rotateReq chan struct{}
}

// New builds a non-started Server. cfg.Display must be non-nil if any
// rendering RPC is to succeed; nil is allowed for boot-without-device tests
// (Health and GetInfo will report the missing device).
func New(cfg Config) (*Server, error) {
	if cfg.FilesDir == "" {
		return nil, errors.New("einkgrpc: Config.FilesDir is required")
	}
	if cfg.Port == 0 {
		cfg.Port = DefaultPort
	}
	if cfg.MaxRecvBytes == 0 {
		cfg.MaxRecvBytes = DefaultMaxRecvBytes
	}
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = "0.1.0"
	}
	cm, err := NewCertManager(cfg.FilesDir)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:       cfg,
		cm:        cm,
		rotateReq: make(chan struct{}, 1),
	}, nil
}

// CertManager exposes the on-disk material owner so the host app can read
// fingerprints, tokens etc.
func (s *Server) CertManager() *CertManager { return s.cm }

// IsRunning reports whether Start has been called and Stop has not.
func (s *Server) IsRunning() bool { return s.running.Load() }

// ListeningPort is 0 until Start has bound a socket.
func (s *Server) ListeningPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// Start binds the listener, registers the Display service, and serves
// until Stop is called or rotateRequest fires.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running.Load() {
		return errors.New("einkgrpc: server already running")
	}

	tlsCer, err := tls.X509KeyPair(s.cm.ServerCertPEM(), s.cm.ServerKeyPEM())
	if err != nil {
		return fmt.Errorf("einkgrpc: load server keypair: %w", err)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCer},
		MinVersion:   tls.VersionTLS12,
	}
	if s.cfg.EnableMTLS {
		pool, err := s.cm.ClientCAPool()
		if err != nil {
			return err
		}
		tlsCfg.ClientCAs = pool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	auth := NewTokenAuth(s.cm)
	srv := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsCfg)),
		grpc.MaxRecvMsgSize(s.cfg.MaxRecvBytes),
		grpc.UnaryInterceptor(auth.Unary),
		grpc.StreamInterceptor(auth.Stream),
	)
	displaySvc := NewDisplayService(
		s.cfg.Display, s.cm, s.cfg.ServerVersion,
		func() {
			select {
			case s.rotateReq <- struct{}{}:
			default:
			}
			_ = s.Stop()
		},
	)
	pb.RegisterDisplayServer(srv, displaySvc)

	addr := fmt.Sprintf(":%d", s.cfg.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("einkgrpc: listen %s: %w", addr, err)
	}
	if tcp, ok := lis.Addr().(*net.TCPAddr); ok {
		s.port = tcp.Port
	}

	s.srv = srv
	s.lis = lis
	s.running.Store(true)

	if s.cfg.EnableMDNS {
		s.mdns = NewMdnsRegistrar()
		isColor := false
		if s.cfg.Display != nil {
			if info, err := s.cfg.Display.Info(); err == nil {
				isColor = info.IsColor()
			}
		}
		_ = s.mdns.Register(s.port, isColor, s.cm.Fingerprint(), s.cfg.ServerVersion)
	}

	go func() {
		_ = srv.Serve(lis)
	}()
	return nil
}

// Stop performs a graceful shutdown with a 5 s deadline.
func (s *Server) Stop() error {
	s.mu.Lock()
	srv := s.srv
	mdns := s.mdns
	s.srv = nil
	s.mdns = nil
	s.mu.Unlock()
	if !s.running.Load() {
		return nil
	}
	s.running.Store(false)
	if mdns != nil {
		mdns.Unregister()
	}
	if srv != nil {
		stopped := make(chan struct{})
		go func() {
			srv.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(5 * time.Second):
			srv.Stop()
		}
	}
	return nil
}

// ConnectURI returns the operator-friendly tableaux:// URL embedding
// host, port, token and short fingerprint, for QR-code generation.
func (s *Server) ConnectURI(host string) string {
	if host == "" {
		host = "127.0.0.1"
	}
	v := url.Values{}
	v.Set("token", s.cm.Token())
	v.Set("fp", truncatedFingerprint(s.cm.Fingerprint()))
	return fmt.Sprintf("tableaux://%s:%d?%s", host, s.ListeningPort(), v.Encode())
}
