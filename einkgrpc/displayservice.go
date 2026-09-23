package einkgrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/go-sicp/tableaux-eink-go/einklib"
	pb "github.com/go-sicp/tableaux-eink-go/einkproto"
)

// DisplayService implements pb.DisplayServer on top of einklib.Display.
type DisplayService struct {
	pb.UnimplementedDisplayServer

	display       einklib.Display
	cm            *CertManager
	serverVersion string
	startedAt     time.Time
	rotateRequest func()

	mu        sync.Mutex
	lastError string
}

// NewDisplayService wires the gRPC implementation to a Display backend and a
// CertManager. rotateRequest is invoked from RotateNow to ask the hosting
// Server to stop and restart with fresh material.
func NewDisplayService(d einklib.Display, cm *CertManager, version string, rotateRequest func()) *DisplayService {
	return &DisplayService{
		display:       d,
		cm:            cm,
		serverVersion: version,
		startedAt:     time.Now(),
		rotateRequest: rotateRequest,
	}
}

func (s *DisplayService) GetInfo(ctx context.Context, _ *emptypb.Empty) (*pb.DisplayInfo, error) {
	if s.display == nil {
		return nil, status.Error(codes.FailedPrecondition, "display not opened")
	}
	info, err := s.display.Info()
	if err != nil {
		s.recordErr(err)
		return nil, status.Errorf(codes.Internal, "display.info: %v", err)
	}
	return &pb.DisplayInfo{
		Width:         info.Width(),
		Height:        info.Height(),
		VirtualWidth:  info.VirtualWidth(),
		VirtualHeight: info.VirtualHeight(),
		Color:         info.IsColor(),
		AppVersion:    s.serverVersion,
	}, nil
}

func (s *DisplayService) Health(_ context.Context, _ *emptypb.Empty) (*pb.HealthResponse, error) {
	s.mu.Lock()
	last := s.lastError
	s.mu.Unlock()
	return &pb.HealthResponse{
		DeviceOpen:    s.display != nil,
		LastError:     last,
		ServerVersion: s.serverVersion,
		UptimeSeconds: int64(time.Since(s.startedAt).Seconds()),
	}, nil
}

func (s *DisplayService) Fill(ctx context.Context, req *pb.FillRequest) (*pb.RefreshResponse, error) {
	if s.display == nil {
		return nil, status.Error(codes.FailedPrecondition, "display not opened")
	}
	t0 := time.Now()
	frame, err := s.display.AcquireFrame()
	if err != nil {
		s.recordErr(err)
		return nil, status.Errorf(codes.Internal, "acquireFrame: %v", err)
	}
	einklib.FillSolid(frame, req.Argb)
	if err := frame.Commit(translateMode(req.Mode), nil); err != nil {
		s.recordErr(err)
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}
	return &pb.RefreshResponse{ElapsedMs: time.Since(t0).Milliseconds()}, nil
}

func (s *DisplayService) Refresh(stream pb.Display_RefreshServer) error {
	if s.display == nil {
		return status.Error(codes.FailedPrecondition, "display not opened")
	}
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	header, ok := first.GetPayload().(*pb.RefreshChunk_Header)
	if !ok || header.Header == nil {
		return status.Error(codes.InvalidArgument, "first chunk must be RefreshHeader")
	}
	t0 := time.Now()
	frame, err := s.display.AcquireFrame()
	if err != nil {
		s.recordErr(err)
		return status.Errorf(codes.Internal, "acquireFrame: %v", err)
	}
	pixels := frame.Pixels()
	written := int64(0)
	totalBytes := header.Header.TotalBytes
	if totalBytes < 0 || totalBytes > int64(len(pixels)) {
		_ = frame.Cancel()
		return status.Errorf(codes.InvalidArgument, "total_bytes %d out of range", totalBytes)
	}

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_ = frame.Cancel()
			return err
		}
		data, ok := chunk.GetPayload().(*pb.RefreshChunk_Data)
		if !ok {
			_ = frame.Cancel()
			return status.Error(codes.InvalidArgument, "expected data chunk after header")
		}
		end := written + int64(len(data.Data))
		if end > totalBytes {
			_ = frame.Cancel()
			return status.Error(codes.InvalidArgument, "chunk overflows total_bytes")
		}
		copy(pixels[written:end], data.Data)
		written = end
	}
	if written != totalBytes {
		_ = frame.Cancel()
		return status.Errorf(codes.InvalidArgument, "got %d bytes, want %d", written, totalBytes)
	}

	var region *einklib.Region
	if r := header.Header.Region; r != nil {
		rr := einklib.NewRegion(r.X1, r.Y1, r.X2, r.Y2)
		region = &rr
	}
	if err := frame.Commit(translateMode(header.Header.Mode), region); err != nil {
		s.recordErr(err)
		return status.Errorf(codes.Internal, "commit: %v", err)
	}
	return stream.SendAndClose(&pb.RefreshResponse{ElapsedMs: time.Since(t0).Milliseconds()})
}

