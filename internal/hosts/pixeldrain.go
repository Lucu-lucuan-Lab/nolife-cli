package hosts

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"nolife-cli/internal/browser"
	"nolife-cli/internal/config"
	"nolife-cli/internal/utils"

	"github.com/rs/zerolog/log"
)

type Pixeldrain struct {
	cfg     *config.HostConfig
	manager *browser.Manager
}

func NewPixeldrain(cfg *config.HostConfig, manager *browser.Manager) *Pixeldrain {
	return &Pixeldrain{
		cfg:     cfg,
		manager: manager,
	}
}

func (p *Pixeldrain) Name() string {
	return "pixeldrain"
}

func (p *Pixeldrain) CanHandle(url string) bool {
	for _, domain := range p.cfg.Domains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

var (
	pxFileIDRegex1 = regexp.MustCompile(`pixeldrain\.com/u/([a-zA-Z0-9]+)`)
	pxFileIDRegex2 = regexp.MustCompile(`pixeldrain\.com/api/file/([a-zA-Z0-9]+)`)
	pxFileIDRegex3 = regexp.MustCompile(`/([a-zA-Z0-9]{6,12})(?:\?|$)`)
)

func (p *Pixeldrain) ExtractDownloadURL(ctx context.Context, pageURL string) (string, error) {
	fileID := p.extractFileID(pageURL)
	if fileID == "" {
		return "", fmt.Errorf("could not extract Pixeldrain file ID: %s", pageURL)
	}

	log.Info().Str("fileId", fileID).Msg("Pixeldrain ID detected")
	downloadURL := fmt.Sprintf("https://pixeldrain.com/api/file/%s?download", fileID)

	if err := p.verifyURL(ctx, downloadURL); err != nil {
		log.Warn().Err(err).Msg("Verification failed with ?download, trying direct")
		downloadURL = fmt.Sprintf("https://pixeldrain.com/api/file/%s", fileID)
		if err := p.verifyURL(ctx, downloadURL); err != nil {
			return "", fmt.Errorf("pixeldrain file not accessible: %w", err)
		}
	}

	return downloadURL, nil
}

func (p *Pixeldrain) extractFileID(url string) string {
	if matches := pxFileIDRegex1.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}
	if matches := pxFileIDRegex2.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}
	if matches := pxFileIDRegex3.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (p *Pixeldrain) verifyURL(ctx context.Context, url string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return err
	}

	for k, v := range utils.GetBrowserHeaders("https://pixeldrain.com/") {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusPartialContent &&
		resp.StatusCode != http.StatusFound {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	return nil
}
