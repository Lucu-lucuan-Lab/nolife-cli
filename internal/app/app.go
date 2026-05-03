package app

import (
	"os"
	"strings"

	"nolife-cli/internal/browser"
	"nolife-cli/internal/config"
	"nolife-cli/internal/hosts"
	"nolife-cli/internal/scraper"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type App struct {
	Cfg            *config.Config
	ConfigPath     string
	BrowserManager *browser.Manager
}

type DownloadOptions struct {
	Input        string
	SeriesTitle  string
	OutputDir    string
	Episodes     string
	Quality      string
	Concurrent   int
	SkipExisting bool
	IsSearch     bool
}

func New(cfg *config.Config, configPath string, headless, noHeadless bool) *App {
	configureLogger(cfg.General.LogLevel)

	browserOpts := browser.NewOptions(&cfg.Browser)
	if noHeadless {
		browserOpts.Headless = false
	} else if headless {
		browserOpts.Headless = true
	}

	manager, err := browser.NewManager(browserOpts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize browser manager")
	}

	for name, siteCfg := range cfg.Sites {
		if name == "oploverz" {
			scraper.Register(scraper.NewOploverz(&siteCfg, manager))
		}
	}

	hostOrder := []string{"filedon", "pixeldrain", "acefile", "mixdrop", "googledrive"}
	for _, name := range hostOrder {
		hostCfg, exists := cfg.Hosts[name]
		if !exists {
			continue
		}
		switch name {
		case "googledrive":
			hosts.Register(hosts.NewGoogleDrive(&hostCfg, manager))
		case "acefile":
			hosts.Register(hosts.NewAcefile(&hostCfg, manager))
		case "mixdrop":
			hosts.Register(hosts.NewMixDrop(&hostCfg, manager))
		case "filedon":
			hosts.Register(hosts.NewFiledon(&hostCfg, manager))
		case "pixeldrain":
			hosts.Register(hosts.NewPixeldrain(&hostCfg, manager))
		}
	}

	return &App{
		Cfg:            cfg,
		ConfigPath:     configPath,
		BrowserManager: manager,
	}
}

func (a *App) Close() {
	if a.BrowserManager != nil {
		log.Debug().Msg("Closing browser manager...")
		a.BrowserManager.Close()
	}
}

func configureLogger(level string) {
	switch strings.ToLower(level) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
}

func (a *App) SetupFileLogger(filename string) func() {
	originalLogger := log.Logger
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return func() {}
	}

	log.Logger = zerolog.New(f).With().Timestamp().Logger()
	return func() {
		log.Logger = originalLogger
		_ = f.Close()
	}
}
