package plugin

import (
	"context"
)

type MaskService interface {
	Process(ctx context.Context, input string) string
}
