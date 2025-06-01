package config

import (
	"os"
)

type Config struct {
	GRPCPort              string
	MetricsPort           string
	TLSCertFile           string
	TLSKeyFile            string
	TLSCAFile             string
	OTLPCollectorEndpoint string
}

func LoadConfig() *Config {
	return &Config{
		GRPCPort:              getEnv("GRPC_PORT", "50051"),
		MetricsPort:           getEnv("METRICS_PORT", "2025"),
		TLSCertFile:           getEnv("TLS_CERT_FILE", "certs/server.crt"),
		TLSKeyFile:            getEnv("TLS_KEY_FILE", "certs/server.key"),
		TLSCAFile:             getEnv("TLS_CA_FILE", "certs/ca.crt"),
		OTLPCollectorEndpoint: getEnv("OTLP_COLLECTOR_ENDPOINT", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
