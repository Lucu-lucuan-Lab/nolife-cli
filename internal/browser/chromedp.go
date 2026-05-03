package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nolife-cli/internal/utils"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

type ChromeDP struct {
	opts        *Options
	allocCancel context.CancelFunc
	ctxCancel   context.CancelFunc
	ctx         context.Context
}

func NewChromeDP(opts *Options) (*ChromeDP, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	manager, err := NewManager(opts)
	if err != nil {
		return nil, err
	}

	ctx, ctxCancel := chromedp.NewContext(manager.allocCtx)

	return &ChromeDP{
		opts:        opts,
		allocCancel: manager.allocCancel,
		ctxCancel:   ctxCancel,
		ctx:         ctx,
	}, nil
}

func (c *ChromeDP) Context() context.Context {
	return c.ctx
}

func (c *ChromeDP) Close() {
	if c.ctxCancel != nil {
		c.ctxCancel()
	}
	if c.allocCancel != nil {
		c.allocCancel()
	}
}

func (c *ChromeDP) withTimeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = c.ctx
	}
	return context.WithTimeout(baseCtx, timeout)
}

func (c *ChromeDP) Navigate(ctx context.Context, url string) error {
	navCtx, cancel := c.withTimeoutContext(ctx, c.opts.NavigationTimeout)
	defer cancel()

	err := chromedp.Run(navCtx, chromedp.Navigate(url))
	if err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}
	return nil
}

func (c *ChromeDP) GetTitle(ctx context.Context) (string, error) {
	titleCtx, cancel := c.withTimeoutContext(ctx, c.opts.PageLoadTimeout)
	defer cancel()

	var title string
	err := chromedp.Run(titleCtx, chromedp.Title(&title))
	if err != nil {
		return "", fmt.Errorf("failed to get page title: %w", err)
	}
	return title, nil
}

func (c *ChromeDP) GetAllLinks(ctx context.Context) ([]*cdp.Node, error) {
	linksCtx, cancel := c.withTimeoutContext(ctx, c.opts.ElementWaitTime)
	defer cancel()

	var nodes []*cdp.Node
	err := chromedp.Run(linksCtx,
		chromedp.WaitVisible(`a`, chromedp.ByQuery),
		chromedp.Nodes(`a`, &nodes, chromedp.ByQueryAll),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get links: %w", err)
	}
	return nodes, nil
}

func (c *ChromeDP) WaitVisible(ctx context.Context, selector string) error {
	waitCtx, cancel := context.WithTimeout(ctx, c.opts.ElementWaitTime)
	defer cancel()

	err := chromedp.Run(waitCtx, chromedp.WaitVisible(selector, chromedp.ByQuery))
	if err != nil {
		return fmt.Errorf("element not visible: %s: %w", selector, err)
	}
	return nil
}

func (c *ChromeDP) Click(ctx context.Context, selector string) error {
	err := chromedp.Run(ctx, chromedp.Click(selector, chromedp.BySearch))
	if err != nil {
		return fmt.Errorf("failed to click %s: %w", selector, err)
	}
	return nil
}

func (c *ChromeDP) ClickByText(ctx context.Context, text string) (bool, error) {
	var clicked bool
	script := fmt.Sprintf(`
		(function() {
			let elements = document.querySelectorAll('a, button');
			for (let el of elements) {
				if (el.innerText && el.innerText.toLowerCase().includes('%s')) {
					el.click();
					return true;
				}
			}
			return false;
		})()
	`, strings.ToLower(text))

	err := chromedp.Run(ctx, chromedp.Evaluate(script, &clicked))
	if err != nil {
		return false, fmt.Errorf("failed to click by text: %w", err)
	}
	return clicked, nil
}

func (c *ChromeDP) GetPageHTML(ctx context.Context) (string, error) {
	var html string
	err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html))
	if err != nil {
		return "", fmt.Errorf("failed to get page HTML: %w", err)
	}
	return html, nil
}

func (c *ChromeDP) GetCurrentURL(ctx context.Context) (string, error) {
	var url string
	err := chromedp.Run(ctx, chromedp.Location(&url))
	if err != nil {
		return "", fmt.Errorf("failed to get current URL: %w", err)
	}
	return url, nil
}

func (c *ChromeDP) Evaluate(ctx context.Context, script string, result interface{}) error {
	return chromedp.Run(ctx, chromedp.Evaluate(script, result))
}

