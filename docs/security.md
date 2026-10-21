# Security

See also: [docs/data_residency.md](./data_residency.md) for information about local data processing and zero-data-exfiltration configurations.

## Command safety guard

All commands executed via the `exec` tool are analyzed by a built-in security guard before execution. The guard uses an LLM to evaluate whether a command is safe based on several criteria:

- **Destructive commands**: `rm -rf /`, `sudo`, `mkfs`, `dd` with `/dev/`, `fdisk`, `chmod 777`
- **Shell pipes**: `| bash`, `| sh`, `| zsh`
- **Git modifications**: `git commit`, `push`, `rebase`, `pull`, `cherry-pick`, `reset`, `stash`, `clean`, `reflog`, `clone`
- **System-level operations**: `su`, `chmod`, `chown`, `dscl`, `fdisk`, `format`, `dseditgroup`, `brew`, `dpkg`, `apt`, `cargo`, `rpm`, `npm`, `apt-get`, `groupadd`, `usermod`, `gpasswd`, `useradd`, `adduser`, `userdel`, `deluser`, `passwd`, `systemctl`, `sysadminctl`
- **Untrusted script execution**: `curl ... | bash`, `wget ... | sh`
- **Environment variable filtering**: Sensitive environment variables containing patterns like `PASS`, `SEC`, `CRED`, `TOKEN`, `KEY`, `AUTH`, `PWD`, `CERT`, `SIGN`, `SALT`, `BEARER`, `AMQP`, `CONNECT` are filtered out before command execution

## Allowed utilities (white list)

You can configure a white list of trusted utilities using the `guard.allow` parameter:

```yaml
guard:
  enable: true
  allow: ["ls", "pwd"]  # trusted utilities
```

When `guard.allow` is set, the guard agent will consider commands using only utilities from this list as safe, **provided they don't use dangerous patterns** (e.g., `ls | bash`, `pwd > /tmp/file`).

**Important notes:**
- Even allowed utilities become dangerous when combined with destructive patterns
- The white list is passed to the guard agent's system prompt to inform its decision-making
- If the white list is empty or not configured, all commands are evaluated based on the security criteria above
- The white list does not bypass the guard check; it only helps the LLM identify safe commands more accurately

See [Configuration > Guard](./configuration.md#guard-agent) for more details on the `guard.allow` parameter.

When the system is in read-only mode, **NO modification operations are allowed** — all such commands are automatically rejected.

## Extra protection

For enhanced security, certain plugins provide additional protection mechanisms beyond the global read-only mode:

### Confluence plugin: read-only mode

The Confluence plugin supports a `readonly` configuration parameter that completely disables Confluence page editing:

```yaml
confluence:
  enable: true
  readonly: true  # Disables confluence_page_edit tool registration
  base_url: https://your-domain.atlassian.net
  api_key: ""  # API key not required in read-only mode
```

When `confluence.readonly` is set to `true`:
- The `confluence_page_edit` tool is **not registered**, making it impossible for the agent to invoke it
- This setting persists across sessions and cannot be overridden at runtime
- Useful for production systems or audit scenarios where Confluence editing should never be allowed

Note: If the Confluence plugin is enabled without the `readonly` flag, the API key is required for the `confluence_page_edit` tool to function.

## Root user prevention

The application **cannot be run as root user**. If the current user is `root` (username or UID 0), the application prints an error message and exits immediately to prevent accidental system-wide modifications.

## Data Residency

See [docs/data_residency.md](./data_residency.md) for complete information about local data processing, Chromem architecture, and privacy guarantees.
