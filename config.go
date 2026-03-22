package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr   string             `yaml:"listen_addr"`
	ClickHouse   ClickHouseConfig   `yaml:"clickhouse"`
	SQLite       SQLiteConfig       `yaml:"sqlite"`
	Evaluation   EvaluationConfig   `yaml:"evaluation"`
	Notification NotificationConfig `yaml:"notifications"`
	Log          LogConfig          `yaml:"log"`
}

type ClickHouseConfig struct {
	DSN          string `yaml:"dsn"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type EvaluationConfig struct {
	DefaultInterval Duration `yaml:"default_interval"`
	QueryTimeout    Duration `yaml:"query_timeout"`
	MaxConcurrent   int      `yaml:"max_concurrent"`
}

type NotificationConfig struct {
	RepeatInterval Duration `yaml:"repeat_interval"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Duration wraps time.Duration for YAML unmarshaling.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = dur
	return nil
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

func DefaultConfig() Config {
	return Config{
		ListenAddr: ":8080",
		ClickHouse: ClickHouseConfig{
			DSN:          "clickhouse://default:@localhost:9000/default",
			MaxOpenConns: 5,
		},
		SQLite: SQLiteConfig{
			Path: "./alerting.db",
		},
		Evaluation: EvaluationConfig{
			DefaultInterval: Duration{60 * time.Second},
			QueryTimeout:    Duration{30 * time.Second},
			MaxConcurrent:   10,
		},
		Notification: NotificationConfig{
			RepeatInterval: Duration{4 * time.Hour},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("reading config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}
