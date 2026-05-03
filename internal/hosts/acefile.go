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

type Acefile struct {
	cfg     *config.HostConfig
	manager *browser.Manager
}

func NewAcefile(cfg *config.HostConfig, manager *browser.Manager) *Acefile {
	return &Acefile{
		cfg:     cfg,
		manager: manager,
	}
}

func (a *Acefile) Name() string {
	return "acefile"
}

func (a *Acefile) CanHandle(url string) bool {
	for _, domain := range a.cfg.Domains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}

const (
	jsAcefileReadyCheck = `(function() {
		return !!document.getElementById('3-2-1') ||
			!!document.getElementById('no-login-dl') ||
			!!document.querySelector('a[href*="drive.google.com"], a[href*="drive.usercontent.google.com"]') ||
			Array.from(document.scripts).some(function(script) {
				var text = script.textContent || '';
				return text.includes('"AceFile":[{') || text.includes('"b":[{') || 
					   text.includes('drive.google.com') || text.includes('drive.usercontent.google.com');
			});
	})()`

	jsAcefileClickButtons = `(function() {
		var btn321 = document.getElementById('3-2-1');
		if (btn321) { btn321.click(); return "clicked 3-2-1"; }
		
		var links = Array.from(document.querySelectorAll('a, button, .btn, .button'));
		
		var gd = links.find(el => {
			let t = el.textContent.toLowerCase();
			return t === 'google drive' || t === 'gd' || t.includes('drive.google.com');
		});
		if (gd) { gd.click(); return "clicked GD button"; }

		var m = links.find(el => {
			let t = el.textContent.toLowerCase();
			return t.includes('acemirror') || t.includes('fast download') || t.includes('no login');
		});
		if (m) { m.click(); return "clicked mirror: " + m.textContent.trim(); }
		
		var dl = links.find(el => el.textContent.toLowerCase().includes('download'));
		if (dl) { dl.click(); return "clicked generic: " + dl.textContent.trim(); }
		
		return "no button found";
	})()`

	jsAcefileDirectExtraction = `(function() {
		var links = document.querySelectorAll('a[href*="drive.google.com"], a[href*="drive.usercontent.google.com"]');
		for (var i = 0; i < links.length; i++) { if (links[i].href) return links[i].href; }

		var scripts = document.querySelectorAll('script');
		for (var i = 0; i < scripts.length; i++) {
			var t = scripts[i].textContent || scripts[i].innerText;
			var m = t.match(/https:\/\/drive\.(?:google\.com|usercontent\.google\.com)[^"'\s]+/);
			if (m) return m[0];
		}

		var btn = document.getElementById('3-2-1');
		if (btn) {
			var oc = btn.getAttribute('onclick') || '';
			var m2 = oc.match(/https:\/\/drive[^"'\s\)]+/);
			if (m2) return m2[0];
		}
		return '';
	})()`

	jsAcefileAdvancedExtraction = `(function() {
		var scripts = document.querySelectorAll('script');
		for (var i = 0; i < scripts.length; i++) {
			var t = scripts[i].textContent || '';
			var m = t.match(/[A-Za-z0-9_-]{33,}/); 
			if (m && t.includes('drive')) return "https://drive.google.com/uc?export=download&id=" + m[0];
		}
		return '';
	})()`

	jsAcefileHasMirror = `(function() {
		if (typeof DUAR !== 'undefined' && DUAR && DUAR.AceFile && DUAR.AceFile.length > 0) return true;
		return Array.from(document.scripts).some(s => {
			let t = s.textContent || '';
			return (t.includes('"AceFile":[{') || t.includes('"b":[{') || t.includes('drive.google.com')) && 
				   !t.includes('DUAR=false') && !t.includes('DUAR = false');
		});
	})()`
)

func (a *Acefile) ExtractDownloadURL(ctx context.Context, pageURL string) (string, error) {
	b, err := a.manager.NewPage()
	if err != nil {
		return "", fmt.Errorf("failed to create page: %w", err)
	}
	defer b.Close()

	browserCtx := b.Context()
	resultChan := make(chan string, 1)

	b.ListenForRequests(browserCtx, func(url string) {
		if !utils.IsAdURL(url) && a.isGoogleDriveURL(url) {
			select {
			case resultChan <- utils.SanitizeVideoURL(url):
				log.Info().Str("url", url).Msg("Captured Google Drive URL from Acefile")
			default:
			}
		}
	})

	if err := b.EnableNetwork(browserCtx); err != nil {
		return "", fmt.Errorf("failed to enable network: %w", err)
	}

	log.Info().Str("url", pageURL).Msg("Navigating to Acefile...")
	if err := b.Navigate(browserCtx, pageURL); err != nil {
		return "", err
	}
	if err := b.WaitVisible(browserCtx, "body"); err != nil {
		return "", err
	}

	log.Debug().Msg("Waiting for extractor elements...")
	_ = waitForTruthyEval(browserCtx, b, extractorReadyTimeout, jsAcefileReadyCheck)

	log.Info().Msg("Executing button flow...")
	var clickResult string
	_ = b.Evaluate(browserCtx, jsAcefileClickButtons, &clickResult)

	result, err := WaitForResult(ctx, resultChan, extractorCaptureTimeout)
	if err != nil {
		log.Warn().Err(err).Msg("Capture timed out, trying fallbacks")
	}

	if result == "" {
		result, _ = a.tryGetRedirectURL(browserCtx, b)
	}
	if result == "" {
		log.Info().Msg("Trying direct extraction...")
		_ = b.Evaluate(browserCtx, jsAcefileDirectExtraction, &result)
	}
	if result == "" {
		log.Info().Msg("Trying advanced script extraction...")
		_ = b.Evaluate(browserCtx, jsAcefileAdvancedExtraction, &result)
	}

	if result == "" || !a.isGoogleDriveURL(result) {
		var hasMirror bool
		_ = b.Evaluate(browserCtx, jsAcefileHasMirror, &hasMirror)
		if !hasMirror {
			return "", fmt.Errorf("NO_MIRROR: file has no Google Drive mirror")
		}
		return "", fmt.Errorf("could not capture Google Drive URL")
	}

	return result, nil
}

func (a *Acefile) isGoogleDriveURL(url string) bool {
	return strings.Contains(url, "drive.google.com") ||
		strings.Contains(url, "drive.usercontent.google.com") ||
		strings.Contains(url, "googleusercontent.com/download")
}

func (a *Acefile) tryGetRedirectURL(ctx context.Context, b browser.Browser) (string, error) {
	currentURL, err := b.GetCurrentURL(ctx)
	if err == nil && a.isGoogleDriveURL(currentURL) {
		return currentURL, nil
	}

	links, err := b.GetAllLinkHrefs(ctx)
	if err != nil {
		return "", err
	}

	for _, link := range links {
		if a.isGoogleDriveURL(link) {
			return link, nil
		}
	}

	return "", fmt.Errorf("no Google Drive URL found")
}
