package config

import "time"

type Config struct {
	General  GeneralConfig         `yaml:"general" mapstructure:"general"`
	Browser  BrowserConfig         `yaml:"browser" mapstructure:"browser"`
	Download DownloadConfig        `yaml:"download" mapstructure:"download"`
	Sites    map[string]SiteConfig `yaml:"sites" mapstructure:"sites"`
	Hosts    map[string]HostConfig `yaml:"hosts" mapstructure:"hosts"`
}

type GeneralConfig struct {
	OutputDir     string `yaml:"output_dir" mapstructure:"output_dir"`
	LogLevel      string `yaml:"log_level" mapstructure:"log_level"`
	MaxConcurrent int    `yaml:"max_concurrent" mapstructure:"max_concurrent"`
}

type BrowserConfig struct {
	Headless      bool          `yaml:"headless" mapstructure:"headless"`
	DisableGPU    bool          `yaml:"disable_gpu" mapstructure:"disable_gpu"`
	DisableImages bool          `yaml:"disable_images" mapstructure:"disable_images"`
	UserAgent     string        `yaml:"user_agent" mapstructure:"user_agent"`
	Timeout       TimeoutConfig `yaml:"timeout" mapstructure:"timeout"`
}

type TimeoutConfig struct {
	Navigation  time.Duration `yaml:"navigation" mapstructure:"navigation"`
	PageLoad    time.Duration `yaml:"page_load" mapstructure:"page_load"`
	ElementWait time.Duration `yaml:"element_wait" mapstructure:"element_wait"`
}

type DownloadConfig struct {
	SkipExisting     bool   `yaml:"skip_existing" mapstructure:"skip_existing"`
	PreferredQuality string `yaml:"preferred_quality" mapstructure:"preferred_quality"`
}

type SiteConfig struct {
	Domains        []string `yaml:"domains" mapstructure:"domains"`
	EpisodePattern string   `yaml:"episode_pattern" mapstructure:"episode_pattern"`
}

type HostConfig struct {
	Domains   []string `yaml:"domains" mapstructure:"domains"`
	Selectors []string `yaml:"selectors" mapstructure:"selectors"`
}
