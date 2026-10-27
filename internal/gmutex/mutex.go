package gmutex

import (
	"sync"
)

var mtx sync.RWMutex

func Lock() {
	mtx.Lock()
}

func Unlock() {
	mtx.Unlock()
}

func RLock() {
	mtx.RLock()
}

func RUnlock() {
	mtx.RUnlock()
}

func TryLock() bool {
	return mtx.TryLock()
}

func TryRLock() bool {
	return mtx.TryRLock()
}
