package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"client/internal/config"
	"google.golang.org/grpc/credentials"
)

func LoadClientTLSCredentials(cfg *config.Config) (credentials.TransportCredentials, error) {
	clientPair, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf(
			"security: could not load client key pair (%s, %s): %w",
			cfg.TLSCertFile, cfg.TLSKeyFile, err,
		)
	}

	caPem, err := os.ReadFile(cfg.TLSCAFile)
	if err != nil {
		return nil, fmt.Errorf(
			"security: could not read CA certificate file (%s): %w",
			cfg.TLSCAFile, err,
		)
	}
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(caPem); !ok {
		return nil, fmt.Errorf(
			"security: failed to append CA certificate(s) from %s",
			cfg.TLSCAFile,
		)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientPair},
		RootCAs:      rootCAs,
		MinVersion:   tls.VersionTLS12,
	}

	return credentials.NewTLS(tlsConfig), nil
}
