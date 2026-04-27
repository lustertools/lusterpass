# lusterpass

**Secrets that AI coding agents can use, but never see.**

![Agent-safe demo](demos/agent-safe/agent-safe-demo.gif)

Lusterpass is a CLI that loads secrets from [Bitwarden Secrets Manager](https://bitwarden.com/products/secrets-manager/) into a child process's environment without ever exposing the values to the agent driving the shell. It is built for the era of AI coding agents — Cline, Cursor, Aider, OpenClaw, Hermes, and any LLM-driven workflow that needs to run real commands against real systems.

```bash
# Your AI agent runs this:
eval "$(lusterpass env --profile dev)"
./run-migrations.sh

# What the agent sees in its transcript: nothing.
# What the migration script sees in its env: DATABASE_URL, API_KEY, ...
```

The agent never reads a `.env` file. The values never enter the LLM's context window, prompt cache, or telemetry pipeline. The only surface area the agent touches is the *names* of secrets — defined in a committed `.lusterpass.yaml` — and the `eval` line that hands them to the next process.

---

## Why this matters now

AI coding agents are now writing code that needs database passwords, API tokens, and OAuth secrets to actually work. The default workflow today is broken in two directions:

- **Agent reads `.env` directly** → secret values flow into the LLM's prompt, cache, and any vendor-side logging.
- **Agent has no secrets** → the agent can't run migrations, deploys, or integration tests, so a human babysits every command.

Lusterpass closes the gap: secrets reach the subprocess environment via shell `eval`, where they belong, while the agent's stdout/stderr stays clean.

This is not a new technique — `direnv export`, `op run`, `doppler run`, `chamber exec` all do the underlying mechanic. What lusterpass adds:

- **Bitwarden Secrets Manager backend** — open-source vault with a usable free tier ($0/mo: unlimited secrets, 2 users, 3 projects, 3 machine accounts). Self-hosting is available on the Enterprise plan.
- **Per-project, per-environment profiles** — `dev`, `staging`, `prod` defined in a committed YAML, with secret *names* (never values) tracked in git.
- **First-class agent-safe usage rules** shipped as an [agent skill](skills/lusterpass/SKILL.md) that any coding-agent runtime can adopt.
- **Encrypted local cache** — secrets fetched once, then served from `~/.lusterpass/cache/` so agents work offline and avoid rate limits.

---

## Quickstart

### 1. Install

```bash
curl -sSfL https://raw.githubusercontent.com/lustertools/lusterpass/main/install.sh | bash
```

Alternatives: `VERSION=v0.1.0 ...` for a specific version, `INSTALL_DIR=~/.local/bin ...` for a custom path.

### 2. Set up Bitwarden (one-time)

You need a [Bitwarden Secrets Manager](https://bitwarden.com/products/secrets-manager/) account and a machine access token. Free tier works. Step-by-step in [docs/bitwarden-setup.md](docs/bitwarden-setup.md).

```bash
lusterpass login
# Prompts for: Bitwarden access token, Organization ID
```

### 3. Define a project config

Drop a `.lusterpass.yaml` in your project root (or copy [`.lusterpass.yaml.example`](.lusterpass.yaml.example)):

```yaml
project: myapp

common:
  vars:
    APP_NAME: myapp
    LOG_FORMAT: json

profiles:
  dev:
    vars:
      LOG_LEVEL: debug
      APP_URL: http://localhost:3000
    secrets:
      DATABASE_URL: db-url--myapp--dev
      OPENAI_API_KEY: openai-key--myapp--dev
```

The right-hand side of `secrets:` is a **reference name** in Bitwarden — never a value.

### 4. Pull and use

```bash
lusterpass pull --profile dev          # fetch + encrypt locally
eval "$(lusterpass env --profile dev)" # load into current shell
```

### 5. Optional: integrate with direnv

```bash
echo 'eval "$(lusterpass env --profile dev)"' > .envrc
direnv allow
```

Now `cd`-ing into the project loads secrets automatically — into your shell, not your agent's transcript.

---

## How it stays agent-safe

| Surface | What lives here | Visible to the agent? |
|---|---|---|
| `.lusterpass.yaml` | Secret *names* + non-secret config vars | Yes — committed to git |
| Bitwarden vault | Secret *values* | No — agent never authenticates |
| `~/.lusterpass/cache/` | Encrypted blob of resolved values | No — encrypted with a local key |
| `lusterpass env` stdout | `export VAR=value` lines | **Yes if printed; no if `eval`'d** |
| Subprocess env | Resolved values | No — child process, not parent shell output |

The only sharp edge: if an agent runs `lusterpass env` *without* `eval`, the values land in stdout. The shipped agent skill enforces "always `eval`, never raw print." Other runtimes need an equivalent rule.

---

## Commands

```text
lusterpass login              # store token + org ID for an account
lusterpass account list       # multi-account: see all configured accounts
lusterpass account use <n>    # switch active account
lusterpass list               # show secret names (never values) in vault
lusterpass enrol              # add a new secret to Bitwarden
lusterpass pull --profile X   # fetch + cache secrets for a profile
lusterpass env --profile X    # emit export lines for `eval`
lusterpass migrate .envrc     # bootstrap config from existing .envrc
lusterpass test               # end-to-end test against your vault
```

Full per-command help: `lusterpass <command> --help`.

---

## What lusterpass is *not*

- **Not a Bitwarden replacement.** It uses Bitwarden Secrets Manager as the backend; you still need an account.
- **Not a DRM solution.** A determined adversary with shell access can still capture env vars from a live process. Lusterpass narrows the leak surface to the agent's transcript, which is the actual threat model in 2026.

---

## AI agent integration

The repo ships an [agent skill](skills/lusterpass/SKILL.md) that teaches a coding agent to:

- Use `eval "$(lusterpass env)"` instead of reading `.env`
- Never print, echo, or log resolved secret values
- Reference secrets as `$VAR_NAME` in generated scripts

To install: copy `skills/lusterpass/` into the skills/rules directory your coding agent reads from. The core invariant is one line — **always `eval`, never raw output** — and translates straightforwardly to any agent's rules format. PRs adding adapted skill files for specific runtimes (Cline, Cursor, Aider, OpenClaw, Hermes, …) are welcome.

---

## Roadmap

- Redacted-by-default subcommand output (mask any secret value that appears in stdout/stderr of wrapped commands)
- Local audit log of `pull` / `env` invocations with caller process attribution
- Process-attestation: only resolve secrets if the parent process matches an allowlist
- Skills for Cline, Cursor, Aider, OpenClaw, Hermes
- Backend adapters beyond Bitwarden (1Password, Vault, AWS Secrets Manager)

If any of these matter to you, open an issue or sponsor the project.

---

## Contributing

Bug reports, agent-runtime skills, and backend adapters are the highest-value contributions. See [docs/release-and-install.md](docs/release-and-install.md) for build details.

## Security disclaimer

Lusterpass *reduces* the surface area where secrets are exposed to AI agents and shell transcripts. It does **not** make secrets unreadable to a determined attacker who already has shell access, debugger access, or a malicious subprocess running under your user.

Specifically, lusterpass does **not** protect against:

- Reading `/proc/<pid>/environ` on Linux for any process you own
- A subprocess that intentionally logs, echoes, or transmits the env values it receives
- Core dumps, debuggers (lldb/gdb), or memory inspection of running processes
- An attacker who has already compromised your user account on the machine
- Misuse: running `lusterpass env` *without* `eval`, copying values into source code, or pasting them into a chat
- Vulnerabilities in Bitwarden Secrets Manager itself, which is the upstream source of truth

The encrypted local cache protects secrets at rest from casual disk inspection, not from an attacker with your filesystem credentials.

**Use at your own risk.** This software is provided "AS IS" under the [MIT License](LICENSE), with no warranty of any kind. The maintainers and contributors are not liable for any damages, data loss, security breaches, downtime, or financial harm arising from use, misuse, or inability to use this software. Review the source code, threat-model your environment, and validate against your organization's security policy before adopting in production-adjacent contexts.

## License

MIT — see [LICENSE](LICENSE).
