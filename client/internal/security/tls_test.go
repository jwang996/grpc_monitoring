package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"google.golang.org/grpc/credentials"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"client/internal/config"
)

func generateTestCA(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate CA private key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA Org"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)})

	return certPEM, keyPEM
}

func generateTestServerCert(t *testing.T, caCert *x509.Certificate, caKey *rsa.PrivateKey) (serverCertPEM, serverKeyPEM []byte) {
	t.Helper()

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate server private key: %v", err)
	}

	serial, err := rand.Int(rand.Reader, big.NewInt(100000))
	if err != nil {
		t.Fatalf("failed to generate serial number: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Test Server Org"},
			CommonName:   "localhost",
		},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(12 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
		IPAddresses: nil,
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create server certificate: %v", err)
	}

	serverCertPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})

	serverKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})

	return serverCertPEM, serverKeyPEM
}

func TestLoadTLSCredentials_Success(t *testing.T) {
	caCertPEM, caKeyPEM := generateTestCA(t)

	caBlock, _ := pem.Decode(caCertPEM)
	if caBlock == nil {
		t.Fatal("failed to decode CA certificate PEM")
	}
	caCertParsed, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse CA certificate: %v", err)
	}

	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil {
		t.Fatal("failed to decode CA key PEM")
	}
	caKeyParsed, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse CA private key: %v", err)
	}

	serverCertPEM, serverKeyPEM := generateTestServerCert(t, caCertParsed, caKeyParsed)

	tmpDir := t.TempDir()
	caPath := filepath.Join(tmpDir, "ca.crt.pem")
	srvCertPath := filepath.Join(tmpDir, "server.crt.pem")
	srvKeyPath := filepath.Join(tmpDir, "server.key.pem")

	if err := os.WriteFile(caPath, caCertPEM, 0o644); err != nil {
		t.Fatalf("failed to write CA cert file: %v", err)
	}
	if err := os.WriteFile(srvCertPath, serverCertPEM, 0o644); err != nil {
		t.Fatalf("failed to write server cert file: %v", err)
	}
	if err := os.WriteFile(srvKeyPath, serverKeyPEM, 0o600); err != nil {
		t.Fatalf("failed to write server key file: %v", err)
	}

	cfg := &config.Config{
		TLSCertFile: srvCertPath,
		TLSKeyFile:  srvKeyPath,
		TLSCAFile:   caPath,
	}

	creds, err := LoadClientTLSCredentials(cfg)
	if err != nil {
		t.Fatalf("expected no error loading TLS credentials, got: %v", err)
	}

	if creds == nil {
		t.Fatal("expected non-nil credentials")
	}

	tlsCreds, ok := creds.(credentials.TransportCredentials)
	if !ok {
		t.Fatalf("expected TransportCredentials, got %T", creds)
	}

	configGetter := tlsCreds.Info()
	if configGetter.SecurityProtocol != "tls" {
		t.Errorf("expected SecurityProtocol \"tls\", got %q", configGetter.SecurityProtocol)
	}
}

func TestLoadTLSCredentials_Failure(t *testing.T) {
	cfg1 := &config.Config{
		TLSCertFile: "/nonexistent/server.crt.pem",
		TLSKeyFile:  "/nonexistent/server.key.pem",
		TLSCAFile:   "/nonexistent/ca.crt.pem",
	}
	if _, err := LoadClientTLSCredentials(cfg1); err == nil {
		t.Error("expected error when server cert/key paths do not exist, got nil")
	}

	tmpDir := t.TempDir()
	caPath := filepath.Join(tmpDir, "ca.crt.pem")
	if err := os.WriteFile(caPath, []byte("not-a-valid-pem"), 0o644); err != nil {
		t.Fatalf("failed to write bad CA file: %v", err)
	}

	cfg2 := &config.Config{
		TLSCertFile: caPath,
		TLSKeyFile:  caPath,
		TLSCAFile:   caPath,
	}
	_, err := LoadClientTLSCredentials(cfg2)
	if err == nil {
		t.Error("expected error when CA PEM is invalid, got nil")
	}
}
