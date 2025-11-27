package config

import (
	"fmt"
	"os"
	"time"

	lilium "github.com/spyder01/lilium-go"
	"gopkg.in/yaml.v3"
)

type LiliumJobRunnerConfig struct {
	Name           string            `yaml:"name"`     // job name
	Enabled        bool              `yaml:"enabled"`  // can disable via config
	Interval       time.Duration     `yaml:"interval"` // e.g. "10s", "1m"
	Repeat         bool              `yaml:"repeat"`   // false = run once
	Task           lilium.LiliumTask `yaml:"-"`        // injected in code
	Description    string            `yaml:"description,omitempty"`
	Retries        int               `yaml:"retries,omitempty"`         // default 0
	InitialBackoff time.Duration     `yaml:"initial_backoff,omitempty"` // default 1s
	MaxBackoff     time.Duration     `yaml:"max_backoff,omitempty"`     // default 30s
	Timeout        time.Duration     `yaml:"timeout,omitempty"`         // per-run timeout
	MaxConcurrency int               `yaml:"max_concurrency,omitempty"` // default 1
}

type LiliumJobs struct {
	Jobs []*LiliumJobRunnerConfig `yaml:"jobs"`
}

func Load(path string) (*LiliumJobs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg LiliumJobs
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &cfg, nil
}