func (s *DisplayService) SetOverlay(_ context.Context, req *pb.SetOverlayRequest) (*emptypb.Empty, error) {
	if s.display == nil {
		return nil, status.Error(codes.FailedPrecondition, "display not opened")
	}
	if err := s.display.SetOverlay(req.Enabled); err != nil {
		s.recordErr(err)
		return nil, status.Errorf(codes.Internal, "setOverlay: %v", err)
	}
	return &emptypb.Empty{}, nil
}

func (s *DisplayService) Rotate(_ context.Context, _ *emptypb.Empty) (*pb.RotateResponse, error) {
	if err := s.cm.Rotate(); err != nil {
		return nil, status.Errorf(codes.Internal, "rotate: %v", err)
	}
	at := time.Now().UnixMilli()
	if s.rotateRequest != nil {
		go s.rotateRequest()
	}
	return &pb.RotateResponse{ScheduledAtMs: at}, nil
}

func (s *DisplayService) ReissueClientCert(_ context.Context, _ *emptypb.Empty) (*pb.ReissueClientCertResponse, error) {
	if err := s.cm.ReissueClientCert(); err != nil {
		return nil, status.Errorf(codes.Internal, "reissue: %v", err)
	}
	resp := &pb.ReissueClientCertResponse{
		ClientCertPem:         string(s.cm.ClientCertPEM()),
		ClientKeyPem:          string(s.cm.ClientKeyPEM()),
		ClientCertFingerprint: s.cm.ClientFingerprint(),
		ServerRestartInMs:     2000,
	}
	if s.rotateRequest != nil {
		go func() {
			time.Sleep(2 * time.Second)
			s.rotateRequest()
		}()
	}
	return resp, nil
}

func (s *DisplayService) GetMetrics(_ context.Context, _ *emptypb.Empty) (*pb.GetMetricsResponse, error) {
	out := fmt.Sprintf(
		"# HELP tableaux_uptime_seconds Server uptime in seconds.\n"+
			"# TYPE tableaux_uptime_seconds counter\n"+
			"tableaux_uptime_seconds %d\n"+
			"# HELP tableaux_device_open Whether the EBC device is opened.\n"+
			"# TYPE tableaux_device_open gauge\n"+
			"tableaux_device_open %d\n",
		int64(time.Since(s.startedAt).Seconds()),
		boolToInt(s.display != nil),
	)
	return &pb.GetMetricsResponse{PromText: out}, nil
}

func (s *DisplayService) recordErr(err error) {
	s.mu.Lock()
	s.lastError = err.Error()
	s.mu.Unlock()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func translateMode(m pb.WaveformMode) einklib.WaveformMode {
	switch m {
	case pb.WaveformMode_WAVEFORM_MODE_INIT:
		return einklib.WaveformInit
	case pb.WaveformMode_WAVEFORM_MODE_DIRECT_UPDATE:
		return einklib.WaveformDirectUpdate
	case pb.WaveformMode_WAVEFORM_MODE_GC16:
		return einklib.WaveformGC16
	case pb.WaveformMode_WAVEFORM_MODE_GCC16:
		return einklib.WaveformGCC16
	case pb.WaveformMode_WAVEFORM_MODE_ANIMATION:
		return einklib.WaveformAnimation
	case pb.WaveformMode_WAVEFORM_MODE_PARTIAL:
		return einklib.WaveformPartial
	case pb.WaveformMode_WAVEFORM_MODE_FULL:
		return einklib.WaveformFull
	case pb.WaveformMode_WAVEFORM_MODE_AUTO:
		return einklib.WaveformAuto
	case pb.WaveformMode_WAVEFORM_MODE_RESET:
		return einklib.WaveformReset
	case pb.WaveformMode_WAVEFORM_MODE_BLACK_WHITE:
		return einklib.WaveformBlackWhite
	case pb.WaveformMode_WAVEFORM_MODE_TRANSPARENT:
		return einklib.WaveformTransparent
	case pb.WaveformMode_WAVEFORM_MODE_REGAL:
		return einklib.WaveformRegal
	case pb.WaveformMode_WAVEFORM_MODE_SPECTRA_FULL_GC16:
		return einklib.WaveformSpectraFullGC16
	case pb.WaveformMode_WAVEFORM_MODE_SPECTRA_GLARE_REDUCED:
		return einklib.WaveformSpectraGlareReduced
	case pb.WaveformMode_WAVEFORM_MODE_SPECTRA_GHOST_REDUCED:
		return einklib.WaveformSpectraGhostReduced
	case pb.WaveformMode_WAVEFORM_MODE_SPECTRA_FULL_GCC16:
		return einklib.WaveformSpectraFullGCC16
	}
	return einklib.WaveformAuto
}

// Compile-time interface check.
var _ pb.DisplayServer = (*DisplayService)(nil)
