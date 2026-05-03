package browser

import (
	"time"

	"nolife-cli/internal/config"
)

type Options struct {
	Headless          bool
	DisableGPU        bool
	DisableImages     bool
	UserAgent         string
	NavigationTimeout time.Duration
	PageLoadTimeout   time.Duration
	ElementWaitTime   time.Duration
}

func NewOptions(cfg *config.BrowserConfig) *Options {
	return &Options{
		Headless:          cfg.Headless,
		DisableGPU:        cfg.DisableGPU,
		DisableImages:     cfg.DisableImages,
		UserAgent:         cfg.UserAgent,
		NavigationTimeout: cfg.Timeout.Navigation,
		PageLoadTimeout:   cfg.Timeout.PageLoad,
		ElementWaitTime:   cfg.Timeout.ElementWait,
	}
}

func DefaultOptions() *Options {
	return &Options{
		Headless:          true,
		DisableGPU:        true,
		DisableImages:     true,
		UserAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		NavigationTimeout: 30 * time.Second,
		PageLoadTimeout:   15 * time.Second,
		ElementWaitTime:   10 * time.Second,
	}
}
