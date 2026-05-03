package hosts

import (
	"context"
	"fmt"
	"time"

	"nolife-cli/internal/browser"

	"github.com/rs/zerolog/log"
)

const (
	extractorReadyTimeout   = 12 * time.Second
	extractorCaptureTimeout = 30 * time.Second
	waitPollInterval        = 250 * time.Millisecond
)

func ClickDownloadButton(ctx context.Context, b browser.Browser, selectors []string) bool {
	for _, selector := range selectors {
		if exists, _ := b.CheckElementExists(ctx, selector); exists {
			log.Debug().Str("selector", selector).Msg("Clicking button via selector")
			if err := b.Click(ctx, selector); err == nil {
				return true
			}
		}
	}

	log.Debug().Msg("Selectors failed, trying common text labels...")
	labels := []string{"download", "fast download", "download anyway", "generate link"}
	for _, label := range labels {
		if clicked, _ := b.ClickByText(ctx, label); clicked {
			log.Debug().Str("label", label).Msg("Clicked button by text")
			return true
		}
	}

	return false
}

func waitUntil(ctx context.Context, timeout time.Duration, check func(context.Context) (bool, error)) bool {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(waitPollInterval)
	defer ticker.Stop()

	for {
		ready, err := check(waitCtx)
		if err == nil && ready {
			return true
		}
		if err != nil {
			log.Trace().Err(err).Msg("Wait condition check failed (retrying)")
		}

		select {
		case <-waitCtx.Done():
			return false
		case <-ticker.C:
		}
	}
}

func waitForAnyXPath(ctx context.Context, b browser.Browser, selectors []string, timeout time.Duration) bool {
	return waitUntil(ctx, timeout, func(checkCtx context.Context) (bool, error) {
		for _, selector := range selectors {
			if exists, _ := b.CheckElementExists(checkCtx, selector); exists {
				return true, nil
			}
		}
		return false, nil
	})
}

func waitForTruthyEval(ctx context.Context, b browser.Browser, timeout time.Duration, script string) bool {
	return waitUntil(ctx, timeout, func(checkCtx context.Context) (bool, error) {
		var ready bool
		if err := b.Evaluate(checkCtx, script, &ready); err != nil {
			return false, err
		}
		return ready, nil
	})
}

func WaitForResult(ctx context.Context, resultChan <-chan string, timeout time.Duration) (string, error) {
	select {
	case result := <-resultChan:
		return result, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for result")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
