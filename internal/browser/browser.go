package browser

import (
	"context"

	"github.com/chromedp/cdproto/cdp"
)

type Browser interface {
	Context() context.Context
	Navigate(ctx context.Context, url string) error
	GetTitle(ctx context.Context) (string, error)
	GetAllLinks(ctx context.Context) ([]*cdp.Node, error)
	WaitVisible(ctx context.Context, selector string) error
	Click(ctx context.Context, selector string) error
	ClickByText(ctx context.Context, text string) (bool, error)
	GetPageHTML(ctx context.Context) (string, error)
	GetCurrentURL(ctx context.Context) (string, error)
	Evaluate(ctx context.Context, script string, result interface{}) error
	ListenForDownloads(ctx context.Context, callback func(url string))
	ListenForRequests(ctx context.Context, callback func(url string))
	EnableNetwork(ctx context.Context) error
	EnableAdBlocking(ctx context.Context) error
	Sleep(ctx context.Context, seconds int)
	CheckElementExists(ctx context.Context, xpath string) (bool, error)
	GetAttributeValue(ctx context.Context, selector, attr string) (string, error)
	GetAllLinkHrefs(ctx context.Context) ([]string, error)
	Close()
}
