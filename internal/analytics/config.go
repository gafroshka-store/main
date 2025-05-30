package analytics

import (
	"gafroshka-main/internal/app"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	CfgDB        app.ConfigDB `yaml:"db"`
	MaxOpenConns int          `yaml:"max_open_conns"`
}

func NewConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
