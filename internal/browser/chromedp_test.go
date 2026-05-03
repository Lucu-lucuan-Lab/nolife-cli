package browser

import (
	"context"
	"testing"
	"time"
)

func TestWithTimeoutContextUsesCallerContext(t *testing.T) {
	t.Parallel()

	c := &ChromeDP{
		ctx:  context.Background(),
		opts: DefaultOptions(),
	}

	parent, parentCancel := context.WithCancel(context.Background())
	child, cancel := c.withTimeoutContext(parent, 5*time.Second)
	defer cancel()

	parentCancel()

	select {
	case <-child.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected child context to follow caller cancellation")
	}
}

func TestWithTimeoutContextFallsBackToBrowserContext(t *testing.T) {
	t.Parallel()

	base, baseCancel := context.WithCancel(context.Background())
	defer baseCancel()

	c := &ChromeDP{
		ctx:  base,
		opts: DefaultOptions(),
	}

	child, cancel := c.withTimeoutContext(nil, 50*time.Millisecond)
	defer cancel()

	deadline, ok := child.Deadline()
	if !ok {
		t.Fatal("expected derived context to have a deadline")
	}

	if remaining := time.Until(deadline); remaining <= 0 || remaining > 100*time.Millisecond {
		t.Fatalf("expected deadline near requested timeout, got %v", remaining)
	}
}
