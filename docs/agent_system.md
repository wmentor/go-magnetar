# Agent system

## Overview

The agent system in go-magnetar is built around a **single central ChatAgent** that manages user interaction through REPL, and **specialized Subagents** that can be invoked to perform specific tasks.

Key components:

```
go-magnetar/
├── ChatAgent (internal/agent/chat/agent.go)
│   └── Main REPL agent handling user input
├── Subagent (internal/agent/subagent/agent.go)
│   └── Specialized agents invoked via tool or /agent command
├── Store (internal/agent/store.go)
│   └── Agent repository loaded from .md files
├── Guard (internal/agent/guard/agent.go)
│   └── Security for exec commands
├── Indexer (internal/agent/indexer/indexer.go)
│   └── Document indexing agent for RAG
└── Summarizer (internal/agent/summarizer/summarizer.go)
    └── Conversation history compression
```

## Types of Agents

### 1. ChatAgent (`internal/agent/chat/agent.go:113`)

- **Main REPL agent**
- Manages conversations, processes instructions, calls tools
- Uses system prompt with project context templates (`AGENTS.md`, `CLAUDE.md`)
- Supports: web search, RAG search, file operations, exec commands, history compression

### 2. Subagent (`internal/agent/subagent/agent.go:35`)

- **Specialized agents** defined in `.md` files
- Have their own prompts and tool sets
- Invoked through:
  - `agent_<name>` tools (automatically when LLM determines a specialized agent is needed)
  - `/agent <name> [request]` (manual invocation)
- Support recursion (one subagent can call another)

### 3. Guard Agent (`internal/agent/guard/agent.go:35`)

- **Security for exec commands**
- Analyzes commands for dangerous patterns (bash pipes, deletions, git mutations)
- Returns `allowed: true/false` with reason

### 4. Indexer Agent (`internal/agent/indexer/indexer.go:15`)

- **Document indexing to RAG**
- Loads files, URLs, splits into chunks
- Saves embeddings to Chromem/Qdrant
- Used by `/index` command

### 5. Summarizer (`internal/agent/summarizer/summarizer.go:21`)

- **Conversation history compression**
- Two modes: `soft` (keeps recent messages) and `full` (summarizes everything)
- Invoked by `/compact` command

## Agent Store

### Storage Location

Agents are stored in two directories:
- **User agents**: `~/.go-magnetar/agents/` — global agents
- **Project agents**: `{current_dir}/.go-magnetar/agents/` — project-specific agents (takes priority)

### File Format

Agents are stored as Markdown files with YAML frontmatter:

```markdown
---
name: security-analyzer
description: Analyzes GitHub repositories for security issues
profile: default
tools:
  - github_repo
  - github_issue
  - github_pr
disable: false
subagents:
  - code-reviewer
---
You are a security expert. Always use search tools first, then delegate to subagents.
```

**YAML Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique agent name |
| `description` | string | Agent description |
| `profile` | string | Profile name for agent-specific configuration (optional) |
| `tools` | []string | List of available tools |
| `disable` | bool | Disable the agent |
| `subagents` | []string | List of subagents this agent can delegate to |

### Loading Process

1. `AgentStore` recursively scans `~/.go-magnetar/agents` and project-specific directories
2. Parses `.md` files: YAML frontmatter + content
3. Stores in `map[string]*plugin.Agent`
4. Accessible via `state.AgentStore.Get(agentName)`

## Creating Agents

### Step-by-Step Guide

1. **Create agent file** in `~/.go-magnetar/agents/` or `{project}/.go-magnetar/agents/`

2. **Define YAML frontmatter** with agent metadata

3. **Write agent prompt** in the content section

4. **Register tools** in the `tools` list (must match available tool names)

5. **Define subagents** in the `subagents` list (optional)

### Example Agent

```markdown
---
name: security-analyzer
description: Analyzes GitHub repositories for security vulnerabilities
profile: production
tools:
  - github_repo
  - github_issue
  - github_pull_request
  - web_search
disable: false
subagents:
  - code-reviewer
---
You are a security expert. Analyze repositories for vulnerabilities and security issues.

When analyzing a GitHub repository:
1. Use web_search to gather general information about security best practices
2. Use github_repo to fetch repository information
3. Use github_issue and github_pull_request to analyze security-related issues
4. If code review is needed, delegate to code-reviewer subagent
5. Provide comprehensive security analysis with actionable recommendations
```

### Profile Configuration

Agents can be configured with different profiles to customize their behavior. The `profile` field in the YAML frontmatter specifies which profile configuration to use:

```markdown
---
name: security-analyzer
description: Analyzes GitHub repositories for security vulnerabilities
profile: production
tools:
  - github_repo
  - github_issue
  - web_search
disable: false
---
```

Profile-specific settings are loaded from the configuration file under `profiles.{profile_name}.` section. This allows agents to use different LLM models, temperature settings, or other parameters based on the profile.

## Calling Agents

### Method 1: Through `agent_<name>` tools (automatic)

The LLM automatically invokes specialized agents when appropriate:

```
User: Can you analyze this GitHub repository for security issues?
Agent: [Uses agent_security_analyzer tool to delegate to security-analyzer agent]
```

**Mechanism:**
1. At startup, go-magnetar generates a dedicated tool for each available agent (e.g., `agent_security_analyzer`)
2. LLM sees these tools in the tool schema with agent descriptions
3. Calls the appropriate tool (e.g., `agent_security_analyzer(message="analyze security issues")`)
4. The subagent uses its tools (`github_repo`, `github_issue`)
5. Returns result to main agent

