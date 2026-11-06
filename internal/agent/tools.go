package agent

import (
	"context"
	"strings"
	"sync/atomic"
	"time"
)

func WaitCanRun(ctx context.Context, counter *atomic.Int64, parallelMode bool) error {
	if parallelMode {
		return nil
	}
	for {
		// Check context before modifying counter
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		val := counter.Add(1)
		if val == 1 {
			return nil
		}
		counter.Add(-1)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

// NormalizeAgentName converts agent name to valid tool name format.
// Replaces hyphens with underscores for OpenAI tool naming compatibility.
func NormalizeAgentName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}
