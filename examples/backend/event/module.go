// Package event wires the backend example event bus and subscriptions.
package event

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DaiYuANg/arcgo/eventx"
	"github.com/arcgolabs/arcgo/dix"
)

// Module wires the backend example event bus and user event subscriptions.
var Module = dix.NewModule("event",
	dix.WithModuleProviders(
		dix.Provider0(func() eventx.BusRuntime {
			return eventx.New(
				eventx.WithAntsPool(4),
				eventx.WithParallelDispatch(true),
			)
		}),
	),
	dix.WithModuleSetup(func(c *dix.Container, lc dix.Lifecycle) error {
		bus, err := dix.ResolveAs[eventx.BusRuntime](c)
		if err != nil {
			return fmt.Errorf("resolve event bus: %w", err)
		}

		log, err := dix.ResolveAs[*slog.Logger](c)
		if err != nil {
			return fmt.Errorf("resolve logger: %w", err)
		}

		unsubscribe, err := eventx.Subscribe[UserCreatedEvent](bus, func(_ context.Context, e UserCreatedEvent) error {
			log.Info("user created (event)",
				slog.Int64("user_id", e.UserID),
				slog.String("name", e.UserName),
				slog.String("email", e.Email),
			)
			return nil
		})
		if err != nil {
			return fmt.Errorf("subscribe user created event: %w", err)
		}

		lc.OnStop(func(_ context.Context) error {
			unsubscribe()
			return bus.Close()
		})
		return nil
	}),
)
