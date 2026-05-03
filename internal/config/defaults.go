package config

import (
	"os"
	"path/filepath"
	"time"
)

func GetDefaultDownloadDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./downloads"
	}
	return filepath.Join(homeDir, "Downloads", "Anime")
}

func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			OutputDir:     GetDefaultDownloadDir(),
			LogLevel:      "error",
			MaxConcurrent: 10,
		},
		Browser: BrowserConfig{
			Headless:      true,
			DisableGPU:    true,
			DisableImages: true,
			UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			Timeout: TimeoutConfig{
				Navigation:  60 * time.Second,
				PageLoad:    30 * time.Second,
				ElementWait: 15 * time.Second,
			},
		},
		Download: DownloadConfig{
			SkipExisting:     true,
			PreferredQuality: "highest",
		},
		Sites: map[string]SiteConfig{
			"oploverz": {
				Domains: []string{
					"coba.oploverz.ltd",
					"anime1.oploverz.ac",
					"oploverz.net",
					"oploverz.ac",
				},
				EpisodePattern: "episode",
			},
		},
		Hosts: map[string]HostConfig{
			"googledrive": {
				Domains: []string{
					"drive.google.com",
					"drive.usercontent.google.com",
				},
				Selectors: []string{
					`//a[@id="uc-download-link"]`,
					`//a[contains(text(), "Download anyway")]`,
					`//button[contains(text(), "Download")]`,
					`//form[@id="download-form"]//button`,
				},
			},
			"acefile": {
				Domains: []string{
					"acefile.co",
					"acefile.com",
					"akirabox.com",
				},
				Selectors: []string{
					`//*[@id="3-2-1"]`,
					`//a[contains(text(), "Fast Download")]`,
					`//button[contains(text(), "Fast Download")]`,
					`//a[contains(text(), "Download")]`,
				},
			},
			"mixdrop": {
				Domains: []string{
					"mixdrop.co",
					"mixdrop.to",
					"mixdrop.ch",
					"mixdrop.sx",
					"mixdrop.bz",
					"mixdrop.gl",
					"m1xdrop.net",
				},
				Selectors: []string{
					`//*[@id="download-link"]`,
					`//a[contains(@class, "download")]`,
					`//button[contains(text(), "Download")]`,
					`//a[contains(text(), "Download")]`,
				},
			},
			"filedon": {
				Domains: []string{
					"filedon.co",
				},
				Selectors: []string{
					`//button[contains(text(), "Download Now")]`,
					`//a[contains(text(), "Download Now")]`,
					`//button[contains(text(), "Download")]`,
					`//a[contains(text(), "Download")]`,
				},
			},
			"pixeldrain": {
				Domains: []string{
					"pixeldrain.com",
				},
				Selectors: []string{},
			},
		},
	}
}
