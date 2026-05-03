package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func Load(configPath string) (*Config, string, error) {
	v := viper.New()

	setDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".config", "nolife"))
		}
	}

	usedConfigFile := ""
	if err := v.ReadInConfig(); err != nil {
		if configPath != "" {
			return nil, "", fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		usedConfigFile = v.ConfigFileUsed()
	}

	v.SetEnvPrefix("ANIME_DL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	bindEnvVars(v)

	cfg := DefaultConfig()
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
		Result:           cfg,
		WeaklyTypedInput: true,
		TagName:          "mapstructure",
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(v.AllSettings()); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, usedConfigFile, nil
}

func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()

	v.SetDefault("general.output_dir", defaults.General.OutputDir)
	v.SetDefault("general.log_level", defaults.General.LogLevel)
	v.SetDefault("general.max_concurrent", defaults.General.MaxConcurrent)

	v.SetDefault("browser.headless", defaults.Browser.Headless)
	v.SetDefault("browser.disable_gpu", defaults.Browser.DisableGPU)
	v.SetDefault("browser.disable_images", defaults.Browser.DisableImages)
	v.SetDefault("browser.user_agent", defaults.Browser.UserAgent)
	v.SetDefault("browser.timeout.navigation", defaults.Browser.Timeout.Navigation)
	v.SetDefault("browser.timeout.page_load", defaults.Browser.Timeout.PageLoad)
	v.SetDefault("browser.timeout.element_wait", defaults.Browser.Timeout.ElementWait)

	v.SetDefault("download.skip_existing", defaults.Download.SkipExisting)
	v.SetDefault("download.preferred_quality", defaults.Download.PreferredQuality)
}

func bindEnvVars(v *viper.Viper) {
	v.BindEnv("general.output_dir", "ANIME_DL_OUTPUT")
	v.BindEnv("general.log_level", "ANIME_DL_LOG_LEVEL")
	v.BindEnv("browser.headless", "ANIME_DL_HEADLESS")
}

func GetConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./.config/nolife"
	}
	return filepath.Join(homeDir, ".config", "nolife")
}

func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.yaml")
}

func SaveConfig(cfg *Config, path string) error {
	v := viper.New()

	v.Set("general", cfg.General)
	v.Set("browser", cfg.Browser)
	v.Set("download", cfg.Download)
	v.Set("sites", cfg.Sites)
	v.Set("hosts", cfg.Hosts)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return v.WriteConfigAs(path)
}
