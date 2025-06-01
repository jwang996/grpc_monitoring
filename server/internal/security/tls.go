package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
	"server/internal/config"
)

func LoadTLSCredentials(cfg *config.Config) (credentials.TransportCredentials, error) {
	serverPair, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("security: could not load server key pair (%s, %s): %w",
			cfg.TLSCertFile, cfg.TLSKeyFile, err)
	}

	caPem, err := os.ReadFile(cfg.TLSCAFile)
	if err != nil {
		return nil, fmt.Errorf("security: could not read CA certificate file (%s): %w",
			cfg.TLSCAFile, err)
	}
	clientCertPool := x509.NewCertPool()
	if ok := clientCertPool.AppendCertsFromPEM(caPem); !ok {
		return nil, fmt.Errorf("security: failed to append CA certificate(s) from %s",
			cfg.TLSCAFile)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverPair},

		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCertPool,

		MinVersion: tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsConfig), nil
}
