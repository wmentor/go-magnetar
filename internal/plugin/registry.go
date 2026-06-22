package plugin

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/alecthomas/kong"
)

// entry holds a registered plugin with its name.
type entry struct {
	name   string
	plugin Plugin
}

// hub is the concrete implementation of Hub. It accumulates registrations
// during Init calls and manages background goroutines.
type hub struct {
	mu sync.Mutex

	tools    []LLMTool
	commands []ChatCommand
	cli      []any

	goroutines  []func(ctx context.Context)
	initialised bool

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newHub() *hub {
	return &hub{}
}

func (h *hub) RegisterTool(tool LLMTool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tools = append(h.tools, tool)
}

func (h *hub) RegisterChatCommand(cmd ChatCommand) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commands = append(h.commands, cmd)
}

func (h *hub) RegisterCLICommand(cmd any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cli = append(h.cli, cmd)
}

// Go enqueues f to be started after all plugins have been initialised.
func (h *hub) Go(f func(ctx context.Context)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.goroutines = append(h.goroutines, f)
}

// start launches all queued goroutines under a single shared context.
// Must be called only after all Init calls have returned.
func (h *hub) start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	for _, f := range h.goroutines {
		h.wg.Add(1)
		go func(fn func(context.Context)) {
			defer h.wg.Done()
			fn(ctx)
		}(f)
	}
}

// Stop cancels all goroutine contexts and waits for them to finish.
func (h *hub) Stop() {
	h.mu.Lock()
	cancel := h.cancel
	h.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	h.wg.Wait()
}

func (h *hub) cliPlugins() kong.Plugins {
	h.mu.Lock()
	defer h.mu.Unlock()
	p := make(kong.Plugins, len(h.cli))
	copy(p, h.cli)
	return p
}

// Global registry — analogous to database/sql driver registry.
var (
	registryMu sync.Mutex
	registry   []entry
	globalHub  = newHub()
	globalRoot rootHolder
)

type rootHolder struct {
	mu   sync.RWMutex
	root *os.Root
}

func (r *rootHolder) set(root *os.Root) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.root = root
}

func (r *rootHolder) get() *os.Root {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.root
}

// Register records a plugin under the given name.
// Intended to be called from a plugin package's init() function.
// Panics if name is already registered.
func Register(name string, p Plugin) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, e := range registry {
		if e.name == name {
			panic(fmt.Sprintf("plugin: duplicate registration %q", name))
		}
	}
	registry = append(registry, entry{name: name, plugin: p})
}

// CLIPlugin is an optional interface for plugins that contribute CLI subcommands.
// RegisterCLI is called before kong.Parse so that commands are available for
// argument parsing. It must not depend on config (which is loaded after parsing).
type CLIPlugin interface {
	Plugin
	RegisterCLI(addCmd func(cmd any))
}

// KongPlugins collects CLI commands from all registered CLIPlugin implementations
// and returns them as a kong.Plugins slice for embedding in the root CLI struct.
// Called before kong.Parse, so before InitAll.
func KongPlugins() kong.Plugins {
	registryMu.Lock()
	entries := make([]entry, len(registry))
	copy(entries, registry)
	registryMu.Unlock()

	var cmds []any
	add := func(cmd any) { cmds = append(cmds, cmd) }
	for _, e := range entries {
		if cp, ok := e.plugin.(CLIPlugin); ok {
			cp.RegisterCLI(add)
		}
	}
	p := make(kong.Plugins, len(cmds))
	copy(p, cmds)
	return p
}

// InitAll initialises every registered plugin in registration order and then
// starts all goroutines that plugins queued via Hub.Go.
//
// Steps:
//  1. Init(s, hub) is called on every plugin.
//  2. All queued goroutines are launched.
//
// Must be called exactly once, after the config is loaded.
// Panics if called a second time.
func InitAll(s State) error {
	registryMu.Lock()
	if globalHub.initialised {
		registryMu.Unlock()
		panic("plugin: InitAll called more than once")
	}
	globalHub.initialised = true
	entries := make([]entry, len(registry))
	copy(entries, registry)
	registryMu.Unlock()

	for _, e := range entries {
		if err := e.plugin.Init(s, globalHub); err != nil {
			return fmt.Errorf("plugin %q: %w", e.name, err)
		}
	}

	globalHub.start()
	return nil
}

// LLMTools returns all LLM tools registered by plugins.
func LLMTools() []LLMTool {
	globalHub.mu.Lock()
	defer globalHub.mu.Unlock()
	out := make([]LLMTool, len(globalHub.tools))
	copy(out, globalHub.tools)
	return out
}

// ChatCommands returns all chat commands registered by plugins.
func ChatCommands() []ChatCommand {
	globalHub.mu.Lock()
	defer globalHub.mu.Unlock()
	out := make([]ChatCommand, len(globalHub.commands))
	copy(out, globalHub.commands)
	return out
}

// Stop cancels all managed goroutine contexts and waits for them to finish.
// Must be called before exit (e.g. via defer in Execute).
func Stop() {
	globalHub.Stop()
}

// SetRoot provides an *os.Root to plugins that need filesystem access.
// Called by the agent subcommand after opening its working directory.
// Plugins that require Root read it via GetRoot().
func SetRoot(root *os.Root) {
	globalRoot.set(root)
}

// GetRoot returns the current *os.Root, or nil if not yet set.
func GetRoot() *os.Root {
	return globalRoot.get()
}

// Reset clears all registered plugins and resets the hub to its initial state.
// Intended for use in tests only. Panics if InitAll has already been called.
func Reset() {
	registryMu.Lock()
	defer registryMu.Unlock()
	if globalHub.initialised {
		panic("plugin: Reset called after InitAll — not safe in production")
	}
	registry = nil
	globalHub = newHub()
	globalRoot = rootHolder{}
}