func (c *ChromeDP) ListenForDownloads(ctx context.Context, callback func(url string)) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			url := e.Request.URL
			isMediaURL := strings.Contains(url, ".mp4") ||
				strings.Contains(url, ".mkv") ||
				strings.Contains(url, ".avi") ||
				strings.Contains(url, ".webm")

			isExcluded := strings.HasSuffix(url, ".js") ||
				strings.HasSuffix(url, ".css") ||
				strings.HasSuffix(url, ".png") ||
				strings.HasSuffix(url, ".jpg") ||
				strings.HasSuffix(url, ".svg") ||
				strings.Contains(url, "/assets/") ||
				strings.Contains(url, "/build/") ||
				strings.Contains(url, "cdn-cgi") ||
				strings.Contains(url, "/rum")

			if isMediaURL && !isExcluded {
				callback(url)
			}
		}
	})
}

func (c *ChromeDP) ListenForRequests(ctx context.Context, callback func(url string)) {
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			url := e.Request.URL

			isExcluded := strings.HasSuffix(url, ".js") ||
				strings.HasSuffix(url, ".css") ||
				strings.HasSuffix(url, ".png") ||
				strings.HasSuffix(url, ".jpg") ||
				strings.HasSuffix(url, ".gif") ||
				strings.HasSuffix(url, ".svg") ||
				strings.HasSuffix(url, ".ico") ||
				strings.HasSuffix(url, ".woff") ||
				strings.HasSuffix(url, ".woff2") ||
				strings.Contains(url, "/assets/") ||
				strings.Contains(url, "/build/") ||
				strings.Contains(url, "cdn-cgi") ||
				strings.Contains(url, "/rum") ||
				strings.Contains(url, "analytics") ||
				strings.Contains(url, "tracking")

			if !isExcluded {
				callback(url)
			}
		}
	})
}

func (c *ChromeDP) EnableNetwork(ctx context.Context) error {
	return chromedp.Run(ctx, network.Enable())
}

func (c *ChromeDP) EnableAdBlocking(ctx context.Context) error {
	err := chromedp.Run(ctx, fetch.Enable().WithPatterns([]*fetch.RequestPattern{
		{URLPattern: "*", RequestStage: fetch.RequestStageRequest},
	}))
	if err != nil {
		return err
	}

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				url := e.Request.URL

				if utils.IsAdURL(url) || c.isBlockedResource(url) {
					_ = chromedp.Run(ctx, fetch.FailRequest(e.RequestID, network.ErrorReasonBlockedByClient))
					return
				}

				_ = chromedp.Run(ctx, fetch.ContinueRequest(e.RequestID))
			}()
		}
	})

	return nil
}

func (c *ChromeDP) isBlockedResource(url string) bool {
	lowerURL := strings.ToLower(url)

	blockedExtensions := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".bmp",
		".woff", ".woff2", ".ttf", ".eot", ".mp3", ".wav", ".ogg"}
	for _, ext := range blockedExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}

	blockedPatterns := []string{
		"googleads", "googlesyndication", "doubleclick",
		"facebook.com/tr", "fbcdn", "facebook.net",
		"amazon-adsystem", "adservice", "pagead",
		"popads", "popcash", "propeller", "exoclick",
		"juicyads", "trafficjunky", "adsterra",
		"/ads/", "/ad/", "/banner/", "/sponsor/",
		"analytics", "tracking", "pixel", "beacon",
		"taboola", "outbrain", "mgid", "revcontent",
	}
	for _, pattern := range blockedPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}

	return false
}

func (c *ChromeDP) Sleep(ctx context.Context, seconds int) {
	chromedp.Run(ctx, chromedp.Sleep(time.Duration(seconds)*time.Second))
}

func (c *ChromeDP) CheckElementExists(ctx context.Context, xpath string) (bool, error) {
	var exists bool
	script := fmt.Sprintf(
		`document.evaluate('%s', document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue !== null`,
		xpath,
	)
	err := chromedp.Run(ctx, chromedp.Evaluate(script, &exists))
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (c *ChromeDP) GetAttributeValue(ctx context.Context, selector, attr string) (string, error) {
	var value string
	var ok bool
	err := chromedp.Run(ctx,
		chromedp.AttributeValue(selector, attr, &value, &ok, chromedp.BySearch),
	)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (c *ChromeDP) GetAllLinkHrefs(ctx context.Context) ([]string, error) {
	var links []string
	err := chromedp.Run(ctx, chromedp.Evaluate(`
		Array.from(document.querySelectorAll('a[href]')).map(a => a.href)
	`, &links))
	if err != nil {
		return nil, err
	}
	return links, nil
}
