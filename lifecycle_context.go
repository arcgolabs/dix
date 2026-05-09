package dix

import (
	"context"
	"time"
)

func lifecycleHookContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx = contextOrBackground(ctx)
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
