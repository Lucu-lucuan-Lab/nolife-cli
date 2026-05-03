package hosts

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"nolife-cli/internal/browser"
	"nolife-cli/internal/config"
	"nolife-cli/internal/utils"

	"github.com/rs/zerolog/log"
)

type GoogleDrive struct {
	cfg     *config.HostConfig
	manager *browser.Manager
}

func NewGoogleDrive(cfg *config.HostConfig, manager *browser.Manager) *GoogleDrive {
	return &GoogleDrive{
		cfg:     cfg,
		manager: manager,
	}
}

func (g *GoogleDrive) Name() string {
	return "googledrive"
}

func (g *GoogleDrive) CanHandle(urlStr string) bool {
	for _, domain := range g.cfg.Domains {
		if strings.Contains(urlStr, domain) {
			return true
		}
	}
	return false
}

var (
	gdFileIDRegex1 = regexp.MustCompile(`/file/d/([a-zA-Z0-9_-]+)`)
	gdFileIDRegex2 = regexp.MustCompile(`[?&]id=([a-zA-Z0-9_-]+)`)
)

const jsGoogleDriveSubmitForm = `(function() {
	let f = document.querySelector('form[action*="download"]');
	if (f) { f.submit(); return true; }
	return false;
})()`

func (g *GoogleDrive) ExtractDownloadURL(ctx context.Context, pageURL string) (string, error) {
	fileID := g.extractFileID(pageURL)
	if fileID == "" {
		return "", fmt.Errorf("could not extract file ID: %s", pageURL)
	}

	log.Info().Str("fileId", fileID).Msg("Google Drive ID detected")

	b, err := g.manager.NewPage()
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}
	defer b.Close()

	browserCtx := b.Context()
	resultChan := make(chan string, 1)

	b.ListenForRequests(browserCtx, func(url string) {
		if !utils.IsAdURL(url) && g.isDirectDownloadURL(url) {
			select {
			case resultChan <- utils.SanitizeVideoURL(url):
				log.Info().Str("url", url).Msg("Captured direct download URL")
			default:
			}
		}
	})

	if err := b.EnableNetwork(browserCtx); err != nil {
		return "", fmt.Errorf("failed to enable network: %w", err)
	}

	directURL := g.constructDirectURL(fileID)
	log.Info().Str("url", directURL).Msg("Navigating to Google Drive...")
	if err := b.Navigate(browserCtx, directURL); err != nil {
		return "", err
	}

	if err := b.WaitVisible(browserCtx, "body"); err != nil {
		return "", err
	}

	if g.handleVirusScanWarning(browserCtx, b) {
		log.Info().Msg("Bypassed virus scan warning")
	}

	result, err := WaitForResult(ctx, resultChan, extractorCaptureTimeout)
	if err != nil {
		log.Warn().Err(err).Msg("Direct capture timed out, trying fallbacks")
	}

	if result == "" {
		result, _ = g.tryExtractFromPage(browserCtx, b, fileID)
	}
	if result == "" {
		result = g.constructConfirmedURL(fileID)
		log.Info().Str("url", result).Msg("Using generated confirmed URL")
	}

	return result, nil
}

func (g *GoogleDrive) extractFileID(urlStr string) string {
	if matches := gdFileIDRegex1.FindStringSubmatch(urlStr); len(matches) > 1 {
		return matches[1]
	}

	if parsed, err := url.Parse(urlStr); err == nil {
		if id := parsed.Query().Get("id"); id != "" {
			return id
		}
	}

	if matches := gdFileIDRegex2.FindStringSubmatch(urlStr); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

func (g *GoogleDrive) constructDirectURL(fileID string) string {
	return fmt.Sprintf("https://drive.usercontent.google.com/download?id=%s&export=download", fileID)
}

func (g *GoogleDrive) constructConfirmedURL(fileID string) string {
	return fmt.Sprintf("https://drive.usercontent.google.com/download?id=%s&export=download&confirm=t", fileID)
}

func (g *GoogleDrive) isDirectDownloadURL(urlStr string) bool {
	return (strings.Contains(urlStr, "googleusercontent.com") && strings.Contains(urlStr, "confirm=")) ||
		strings.Contains(urlStr, "&export=download&confirm=") ||
		strings.Contains(urlStr, "doc-") ||
		(strings.Contains(urlStr, "google") && (strings.Contains(urlStr, ".mp4") || strings.Contains(urlStr, ".mkv")))
}

func (g *GoogleDrive) handleVirusScanWarning(ctx context.Context, b browser.Browser) bool {
	for _, selector := range g.cfg.Selectors {
		if exists, _ := b.CheckElementExists(ctx, selector); exists {
			if err := b.Click(ctx, selector); err == nil {
				return true
			}
		}
	}

	if clicked, _ := b.ClickByText(ctx, "download anyway"); clicked {
		return true
	}

	var submitted bool
	if err := b.Evaluate(ctx, jsGoogleDriveSubmitForm, &submitted); err == nil && submitted {
		return true
	}

	return false
}

func (g *GoogleDrive) tryExtractFromPage(ctx context.Context, b browser.Browser, fileID string) (string, error) {
	log.Debug().Msg("Trying to extract download URL from page")

	currentURL, err := b.GetCurrentURL(ctx)
	if err == nil && strings.Contains(currentURL, "confirm=") {
		return currentURL, nil
	}

	links, err := b.GetAllLinkHrefs(ctx)
	if err != nil {
		return "", err
	}

	for _, link := range links {
		if strings.Contains(link, fileID) && strings.Contains(link, "confirm=") {
			return link, nil
		}
	}

	return "", fmt.Errorf("could not extract download URL from page")
}
