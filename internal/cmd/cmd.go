package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/agent/chat"
	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
	skillplugin "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/skill"
	version "github.com/wmentor/go-magnetar/internal/plugins/chatcmd/version"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/response"
	"github.com/wmentor/go-magnetar/internal/skill"
	"github.com/wmentor/go-magnetar/internal/template"
)

type Globals struct {
	File     string `short:"f" type:"path" help:"Input file"`
	Profile  string `short:"p" help:"Profile name to use"`
	Skill    string `help:"Skill name to apply at startup"`
	Session  string `type:"path" help:"Load conversation session from file"`
	ReadOnly bool   `long:"read-only" help:"Run in read-only mode"`
	Timing   bool   `name:"enable-timing" help:"Show the execution time of every answer"`
}

type cli struct {
	Globals
}

func Execute() error {
	root := &cli{}
	kong.Parse(root,
		kong.Name("go-magnetar"),
		kong.Description("RAG indexer and AI chat agent"),
		kong.UsageOnError(),
		kong.Bind(&root.Globals),
	)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir error: %w", err)
	}
	configPath := filepath.Join(homeDir, ".go-magnetar", "config.yml")

	// Ensure config directory and sync templates
	if err := template.EnsureConfigDir(homeDir); err != nil {
		return fmt.Errorf("ensure config dir error: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if root.Profile != "" {
		_ = cfg.SetProfile(root.Profile)
	}

	printer.SetDefault(printer.New(os.Stdout, false))

	if root.File == "" {
		version.PrintVersion(cfg.ProfileParamString("llm.model"))
		printEnabledModules(printer.GetDefault(), cfg)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get work dir error: %w", err)
	}

	skillStore := skill.NewStore(
		filepath.Join(homeDir, ".go-magnetar", "skills"),
		filepath.Join(workDir, ".go-magnetar", "skills"),
	)

	responseStore := response.New(plugin.DefaultResponseStoreSize) // circular buffer with capacity DefaultResponseStoreSize

	state := &plugin.State{Config: cfg, ReadOnly: root.ReadOnly, SkillStore: skillStore, ResponseStore: responseStore}
	state.TimingEnabled.Store(root.Timing)

	if err := plugin.InitAll(state); err != nil {
		return err
	}

	defer plugin.Stop()

	rootFS, err := os.OpenRoot(workDir)
	if err != nil {
		return fmt.Errorf("open work dir error: %w", err)
	}
	defer rootFS.Close()

	// Provide the agent's working-directory root to plugins that need
	// filesystem access (generic file tools, web preprocessor).
	plugin.SetRoot(rootFS)

	printer.Verbose(cfg.Bool("verbose") && root.File == "")

	agent, err := chat.New(cfg, rootFS)
	if err != nil {
		return err
	}

	if root.Session != "" {
		if err := loadSession(agent, root.Session); err != nil {
			return fmt.Errorf("failed to load session: %w", err)
		}
		printer.ToolCall(printer.IconDone, "session was loaded from "+root.Session)
		printer.EmptyLine()
	}

	var skill *plugin.Skill

	// Validate skill if specified
	if root.Skill != "" {
		skill, err = skillStore.Get(context.Background(), root.Skill)
		if err != nil {
			return fmt.Errorf("failed to load skill '%s': %w", root.Skill, err)
		}
		if !skill.Enable {
			return fmt.Errorf("skill '%s' is not enabled", root.Skill)
		}
	}

	if root.File != "" {
		data, err := readFileInput(root.File, os.Stdin)
		if err != nil {
			return errors.Wrapf(err, "read file %s error", root.File)
		}

		txt := string(data)

		if skill != nil {
			txt = skillplugin.MakeApplySkillMessage(skill, txt)
		}

		for _, preprocessor := range plugin.Preprocessors() {
			if str, err := preprocessor(context.Background(), txt); err == nil {
				txt = str
			} else {
				printer.Error("chat: preprocessor error", "err", err)
			}
		}

		answer, err := agent.Ask(txt)
		if err != nil {
			return errors.Wrap(err, "agent error")
		}

		fmt.Println(answer)

		return nil
	}

	return agent.Run(state, root.Skill)
}

func readFileInput(filename string, stdin io.Reader) ([]byte, error) {
	if filename == "-" {
		return io.ReadAll(stdin)
	}

	return os.ReadFile(filename)
}

func loadSession(a *chat.ChatAgent, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var msgs []openai.ChatCompletionMessage
	if err := json.Unmarshal(data, &msgs); err != nil {
		return err
	}

	a.SetMessages(msgs)
	return nil
}

func printEnabledModules(prnt printer.Printer, cfg *config.Config) {
	has := false
	if cfg.String("confluence.base_url") != "" && cfg.Bool("confluence.enable") {
		prnt.Print(printer.IconModule, "confluence plugin is enabled")
		has = true
	}
	if cfg.String("gitlab.base_url") != "" && cfg.Bool("gitlab.enable") {
		prnt.Print(printer.IconModule, "gitlab plugin is enabled")
		has = true
	}
	if cfg.String("github.base_url") != "" && cfg.Bool("github.enable") {
		prnt.Print(printer.IconModule, "github plugin is enabled")
		has = true
	}
	if cfg.Bool("guard.enable") {
		prnt.Print(printer.IconModule, "guard plugin is enabled")
		has = true
	}
	if cfg.String("jira.base_url") != "" && cfg.Bool("jira.enable") {
		prnt.Print(printer.IconModule, "jira plugin is enabled")
		has = true
	}
	if cfg.String("rag.llm.base_url") != "" && cfg.Bool("rag.enable") {
		prnt.Print(printer.IconModule, "rag plugin is enabled")
		has = true
	}
	if cfg.Bool("ssh.enable") {
		prnt.Print(printer.IconModule, "ssh plugin is enabled")
		has = true
	}
	if has {
		prnt.EmptyLine()
	}
}
