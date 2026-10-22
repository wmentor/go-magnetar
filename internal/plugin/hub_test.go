package plugin

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestHub_NewHubIsEmpty(t *testing.T) {
	t.Parallel()

	h := newHub()
	if len(h.tools) != 0 || len(h.commands) != 0 || len(h.preprocessors) != 0 || len(h.goroutines) != 0 {
		t.Fatalf("a new hub must hold no registrations, got %+v", h)
	}
	if h.initialised || h.cancel != nil {
		t.Fatalf("a new hub must not be initialised or started, got %+v", h)
	}
}

func TestHub_RegistrationsKeepTheirOrder(t *testing.T) {
	t.Parallel()

	h := newHub()
	for _, name := range []string{"stat", "help", "exit"} {
		h.RegisterChatCommand(ChatCommand{Name: name})
		h.RegisterTool(LLMTool{IsSearchTool: name == "help"})
		h.RegisterPreprocessor(func(_ context.Context, text string) (string, error) { return name + ":" + text, nil })
	}

	if got := []string{h.commands[0].Name, h.commands[1].Name, h.commands[2].Name}; fmt.Sprint(got) != "[stat help exit]" {
		t.Errorf("commands in registration order: got %v", got)
	}
	if len(h.tools) != 3 || !h.tools[1].IsSearchTool || h.tools[0].IsSearchTool {
		t.Errorf("tools in registration order: got %+v", h.tools)
	}
	var got []string
	for _, pre := range h.preprocessors {
		out, err := pre(context.Background(), "x")
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, out)
	}
	if fmt.Sprint(got) != "[stat:x help:x exit:x]" {
		t.Errorf("preprocessors in registration order: got %v", got)
	}
}

func TestHub_ConcurrentRegistrationsAreAllKept(t *testing.T) {
	t.Parallel()

	h := newHub()
	const n = 50
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			h.RegisterChatCommand(ChatCommand{Name: fmt.Sprint(i)})
			h.RegisterTool(LLMTool{})
			h.RegisterPreprocessor(func(_ context.Context, s string) (string, error) { return s, nil })
			h.Go(func(context.Context) {})
		})
	}
	wg.Wait()

	if len(h.commands) != n || len(h.tools) != n || len(h.preprocessors) != n || len(h.goroutines) != n {
		t.Fatalf("expected %d of each, got commands=%d tools=%d preprocessors=%d goroutines=%d",
			n, len(h.commands), len(h.tools), len(h.preprocessors), len(h.goroutines))
	}
}

func TestHub_GoWaitsForStartThenStopCancelsAndWaits(t *testing.T) {
	t.Parallel()

	h := newHub()
	const n = 3
	started := make(chan struct{}, n)
	var finished sync.WaitGroup
	finished.Add(n)
	for range n {
		h.Go(func(ctx context.Context) {
			defer finished.Done()
			started <- struct{}{}
			<-ctx.Done()
		})
	}

	select {
	case <-started:
		t.Fatal("a goroutine queued with Go must not run before start")
	case <-time.After(20 * time.Millisecond):
	}

	h.start()
	for range n {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("start must run every queued goroutine")
		}
	}

	done := make(chan struct{})
	go func() {
		h.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop must cancel the shared context and return once the goroutines finish")
	}
	finished.Wait()
}

func TestHub_StopWaitsForAGoroutineToReturn(t *testing.T) {
	t.Parallel()

	h := newHub()
	release := make(chan struct{})
	var returned bool
	var mu sync.Mutex
	h.Go(func(ctx context.Context) {
		<-ctx.Done()
		<-release // still busy after the cancel
		mu.Lock()
		returned = true
		mu.Unlock()
	})
	h.start()

	done := make(chan struct{})
	go func() {
		h.Stop()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("Stop returned while a goroutine was still running")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	<-done
	mu.Lock()
	defer mu.Unlock()
	if !returned {
		t.Fatal("Stop returned before the goroutine did")
	}
}

func TestHub_StopBeforeStartOrWithNothingQueued(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		start bool
	}{
		{"never started", false},
		{"started with no goroutines", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newHub()
			if tt.start {
				h.start()
			}
			done := make(chan struct{})
			go func() {
				h.Stop()
				h.Stop() // a second Stop is harmless too
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("Stop must return at once when nothing runs")
			}
		})
	}
}
