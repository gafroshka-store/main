package app

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CfgDB           ConfigDB      `yaml:"db"`
	CfgES           ConfigES      `yaml:"es"`
	ETLTimeout      time.Duration `yaml:"etl_search_timeout"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	Secret          string        `yaml:"secret"`
	ServerPort      string        `yaml:"srv_port"`
	SessionDuration time.Duration `yaml:"session_duration"`
}

type ConfigDB struct {
	Login    string `yaml:"login"`
	Password string `yaml:"password"`
	Port     uint   `yaml:"port"`
	Database string `yaml:"database"`
	Host     string `yaml:"host"`
}

type ConfigES struct {
	Index string `yaml:"index"`
}

func NewConfig(configPath string) (*Config, error) {
	cfg, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var c Config
	err = yaml.Unmarshal(cfg, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
