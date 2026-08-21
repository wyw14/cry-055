package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Address         string
	DatabaseURL     string
	AttachmentRoot  string
	ShutdownTimeout time.Duration
	RequestTimeout  time.Duration
	MaxUploadBytes  int64
}

func Load() (Config, error) {
	cfg := Config{
		Address:         env("APP_ADDR", ":8080"),
		DatabaseURL:     env("DATABASE_URL", "postgres://calibration:calibration@localhost:5432/calibration?sslmode=disable"),
		AttachmentRoot:  env("ATTACHMENT_ROOT", "./runtime/attachments"),
		ShutdownTimeout: 10 * time.Second,
		RequestTimeout:  5 * time.Second,
		MaxUploadBytes:  10 << 20,
	}
	var err error
	if cfg.ShutdownTimeout, err = duration("SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.RequestTimeout, err = duration("REQUEST_TIMEOUT", cfg.RequestTimeout); err != nil {
		return Config{}, err
	}
	if raw := os.Getenv("MAX_UPLOAD_BYTES"); raw != "" {
		cfg.MaxUploadBytes, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || cfg.MaxUploadBytes <= 0 {
			return Config{}, errors.New("MAX_UPLOAD_BYTES must be a positive integer")
		}
	}
	return cfg, nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, errors.New(name + " must be a positive duration")
	}
	return value, nil
}
