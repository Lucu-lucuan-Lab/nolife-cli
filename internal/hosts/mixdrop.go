package hosts

import (
	"context"
	"fmt"
	"strings"

	"nolife-cli/internal/browser"
	"nolife-cli/internal/config"
	"nolife-cli/internal/utils"

	"github.com/rs/zerolog/log"
)

type MixDrop struct {
	cfg     *config.HostConfig
	manager *browser.Manager
}

func NewMixDrop(cfg *config.HostConfig, manager *browser.Manager) *MixDrop {
	return &MixDrop{
		cfg:     cfg,
		manager: manager,
	}
}

func (m *MixDrop) Name() string {
	return "mixdrop"
}

func (m *MixDrop) CanHandle(url string) bool {
	for _, domain := range m.cfg.Domains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

const (
	jsMixDropReadyCheck = `(function() {
		return !!document.querySelector('video, video source, a[download], a[href*=".mp4"], a[href*=".mkv"]') ||
			(typeof MDCore !== 'undefined' && !!MDCore.wurl) || (typeof wurl !== 'undefined' && !!wurl);
	})()`

	jsMixDropPlayVideo = `(function() {
		let v = document.querySelector('video');
		if (v) { v.play(); return "played video"; }
		let b = document.querySelector('.vjs-big-play-button, .play-button, [class*="play"]');
		if (b) { b.click(); return "clicked play"; }
		return "no player found";
	})()`

	jsMixDropAlternativeMethods = `(function() {
		let v = document.querySelector('video');
		if (v && v.src) return v.src;
		let s = document.querySelector('video source');
		if (s && s.src) return s.src;
		let links = document.querySelectorAll('a[href*=".mp4"], a[href*=".mkv"], a[download]');
		for (let l of links) { if (l.href && !l.href.includes('ads')) return l.href; }
		if (typeof MDCore !== 'undefined' && MDCore.wurl) return MDCore.wurl;
		if (typeof wurl !== 'undefined') return wurl;
		return '';
	})()`
)

func (m *MixDrop) ExtractDownloadURL(ctx context.Context, pageURL string) (string, error) {
	b, err := m.manager.NewPage()
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}
	defer b.Close()

	browserCtx := b.Context()
	resultChan := make(chan string, 1)

	b.ListenForRequests(browserCtx, func(url string) {
		if !utils.IsAdURL(url) && utils.FilterVideoURL(url) {
			select {
			case resultChan <- utils.SanitizeVideoURL(url):
				log.Info().Str("url", url).Msg("Captured download URL from MixDrop traffic")
			default:
			}
		}
	})

	if err := b.EnableNetwork(browserCtx); err != nil {
		return "", fmt.Errorf("failed to enable network: %w", err)
	}

	log.Info().Str("url", pageURL).Msg("Navigating to MixDrop...")
	if err := b.Navigate(browserCtx, pageURL); err != nil {
		return "", err
	}
	if err := b.WaitVisible(browserCtx, "body"); err != nil {
		return "", err
	}

	waitForAnyXPath(browserCtx, b, m.cfg.Selectors, extractorReadyTimeout)
	waitForTruthyEval(browserCtx, b, extractorReadyTimeout, jsMixDropReadyCheck)

	log.Info().Msg("Attempting download triggers...")
	if ClickDownloadButton(browserCtx, b, m.cfg.Selectors) {
		log.Info().Msg("Clicked MixDrop download button")
	} else {
		var playRes string
		if err := b.Evaluate(browserCtx, jsMixDropPlayVideo, &playRes); err == nil {
			log.Info().Str("action", playRes).Msg("Triggered video player")
		}
	}

	result, err := WaitForResult(ctx, resultChan, extractorCaptureTimeout)
	if err != nil {
		log.Warn().Err(err).Msg("Capture timed out, trying alternatives")
	}

	if result == "" {
		var altURL string
		if err := b.Evaluate(browserCtx, jsMixDropAlternativeMethods, &altURL); err == nil && altURL != "" {
			if utils.FilterVideoURL(altURL) {
				result = utils.SanitizeVideoURL(altURL)
				log.Info().Msg("Found URL via alternative JS methods")
			}
		}
	}

	if result == "" {
		return "", fmt.Errorf("failed to capture MixDrop download URL")
	}

	return result, nil
}
