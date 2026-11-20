package processor

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

const (
	defaultAllocSize = 8
)

type store struct {
	mtx        sync.RWMutex
	processors []plugin.WebProcessorFunc
}

func NewStore() plugin.WebProcessorStore {
	return &store{
		processors: make([]plugin.WebProcessorFunc, 0, defaultAllocSize),
	}
}

func (st *store) AddProcessorFunc(fn plugin.WebProcessorFunc) {
	st.mtx.Lock()
	defer st.mtx.Unlock()

	if fn != nil {
		st.processors = append(st.processors, fn)
	}
}

func (st *store) Process(ctx context.Context, url string) (string, error) {
	st.mtx.RLock()
	defer st.mtx.RUnlock()

	for _, fn := range st.processors {
		ret, err := fn(ctx, url)
		if err != nil {
			if errors.Is(err, plugin.ErrNoWebProcessor) {
				continue
			}
			return "", fmt.Errorf("process URL %q error: %w", url, err)
		}
		return ret, nil
	}

	return "", plugin.ErrNoWebProcessor
}
