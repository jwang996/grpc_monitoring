package config

import "os"

type Config struct {
	GRPCServerAddress string
	MetricsPort       string
	TLSCertFile       string
	TLSKeyFile        string
	TLSCAFile         string
}

func LoadConfig() *Config {
	return &Config{
		GRPCServerAddress: getEnv("GRPC_SERVER_ADDRESS", "server:50059"),
		MetricsPort:       getEnv("METRICS_PORT", "2024"),
		TLSCertFile:       getEnv("TLS_CERT_FILE", "certs/client.crt.pem"),
		TLSKeyFile:        getEnv("TLS_KEY_FILE", "certs/client.key.pem"),
		TLSCAFile:         getEnv("TLS_CA_FILE", "certs/ca.crt.pem"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
