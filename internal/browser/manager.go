package browser

import (
	"context"
	"fmt"
	"sync"

	"github.com/chromedp/chromedp"
)

type Manager struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	opts        *Options
	mu          sync.Mutex
}

func NewManager(opts *Options) (*Manager, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	var chromeOpts []chromedp.ExecAllocatorOption

	if opts.Headless {
		chromeOpts = append(chromeOpts, chromedp.Flag("headless", "new"))
		chromeOpts = append(chromeOpts, chromedp.DisableGPU)
	} else {
		chromeOpts = append(chromeOpts, chromedp.Flag("headless", false))
		chromeOpts = append(chromeOpts, chromedp.Flag("start-maximized", true))
	}

	chromeOpts = append(chromeOpts,
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("excludeSwitches", "enable-automation"),
	)

	chromeOpts = append(chromeOpts,
		chromedp.UserAgent(opts.UserAgent),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),
		chromedp.Flag("disable-features", "PrivateNetworkAccessPermissionPrompt"),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("deny-permission-prompts", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("disable-java", true),
		chromedp.Flag("disable-translate", true),
		chromedp.Flag("disable-component-update", true),
		chromedp.Flag("no-pings", true),
	)

	chromeOpts = append(chromeOpts, chromedp.Flag("blink-settings", "imagesEnabled=false"))

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), chromeOpts...)

	return &Manager{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		opts:        opts,
	}, nil
}

func (m *Manager) NewPage() (Browser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.allocCtx == nil {
		return nil, fmt.Errorf("browser manager is closed")
	}

	ctx, ctxCancel := chromedp.NewContext(m.allocCtx)

	if err := chromedp.Run(ctx); err != nil {
		ctxCancel()
		return nil, fmt.Errorf("failed to start browser tab: %w", err)
	}

	return &ChromeDP{
		ctx:       ctx,
		ctxCancel: ctxCancel,
		opts:      m.opts,
	}, nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.allocCancel != nil {
		m.allocCancel()
		m.allocCancel = nil
	}
}
