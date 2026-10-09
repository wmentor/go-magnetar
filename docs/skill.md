# Skills

Skills provide specialized instructions and workflows for specific tasks. go-magnetar loads skills from two locations:

1. **User skills**: `~/.go-magnetar/skills`
2. **Project skills**: `{current_dir}/.go-magnetar/skills`

Project skills have priority over user skills. If a skill with the same name exists in both locations, the project skill will be used.

## Skill Format

Each skill is stored in a directory containing a `SKILL.md` file with the following structure:

```markdown
---
name: skill-name
description: Skill description
disable: false
---
Skill content goes here...
```

### Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Unique skill identifier |
| `description` | string | Yes | Brief description of what the skill does |
| `disable` | boolean | No | Set to `true` to disable the skill (default: `false`) |

## Examples

### Basic Skill

```
my-skill/
└── SKILL.md
```

```markdown
---
name: go-module-help
description: Helper for Go module operations
disable: false
---
When working with Go modules, always run `go mod tidy` after adding or removing imports...
```

### Disabled Skill

```markdown
---
name: legacy-workflow
description: Old workflow instructions
disable: true
---
These instructions are outdated and should not be used...
```

## Directory Structure

Skills are loaded from `.go-magnetar/skills` directories. Hidden directories (starting with `.`) are skipped.

```
~/.go-magnetar/skills/
└── go-mod/
    └── SKILL.md
└── docker-help/
    └── SKILL.md

project/.go-magnetar/skills/
└── project-specific/
    └── SKILL.md
```

## Skill Priority

When the same skill exists in multiple locations:

1. **Project skills** (current directory) have highest priority
2. **User skills** (home directory) are loaded as fallback

This allows overriding user-level skills with project-specific ones.

