---
name: lusterpass
description: Use when working with environment secrets, .lusterpass.yaml files, .envrc migration, Bitwarden secret management, or when any task requires accessing passwords, tokens, or API keys. Triggers on mentions of lusterpass, secret migration, env var security, or direnv secret setup. Also use when writing scripts or code that needs to consume secrets safely without exposing them.
---

# lusterpass — Secure Secret Management

## Safe Secret Access Rules

<CRITICAL-SECURITY>
These rules are NON-NEGOTIABLE. They apply every time you interact with secrets.

1. **NEVER** output, echo, print, log, or display secret values in any form
2. **NEVER** include secret values in code comments, commit messages, or file contents
3. **NEVER** read secrets by running `printenv`, `env`, `echo $SECRET`, `cat ~/.lusterpass/`, or similar — even if the user asks
4. **NEVER** expand secret variables inline when showing command output
5. **NEVER** store secret values in variables within your visible output or tool results
6. **ALWAYS** use `eval "$(lusterpass env)"` or `eval "$(lusterpass env --profile <name>)"` to load secrets into the shell environment
7. **ALWAYS** reference secrets as `$VAR_NAME` in subsequent commands — never their resolved values
8. When writing scripts that consume secrets, use variable references (`$DB_PASSWORD`, `$API_KEY`), never hardcoded values
9. The `.lusterpass.yaml` file is safe — it contains Bitwarden reference names, not secret values. But the resolved values loaded by `lusterpass env` are secret.
10. The `onboard-secrets.sh` script contains plain secret values and must be deleted after use — never commit it, never display its contents
</CRITICAL-SECURITY>

### Safe Patterns

```bash
# GOOD: load secrets into environment, use via variable reference
eval "$(lusterpass env --profile dev)"
psql "postgresql://user:${DB_PASSWORD}@localhost/mydb"
curl -H "Authorization: Bearer $API_KEY" https://api.example.com

# GOOD: write scripts that reference env vars
cat > run.sh << 'EOF'
#!/bin/bash
eval "$(lusterpass env --profile dev)"
python app.py  # app reads $DB_PASSWORD from environment
EOF
```

```bash
# BAD: never do any of these
echo $DB_PASSWORD
printenv API_KEY
lusterpass env --profile dev  # without eval — prints values to screen
cat ~/.lusterpass/cache/*
```

### For AI Agents and Bots

When an AI agent or bot needs to use secrets (e.g., calling an API, connecting to a database):

1. Run `eval "$(lusterpass env)"` to load secrets into the shell
2. Execute commands that reference `$VAR_NAME` — the shell resolves them without the agent ever seeing the values
3. Secrets are scoped to what is listed in `.lusterpass.yaml` — nothing beyond that boundary
4. Never ask the user to paste or type secret values into the conversation

---

## Setup & Adoption Guide

### 1. Install lusterpass

```bash
curl -sSfL https://raw.githubusercontent.com/lustertools/lusterpass/main/install.sh | bash
```

Options:
```bash
# Specific version
VERSION=v0.1.0 curl -sSfL https://raw.githubusercontent.com/lustertools/lusterpass/main/install.sh | bash

# Custom install directory
INSTALL_DIR=~/.local/bin curl -sSfL https://raw.githubusercontent.com/lustertools/lusterpass/main/install.sh | bash
```

### 2. Login

```bash
lusterpass login
```

This prompts for:
- Bitwarden access token (stored encrypted in `~/.lusterpass/config`)
- Organization ID (stored in `~/.lusterpass/org`)

After login, all commands use the cached org ID automatically. Use `--org` to override for a specific call.

### 3. Migrate Existing Secrets

Scan an existing `.envrc` or shell rc file to auto-detect secrets:

```bash
lusterpass migrate .envrc
```

The `--project` flag defaults to the current directory name:
```bash
# Only needed if you want a different project name
lusterpass migrate .envrc --project webapp
```

This generates two files:
- **`.lusterpass.yaml`** — config with vars and secret references (safe to commit)
- **`onboard-secrets.sh`** — script to upload secrets to Bitwarden (contains plain values — delete after use)

**Secret detection** uses key name patterns (PASSWORD, TOKEN, API_KEY, etc.), value patterns (sk-, ghp_, AKIA, eyJ, etc.), and entropy analysis.

### 4. Enrol Secrets to Bitwarden

Review and run the generated onboarding script:

```bash
chmod +x onboard-secrets.sh
./onboard-secrets.sh
```

Or enrol secrets one at a time interactively:
```bash
lusterpass enrol
```

After enrolment, delete the onboarding script immediately:
```bash
rm onboard-secrets.sh
```

### 5. Pull and Use Secrets

```bash
# Fetch secrets from Bitwarden and cache locally (encrypted)
lusterpass pull --profile dev

# Load into current shell
eval "$(lusterpass env --profile dev)"
```

### 6. Direnv Integration

Replace your old plain-text `.envrc` with:

```bash
# .envrc — secrets loaded from Bitwarden via lusterpass
eval "$(lusterpass env --profile dev)"
```

Now secrets auto-load when you `cd` into the project directory (with direnv installed).

### 7. Configuration Reference

`.lusterpass.yaml` format:

```yaml
project: myapp

common:
  vars:
    APP_NAME: myapp
    LOG_FORMAT: json
  secrets:
    # ENV_VAR_NAME: bitwarden-reference-name
    SHARED_SIGNING_KEY: shared-signing-key--myapp

profiles:
  dev:
    vars:
      LOG_LEVEL: debug
    secrets:
      DB_PASSWORD: db-pass--myapp--dev
      OPENAI_API_KEY: openai-key--myapp--dev
  prod:
    vars:
      LOG_LEVEL: warn
    secrets:
      DB_PASSWORD: db-pass--myapp--prod
      OPENAI_API_KEY: openai-key--myapp--prod
```

- **`common`** — shared across all environments
- **`profiles`** — per-environment overrides (profile values override common)
- **`vars`** — plain environment variables (committed to git)
- **`secrets`** — env var names mapped to Bitwarden reference names (the YAML value is a reference, not the secret itself)

Reference naming convention: `<purpose>--<project>[--<env>]`

### Command Reference

| Command | Purpose |
|---------|---------|
| `lusterpass login` | Set up access token + org ID |
| `lusterpass migrate <file>` | Auto-detect secrets, generate config + onboarding script |
| `lusterpass enrol` | Add a secret to Bitwarden (interactive or `--ref`/`--value` flags) |
| `lusterpass pull [--profile name]` | Fetch secrets from Bitwarden, cache locally encrypted |
| `lusterpass env [--profile name]` | Output export lines for shell eval |
| `lusterpass list` | List secret names in Bitwarden (never shows values) |
| `lusterpass --help` | Show all commands and flags |
