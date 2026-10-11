# Skills

Skills provide specialized instructions and workflows for specific tasks. go-magnetar loads skills from two locations:

1. **User skills**: `~/.go-magnetar/skills`
2. **Project skills**: `{current_dir}/.go-magnetar/skills`

Project skills have priority over user skills. If a skill with the same name exists in both locations, the project skill will be used.

## Skill Format

Each skill is stored in a directory containing a `SKILL.md` file with the following structure:

**IMPORTANT LIMITATION (Current Version):**
- Skills currently only support the `SKILL.md` file
- Supporting files and scripts are not yet loaded or executed
- If a skill directory contains additional files (e.g., scripts, templates, data files), they are ignored
- This is a known limitation and support for additional files will be added in future releases

```markdown
my-skill/
├── SKILL.md           ← Only supported file currently
└── auxiliary.txt      ← Currently ignored (will be supported in future)
```

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

## Skill Content Structure

The skill content is divided into several sections:

### Fields Section
The YAML frontmatter at the top of the file defines skill metadata:

```markdown
---
name: skill-name
description: Skill description
disable: false
---
```

### Main Instructions
The primary instructions for the skill, explaining what the skill does and how to use it.

### Important Rules
Mandatory rules that must be followed. Use bold formatting for emphasis:
- **YOU ARE PROHIBITED FROM** ...
- **YOU MUST** ...

### Critical Restrictions
Actions that are strictly forbidden:
- **YOU ARE PROHIBITED FROM MODIFYING ANY FILES** in the project directory
- Your only responsibility is to formulate the issue content in Markdown format
- Do not run any commands that would modify the filesystem

### Prohibited Tools
Skills can specify which tools are forbidden to use:
- FORBIDDEN from using the *search* tool
- FORBIDDEN from using any GitHub-related tools (e.g., github_*)

### Templates
Skills can provide templates to guide output format:
- Bug Report Template
- Feature Request Template

### Output Format
Specifies the exact format requirements for the skill output:
- Begin with the appropriate headers
- Use proper Markdown formatting throughout
- Include code blocks for YAML, JSON, or logs
- **ALL ISSUES MUST START WITH "Task: <title issue>" PREFIX**

### Workflow
Step-by-step instructions for the skill execution:
1. Read the user's request carefully
2. If details are unclear, ask specific questions to clarify before finalizing
3. Determine whether it's a bug or feature request
4. Apply the appropriate template structure
5. Fill in all relevant sections with detailed information
6. Review for completeness and clarity
7. Output the final Markdown issue

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

## Skill Priority and Conflict Resolution

When skills with the same name exist in multiple locations, **project skills always take precedence over user skills**.

### Conflict Resolution Rules

1. **Project skills have absolute priority** — if a skill exists in `{current_dir}/.go-magnetar/skills`, it will be used regardless of whether the same skill exists in `~/.go-magnetar/skills`

2. **User skills serve as fallback** — if a skill is not found in the project directory, the system falls back to user-level skills

3. **No merging occurs** — when a conflict arises, the entire project skill replaces the user skill; partial merges or combinations are not supported

4. **Explicit disabling override** — a project skill can disable a user skill by setting `disable: true` in its frontmatter, effectively making the skill unavailable even if the user skill is enabled

This priority system ensures that project-specific skills always override general-purpose user skills, providing a clean way to customize behavior per project while maintaining user-level defaults.

## Using Skills

Skills are activated via the `/skill` chat command in the interactive REPL.

### List Available Skills

```
> /skill
Available skills:
  * coding - Programming and code assistance
  * writing - Writing and content creation
```

### Load a Skill

```
> /skill coding
Loaded skill: coding
```

### Load a Skill with Context

```
> /skill writing "Help me write a blog post"
Loaded skill: writing
```

The skill content is applied as system instructions to the model, and the conversation continues with the new context.

See [docs/chat_command.md](./chat_command.md) for complete command documentation.

## License

