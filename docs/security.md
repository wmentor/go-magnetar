# Security restrictions

## Command safety guard

All commands executed via the `exec` tool are analyzed by a built-in security guard before execution. The guard uses an LLM to evaluate whether a command is safe based on several criteria:

- **Destructive commands**: `rm -rf /`, `sudo`, `mkfs`, `dd` with `/dev/`, `fdisk`, `chmod 777`
- **Shell pipes**: `| bash`, `| sh`, `| zsh`
- **Git modifications**: `git commit`, `push`, `rebase`, `pull`, `cherry-pick`, `reset`, `stash`, `clean`, `reflog`, `clone`
- **System-level operations**: `su`, `chmod`, `chown`, `dscl`, `fdisk`, `format`, `dseditgroup`, `brew`, `dpkg`, `apt`, `cargo`, `rpm`, `npm`, `apt-get`, `groupadd`, `usermod`, `gpasswd`, `useradd`, `adduser`, `userdel`, `deluser`, `passwd`, `systemctl`, `sysadminctl`
- **Untrusted script execution**: `curl ... | bash`, `wget ... | sh`
- **Environment variable filtering**: Sensitive environment variables containing patterns like `PASS`, `SEC`, `CRED`, `TOKEN`, `KEY`, `AUTH`, `PWD`, `CERT`, `SIGN`, `SALT`, `BEARER`, `AMQP`, `CONNECT` are filtered out before command execution

When the system is in read-only mode, **NO modification operations are allowed** — all such commands are automatically rejected.

## Read-only mode

The read-only mode can be toggled via the `/readonly` chat command. In this mode:
- File writes (`file_write`) are blocked
- Command execution that modifies files or system state is blocked
- Only read-only operations are permitted

No output is ever displayed from blocked commands — they return an error message instead.

## Root user prevention

The application **cannot be run as root user**. If the current user is `root` (username or UID 0), the application prints an error message and exits immediately to prevent accidental system-wide modifications.
