package dix

import (
	"context"
	"time"
)

func (c *Container) logServiceResolution(ctx context.Context, name, op string, duration time.Duration, err error) {
	if c == nil || !c.resolutionLoggingEnabled {
		return
	}
	event := ResolveEvent{
		Meta:      c.meta,
		Profile:   c.profile,
		Service:   name,
		Operation: op,
		Duration:  duration,
		Err:       err,
	}
	if c.emitResolveEvent != nil {
		c.emitResolveEvent(ctx, event)
		return
	}
	emitEventLogger(ctx, c.eventLogger, event)
}
