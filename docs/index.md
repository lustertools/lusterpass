---
title: lusterpass
description: Agent-safe secret manager for AI coding agents.
---

# lusterpass

**Secrets that AI coding agents can use, but never see.**

Lusterpass is a CLI that loads secrets from [Bitwarden Secrets Manager](https://bitwarden.com/products/secrets-manager/) into a child process's environment via shell `eval`, so AI coding agents can run real commands against real systems without ever exposing the values to the LLM driving the shell.

[View on GitHub](https://github.com/lustertools/lusterpass) · [Latest release](https://github.com/lustertools/lusterpass/releases/latest) · [Report an issue](https://github.com/lustertools/lusterpass/issues)

---

## Install

```bash
curl -sSfL https://raw.githubusercontent.com/lustertools/lusterpass/main/install.sh | bash
```

For specific versions and custom install directories, see the [README](https://github.com/lustertools/lusterpass#quickstart).

---

## Documentation

- **[Bitwarden setup guide](bitwarden-setup.html)** — set up your Bitwarden Secrets Manager account, organization, and machine access token. Start here if you don't have a Bitwarden Secrets Manager account yet.
- **[Migration guide](migration-guide.html)** — migrate an existing `.envrc` (or any shell rc file) to lusterpass with the built-in `lusterpass migrate` command.

---

## Quickstart

After installing and setting up Bitwarden, drop a `.lusterpass.yaml` in your project root:

```yaml
project: myapp

profiles:
  dev:
    vars:
      LOG_LEVEL: debug
    secrets:
      DATABASE_URL: db-url--myapp--dev
      OPENAI_API_KEY: openai-key--myapp--dev
```

Then:

```bash
lusterpass login                       # one-time: store token + org ID
lusterpass pull --profile dev          # fetch + encrypt locally
eval "$(lusterpass env --profile dev)" # load into current shell
```

Your AI coding agent's transcript stays clean. The secret values reach the subprocess environment, where they belong, without ever touching the LLM's context window.
