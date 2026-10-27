package ssh

import "errors"

// ErrInvalidParams is returned when invalid parameters are provided to the tool.
var ErrInvalidParams = errors.New("invalid params")

// ErrSSHDisabled is returned when SSH is disabled in config.
var ErrSSHDisabled = errors.New("ssh disabled")

// ErrNoAddress is returned when no address is provided.
var ErrNoAddress = errors.New("no address provided")

// ErrNoAuthMethod is returned when no authentication method is configured.
var ErrNoAuthMethod = errors.New("no authentication method configured")
