package versionplugin

import (
	"context"
	"fmt"
	"regexp"
	"runtime/debug"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("chatcmd.version", &Plugin{})
}

var (
	template = `✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦

                           🌌  GO-MAGNETAR  🌌

✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦

     💫  Version:      %s
     👨  Commander:    Mikhail Kirillov (wmentor) 
     🌐  Repository:   github.com/wmentor/go-magnetar
     ✨  Engine:       Retrieval-Augmented Generation Engine
     🤖  LLM:          %s
     🪐  License:      MIT
     🛰️  Orbit:        %s
     🌌  Made with:    💖 among the stars

✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦  ·  ✧  ·  ✦

`

	reStable = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
)

// Plugin registers the /version chat command.
type Plugin struct{}

func (p *Plugin) Init(st *plugin.State, hub plugin.Hub) error {
	hub.RegisterChatCommand(plugin.ChatCommand{
		Name: "version",
		Help: "Show the current program version.",
		Execute: func(_ context.Context, a plugin.AgentHandle, args string) error {
			PrintVersion(st.Config.String("llm.model"))
			return nil
		},
	})
	return nil
}

func PrintVersion(model string) {
	version := "devel"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}

	stable := "Stable"
	if !reStable.MatchString(version) {
		stable = "Unstable"
	}

	fmt.Printf(template, version, model, stable)
}
