package einkgrpc_test

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-sicp/tableaux-eink-go/einkgrpc"
)

func TestCertManagerInitGeneratesAllArtifacts(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatalf("NewCertManager: %v", err)
	}
	for _, name := range []string{"cert.pem", "key.pem", "client-cert.pem", "client-key.pem", "token.txt", "admin-token.txt"} {
		p := filepath.Join(dir, "tls", name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s on disk: %v", name, err)
		}
	}
	if cm.Token() == "" || cm.AdminToken() == "" {
		t.Errorf("tokens empty")
	}
	if cm.Token() == cm.AdminToken() {
		t.Errorf("operator and admin token must differ")
	}
	if cm.Fingerprint() == "" {
		t.Errorf("server fingerprint empty")
	}
	if cm.ClientFingerprint() == "" {
		t.Errorf("client fingerprint empty")
	}
}

func TestCertManagerReusesArtifactsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	cm1, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	tok := cm1.Token()
	fp := cm1.Fingerprint()

	cm2, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cm2.Token() != tok {
		t.Errorf("second instance must reuse the same token")
	}
	if cm2.Fingerprint() != fp {
		t.Errorf("second instance must reuse the same fingerprint")
	}
}

func TestCertManagerRotateChangesEverything(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	prevTok, prevAdm, prevFp, prevCfp := cm.Token(), cm.AdminToken(), cm.Fingerprint(), cm.ClientFingerprint()
	if err := cm.Rotate(); err != nil {
		t.Fatal(err)
	}
	if cm.Token() == prevTok || cm.AdminToken() == prevAdm {
		t.Errorf("Rotate must change tokens")
	}
	if cm.Fingerprint() == prevFp || cm.ClientFingerprint() == prevCfp {
		t.Errorf("Rotate must change cert fingerprints")
	}
}

func TestCertManagerReissueClientOnly(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	prevTok, prevSrvFp, prevCfp := cm.Token(), cm.Fingerprint(), cm.ClientFingerprint()
	if err := cm.ReissueClientCert(); err != nil {
		t.Fatal(err)
	}
	if cm.Token() != prevTok {
		t.Errorf("ReissueClientCert must NOT touch the operator token")
	}
	if cm.Fingerprint() != prevSrvFp {
		t.Errorf("ReissueClientCert must NOT touch the server cert")
	}
	if cm.ClientFingerprint() == prevCfp {
		t.Errorf("ReissueClientCert must change the client fingerprint")
	}
}

func TestServerCertParsesAsX509(t *testing.T) {
	dir := t.TempDir()
	cm, err := einkgrpc.NewCertManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(cm.ServerCertPEM())
	if block == nil {
		t.Fatal("server cert PEM did not decode")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Errorf("parse server cert: %v", err)
	}
}
