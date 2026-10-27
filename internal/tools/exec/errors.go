package exec

import "errors"

// ErrInvalidParams is returned when invalid parameters are provided to the tool.
var ErrInvalidParams = errors.New("invalid params")

// ErrSecurityCheckFailed is returned when security check fails.
var ErrSecurityCheckFailed = errors.New("security check failed")

// ErrCommandBlocked is returned when a command is blocked by security patterns.
var ErrCommandBlocked = errors.New("command blocked by security patterns")
