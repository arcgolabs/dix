// Package config loads configuration for the backend example.
package config

// AppConfig describes the backend example configuration.
type AppConfig struct {
	Server struct {
		Port int `koanf:"port"`
	} `koanf:"server"`
	DB struct {
		DSN string `koanf:"dsn"`
	} `koanf:"db"`
}
