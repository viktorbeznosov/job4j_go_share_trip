package jobs

import (
	"context"
	"time"
)

func Run(ctx context.Context, interval time.Duration, tick func(context.Context) error) error {
	if interval <= 0 {
		interval = time.Minute
	}

	_ = tick(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = tick(ctx)
		}
	}
}
