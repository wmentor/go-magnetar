package agent

import "errors"

// ErrInfiniteLoop is returned when a subagent detects an infinite loop.
var ErrInfiniteLoop = errors.New("infinite loop detected")
