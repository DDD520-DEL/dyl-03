package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds runtime settings for the shadow gateway.
type Config struct {
	ListenAddr        string        `json:"listen_addr"`
	WALDir            string        `json:"wal_dir"`
	WALSegmentBytes   int           `json:"wal_segment_bytes"`
	SessionTimeout    time.Duration `json:"session_timeout"`
	CommandTimeout    time.Duration `json:"command_timeout"`
	RetentionWindow   time.Duration `json:"retention_window"`
	TelemetryMaxSeq   int64         `json:"telemetry_max_seq"`
}

// Default returns the standard configuration.
func Default() Config {
	return Config{
		ListenAddr:      ":9090",
		WALDir:          "./data/wal",
		WALSegmentBytes: 4 << 20,
		SessionTimeout:  30 * time.Second,
		CommandTimeout:  60 * time.Second,
		RetentionWindow: 24 * time.Hour,
		TelemetryMaxSeq: 1 << 20,
	}
}

// Load reads a JSON config file with defaults for zero fields.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
