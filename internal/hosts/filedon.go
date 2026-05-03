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

type Filedon struct {
	cfg     *config.HostConfig
	manager *browser.Manager
}

func NewFiledon(cfg *config.HostConfig, manager *browser.Manager) *Filedon {
	return &Filedon{
		cfg:     cfg,
		manager: manager,
	}
}

func (f *Filedon) Name() string {
	return "filedon"
}

func (f *Filedon) CanHandle(url string) bool {
	for _, domain := range f.cfg.Domains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

const (
	jsFiledonReadyCheck = `(function() {
		return !!document.querySelector('video, video source, a[download], a[href*="download"], form[action*="download"], iframe[src]');
	})()`

	jsFiledonAlternativeMethods = `(function() {
		function firstNonAd(values) {
			for (const value of values) {
				if (!value) continue;
				const lower = value.toLowerCase();
				if (lower.includes('/ads/') || lower.includes('doubleclick') || lower.includes('googlesyndication')) continue;
				return value;
			}
			return '';
		}

		const video = document.querySelector('video');
		if (video) {
			const found = firstNonAd([video.currentSrc, video.src]);
			if (found) return found;
		}

		const source = document.querySelector('video source');
		if (source) {
			const found = firstNonAd([source.src]);
			if (found) return found;
		}

		const selectors = [
			'a[download]', 'a[href*="download"]', 'a[href*=".mp4"]', 'a[href*=".mkv"]',
			'a[href*=".m3u8"]', 'form[action*="download"]', 'iframe[src]'
		];

		for (const selector of selectors) {
			const elements = document.querySelectorAll(selector);
			for (const el of elements) {
				const value = el.href || el.action || el.src || '';
				const found = firstNonAd([value]);
				if (found) return found;
			}
		}

		const dataAttrs = ['data-url', 'data-href', 'data-download', 'data-file', 'data-src'];
		for (const el of document.querySelectorAll('*')) {
			for (const attr of dataAttrs) {
				const value = el.getAttribute && el.getAttribute(attr);
				const found = firstNonAd([value]);
				if (found) return found;
			}
			const onclick = el.getAttribute && el.getAttribute('onclick');
			if (onclick) {
				const match = onclick.match(/https?:\/\/[^"'\s)]+/i);
				if (match && match[0]) return match[0];
			}
		}

		for (const script of document.scripts) {
			const text = script.textContent || '';
			const matches = text.match(/https?:\/\/[^"'\s\\]+/gi) || [];
			for (const match of matches) {
				const found = firstNonAd([match]);
				if (found) return found;
			}
		}
		return '';
	})()`
)

func (f *Filedon) ExtractDownloadURL(ctx context.Context, episodeURL string) (string, error) {
	b, err := f.manager.NewPage()
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}
	defer b.Close()

	browserCtx := b.Context()
	var filedonURL string

	if f.CanHandle(episodeURL) {
		filedonURL = episodeURL
		log.Info().Str("url", episodeURL).Msg("Using direct Filedon URL")
	} else {
		log.Info().Str("url", episodeURL).Msg("Finding Filedon link on page")
		if err := b.Navigate(browserCtx, episodeURL); err != nil {
			return "", err
		}
		if err := b.WaitVisible(browserCtx, "body"); err != nil {
			return "", err
		}
		links, err := b.GetAllLinkHrefs(browserCtx)
		if err != nil {
			return "", err
		}
		for _, href := range links {
			if f.CanHandle(href) {
				filedonURL = href
				break
			}
		}
		if filedonURL == "" {
			return "", fmt.Errorf("filedon link not found")
		}
	}

	resultChan := make(chan string, 1)
	b.ListenForDownloads(browserCtx, func(url string) {
		if utils.FilterVideoURL(url) {
			select {
			case resultChan <- utils.SanitizeVideoURL(url):
				log.Info().Str("url", url).Msg("Captured download URL via download event")
			default:
			}
		}
	})
	b.ListenForRequests(browserCtx, func(url string) {
		if utils.FilterVideoURL(url) {
			select {
			case resultChan <- utils.SanitizeVideoURL(url):
				log.Info().Str("url", url).Msg("Captured download URL via network traffic")
			default:
			}
		}
	})

	if err := b.EnableNetwork(browserCtx); err != nil {
		return "", fmt.Errorf("failed to enable network: %w", err)
	}

	log.Info().Str("url", filedonURL).Msg("Navigating to Filedon...")
	if err := b.Navigate(browserCtx, filedonURL); err != nil {
		return "", err
	}
	if err := b.WaitVisible(browserCtx, "body"); err != nil {
		return "", err
	}

	waitForAnyXPath(browserCtx, b, f.cfg.Selectors, extractorReadyTimeout)
	waitForTruthyEval(browserCtx, b, extractorReadyTimeout, jsFiledonReadyCheck)

	log.Info().Msg("Executing download sequence...")
	if ClickDownloadButton(browserCtx, b, f.cfg.Selectors) {
		log.Info().Msg("Clicked download button (Step 1)")
		b.Sleep(browserCtx, 2)
		if ClickDownloadButton(browserCtx, b, f.cfg.Selectors) {
			log.Info().Msg("Clicked download button (Step 2)")
		}
	}

	result, err := WaitForResult(ctx, resultChan, extractorCaptureTimeout)
	if err != nil {
		log.Warn().Err(err).Msg("Capture timed out, trying fallbacks")
	}

	if result == "" {
		result, _ = f.tryAlternativeMethods(browserCtx, b)
	}

	if result == "" {
		return "", fmt.Errorf("failed to capture download URL")
	}

	return result, nil
}

func (f *Filedon) tryAlternativeMethods(ctx context.Context, b browser.Browser) (string, error) {
	log.Debug().Msg("Trying alternative extraction methods")

	var candidateURL string
	if err := b.Evaluate(ctx, jsFiledonAlternativeMethods, &candidateURL); err == nil && utils.FilterVideoURL(candidateURL) {
		return utils.SanitizeVideoURL(candidateURL), nil
	}

	if currentURL, err := b.GetCurrentURL(ctx); err == nil && utils.FilterVideoURL(currentURL) {
		return utils.SanitizeVideoURL(currentURL), nil
	}

	if links, err := b.GetAllLinkHrefs(ctx); err == nil {
		for _, link := range links {
			if utils.FilterVideoURL(link) {
				return utils.SanitizeVideoURL(link), nil
			}
		}
	}

	return "", fmt.Errorf("all alternative methods failed")
}
