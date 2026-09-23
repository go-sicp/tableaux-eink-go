package einkgrpc_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/go-sicp/tableaux-eink-go/einkgrpc"
)

func TestTokenAuthAcceptsOperatorToken(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	ta := einkgrpc.NewTokenAuth(cm)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.New(map[string]string{"authorization": "Bearer " + cm.Token()}),
	)
	called := false
	_, err = ta.Unary(ctx, nil, &grpc.UnaryServerInfo{}, func(_ context.Context, _ any) (any, error) {
		called = true
		return "ok", nil
	})
	if err != nil || !called {
		t.Fatalf("operator token rejected: err=%v called=%v", err, called)
	}
}

func TestTokenAuthAcceptsAdminToken(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	ta := einkgrpc.NewTokenAuth(cm)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.New(map[string]string{"authorization": "Bearer " + cm.AdminToken()}),
	)
	_, err = ta.Unary(ctx, nil, &grpc.UnaryServerInfo{}, func(_ context.Context, _ any) (any, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("admin token rejected: %v", err)
	}
}

func TestTokenAuthRejectsWrongToken(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	ta := einkgrpc.NewTokenAuth(cm)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.New(map[string]string{"authorization": "Bearer not-a-real-token"}),
	)
	_, err = ta.Unary(ctx, nil, &grpc.UnaryServerInfo{}, func(_ context.Context, _ any) (any, error) {
		t.Fatal("handler should not run on auth failure")
		return nil, nil
	})
	if !isCode(err, codes.Unauthenticated) {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestTokenAuthRejectsMissingHeader(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	ta := einkgrpc.NewTokenAuth(cm)

	_, err = ta.Unary(context.Background(), nil, &grpc.UnaryServerInfo{}, func(_ context.Context, _ any) (any, error) {
		t.Fatal("handler should not run")
		return nil, nil
	})
	if !isCode(err, codes.Unauthenticated) {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func TestTokenAuthRejectsNonBearerScheme(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	ta := einkgrpc.NewTokenAuth(cm)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.New(map[string]string{"authorization": "Basic " + cm.Token()}),
	)
	_, err = ta.Unary(ctx, nil, &grpc.UnaryServerInfo{}, func(_ context.Context, _ any) (any, error) {
		return nil, nil
	})
	if !isCode(err, codes.Unauthenticated) {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func isCode(err error, want codes.Code) bool {
	if err == nil {
		return false
	}
	st, ok := status.FromError(err)
	if !ok {
		return errors.Is(err, errors.New(want.String()))
	}
	return st.Code() == want
}