### Method 2: Through `/agent` command (manual)

```bash
# Invoking subagent manually
> /agent security-analyzer analyze this repo for vulnerabilities

# Subagents can call other subagents
> /agent code-reviewer review the changes in this PR
```

**Command code:**
```go
// internal/plugins/chatcmd/agent/plugin.go:56
sa, err := subagent.New(agentName, p.state)
response, err = sa.Ask(request)
```

### Method 3: Recursive agent chains

```yaml
# security-analyzer.md
subagents:
  - code-reviewer

# code-reviewer.md
subagents:
  - unit-test-generator
```

**Call chain:**
```
User → security-analyzer → code-reviewer → unit-test-generator
```

### Method 4: Programmatic API

```go
// Get agent
agent, err := state.AgentStore.Get("security-analyzer")

// Check profile
profile := agent.Profile // "production"

// Check subagents
subagents := agent.Subagents // ["code-reviewer"]

// Get available tools
tools := agent.AvailableTools // ["github_repo", "github_issue"]

// Create deep copy
agentCopy := agent.Copy()
```

## Architecture Patterns

### 1. Strategy Pattern — Compression Modes

```go
// internal/agent/summarizer/summarizer.go:72
func (s *Summarizer) Compact(messages []openai.ChatCompletionMessage, mode ...string) {
    switch effectiveMode {
    case ModeCompact: return s.compactFull(messages)
    case ModeSoft:    return s.compactSoft(messages)
    }
}
```

### 2. Factory Pattern — Agent Creation

```go
// internal/agent/subagent/agent.go:47
func New(agentName string, state *plugin.State) (*Agent, error) {
    a, err := state.AgentStore.Get(agentName)
    // ...
}
```

### 3. Adapter Pattern — AgentHandle

```go
// internal/plugin/plugin.go:120
type AgentHandle interface {
    Messages() []openai.ChatCompletionMessage
    SetMessages([]openai.ChatCompletionMessage)
    Config() *config.Config
    // ...
}

// internal/agent/chat/agent.go:179
type agentHandle struct {
    a *ChatAgent
}
```

### 4. Composite Pattern — Agent Hierarchy

```go
// internal/plugin/agent.go:14
type Agent struct {
    Name           string
    Description    string
    Prompt         string
    AvailableTools []string
    Disable        bool
    Subagents      []string // recursive structure
}
```

### 5. Repository Pattern — AgentStore

```go
// internal/agent/store.go:28
type Store struct {
    dirs   []string
    agents map[string]*plugin.Agent
}

func (st *Store) Get(agentName string) (*plugin.Agent, error)
func (st *Store) List() []*plugin.Agent
func (st *Store) Filter(agents []string) (plugin.AgentStore, error)
```

## Best Practices

### Naming Conventions
- Use lowercase with hyphens: `security-analyzer`, `code-reviewer`
- Make names descriptive and specific
- Avoid generic names like `helper` or `assistant`

### Tool Selection
- Only include tools the agent actually needs
- Document which tools are essential vs optional
- Avoid tool overload — 3-5 tools is ideal

### Subagent Chains
- Keep chains shallow (2-3 levels max)
- Document delegation logic in agent descriptions
- Use clear, descriptive agent names for chain steps

### Error Handling
- Agents should handle tool failures gracefully
- Provide fallback strategies when tools fail
- Log errors for debugging

## Examples

### Security Analyzer

```markdown
---
name: security-analyzer
description: Analyzes repositories for security vulnerabilities
profile: default
tools:
  - github_repo
  - github_issue
  - web_search
disable: false
---
You are a security expert. Analyze repositories for vulnerabilities and provide recommendations.
```

### Code Reviewer

```markdown
---
name: code-reviewer
description: Performs code reviews and suggests improvements
profile: default
tools:
  - github_file
  - github_pull_request
  - web_search
disable: false
---
You are a code review expert. Analyze code changes and suggest improvements for security, performance, and maintainability.
```

### Unit Test Generator

```markdown
---
name: unit-test-generator
description: Generates unit tests for code
profile: default
tools:
  - github_file
  - file_write
  - web_search
disable: false
---
You are a test generation expert. Create comprehensive unit tests for the provided code.
```

## Agent Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                       ChatAgent (REPL)                       │
│  - User input                                                │
│  - Tool dispatch (web_search, rag_search, file_write, ...) │
│  - LLM loop with tool calls                                  │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
   ┌───────────┐  ┌───────────┐  ┌──────────────┐
   │  Agent    │  │  Guard    │  │   Summarizer │
   │  Tool     │  │  Agent    │  │     (Compact)│
   └─────┬─────┘  └───────────┘  └──────────────┘
         │
         ▼
   ┌──────────────┐
   │  Subagent    │
   │  Agent       │
   └──────┬───────┘
          │
   ┌──────┴───────┐
   ▼              ▼
┌────────┐   ┌──────────┐
│ Tools  │   │ Subagents│
└────────┘   └──────────┘
```

## Summary

The agent system in go-magnetar provides a flexible, extensible architecture for specialized AI tasks. By combining:

1. **Centralized ChatAgent** for general-purpose interaction
2. **Specialized Subagents** for specific domains
3. **Automated delegation** via LLM tool calling
4. **Manual invocation** via `/agent` command
5. **Recursive chaining** for complex workflows

The system enables powerful, modular AI workflows while maintaining clean separation of concerns and reusability.
