package main

import (
	"io"
	"os"
	"gopkg.in/yaml.v3"
)

type MaintenancePeriodConfig struct {
	Repeat        string `yaml:"repeat"`
	StartingDay   string `yaml:"starting_day"`
	StartingTime  string `yaml:"starting_time"`
	Duration      string `yaml:"duration"` // e.g. "@1m", "@5h", "@1d" or "30s"
}

type ServiceConfig struct {
	FQDN     string `yaml:"fqdn"`
	TimeoutS int    `yaml:"timeout_s"`
	Retries  int    `yaml:"retries"`
	Method   string `yaml:"method"`
	Interval string `yaml:"interval"` // e.g. "@1m", "@5h", "@1d" or "30s"
	MaintenancePeriod *MaintenancePeriodConfig `yaml:"maintenance_period,omitempty"`
	Channels []string `yaml:"channels,omitempty"`
}

type ChannelConfig struct {
	Type                string `yaml:"type"`
	SuccessNotification string `yaml:"success_notification"`
	ErrorNotification   string `yaml:"error_notification"`
}

type Config struct {
	Services map[string]ServiceConfig `yaml:"services"`
	Channels map[string]ChannelConfig `yaml:"channels,omitempty"`
}

func loadConfig(path string) (Config, error) {
	debugf("loading config from %s", path)
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
