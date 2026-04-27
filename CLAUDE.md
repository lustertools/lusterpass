# CLAUDE.md — context for future sessions

This file is for Claude (or another AI coding agent) working in this repo. Read it before making changes; it captures things that aren't obvious from the code or git log alone.

## What lusterpass is

A Go CLI that loads secrets from Bitwarden Secrets Manager into a child process's environment, positioned as agent-safe (secrets never enter the agent's transcript) AND as a `.env`-replacement for CI/dev workstations. Two audiences, same mechanic. Backed by Bitwarden's free-tier Secrets Manager.

Public repo: `github.com/lustertools/lusterpass`. Org: `lustertools`. Brand: `lustertools`. Pages site: <https://lustertools.github.io/lusterpass/> (served from `/docs` on `main`).

## Repository conventions you must follow

- **Never add `Co-Authored-By: Claude` to commit trailers.** The maintainer does not want Claude attribution in the public git log. This rule is recorded in the project's persistent memory; respect it without exception. Product references to Claude Code in docs/skill files are fine; commit-trailer attribution is not.
- **Don't tag releases without explicit instruction.** The maintainer tags manually after reviewing batched changes. Sit on `main` until told to cut a version.
- **Module path is `github.com/lustertools/lusterpass`.** All imports must use it.
- **The `lusterpass exec` and `eval "$(lusterpass env)"` paths are both first-class.** Don't write copy that frames `eval` as deprecated. Skill content (which targets AI agents) can prefer `exec`; user-facing docs must present both as supported.

## Build, test, release

```bash
go build ./...                                    # compile-check
go test ./cmd/ ./internal/... -timeout 30s        # unit tests; CI runs these
go build -trimpath -ldflags "-X main.version=$(git describe --tags --always)" -o build/lusterpass .  # local binary
./local-release.sh                                # build + sudo install to /usr/local/bin
./build/lusterpass test                           # hidden e2e command — needs a real Bitwarden account with a 'testing' project
```

`-trimpath` matters for releases — without it, the absolute source path is embedded in the binary, which has caused brand-leakage issues in the past. The CI release workflow and `local-release.sh` both use `-trimpath`. Don't drop the flag.

### Release flow

Tag `vX.Y.Z` and push: `git tag vX.Y.Z && git push origin vX.Y.Z`. The `.github/workflows/release.yml` builds 4 platform binaries (linux-amd64, linux-arm64, darwin-arm64, windows-amd64) with CGO enabled on native runners, generates `checksums.txt`, and creates a GitHub Release with all five assets attached. Source archives auto-attached by GitHub are intentionally left in place — don't restore the previous "delete tag-ref to strip them" hack, it demoted the release to draft.

Do not re-tag an existing version. The release workflow's `gh release create` step is not idempotent; force-pushing a tag triggers a workflow run that fails at `gh release create` because the release already exists. The build still succeeds; binaries are unaffected; the cosmetic cost is one red workflow run.

## Repository layout

```
lusterpass/
├── main.go                         # entry point, sets version
├── cmd/                            # cobra commands
│   ├── root.go account.go login.go list.go enrol.go
│   ├── pull.go env.go              # env has TTY guard via stdoutIsTerminal var
│   ├── exec.go                     # OS-agnostic exec command shell
│   ├── exec_unix.go                # //go:build unix — syscall.Exec
│   ├── exec_windows.go             # //go:build windows — exec.Cmd.Run
│   ├── migrate.go                  # .envrc → .lusterpass.yaml + onboard-secrets.sh
│   └── test_cmd.go                 # Hidden:true — maintainer self-check, NOT for end users
├── internal/
│   ├── auth/                       # access-token storage + lookup
│   ├── bitwarden/                  # SDK client + a mock for tests
│   ├── cache/                      # encrypted local cache (AES-GCM, key from access token)
│   └── config/                     # .lusterpass.yaml schema + ResolveProfile + CacheKey
├── skills/lusterpass/SKILL.md      # Claude Code skill enforcing eval/exec usage
├── demos/agent-safe/               # vhs demo: leak vs. exec, README has reproduction steps
├── demos/commands-tour/            # vhs demo: login/list/pull/env/enrol workflow
├── docs/
│   ├── index.md                    # Pages landing
│   ├── bitwarden-setup.md          # user-facing setup guide
│   ├── migration-guide.md          # .envrc → lusterpass walkthrough
│   ├── release-and-install.md      # excluded from Pages via _config.yml
│   ├── security-model.md           # threat model, three execution paths, on-disk layout
│   └── _config.yml                 # Jekyll config; excludes release-and-install.md
├── branding/lustertools-brand.png  # used by README + Pages "About lustertools" section
├── testdata/mockapp/               # ONLY fixture — used by `lusterpass test`
├── install.sh                      # one-liner installer, points at lustertools/lusterpass releases
├── local-release.sh                # build+sudo install for the maintainer
├── .github/workflows/
│   ├── ci.yml                      # build + unit tests on push/PR
│   └── release.yml                 # tag-triggered platform builds + release
└── .gitignore                      # see notes below
```

### Gitignore notes (non-obvious)

- `lusterpass` (binary) is gitignored at repo root via `/lusterpass` — anchored slash so `skills/lusterpass/` is NOT excluded.
- `.lusterpass.yaml` and `.envrc` in the root are gitignored, but `testdata/**/.lusterpass.yaml` and `testdata/**/.envrc` are explicitly re-included via `!` patterns. Don't break this — the test fixtures need to be tracked.
- The user's GLOBAL gitignore at `~/.config/gitignore_global` ignores `.envrc*` and `*local*`. The project `.gitignore` has explicit `!.envrc.example` and `!local-release.sh` overrides. Don't remove those overrides.
- `docs/superpowers/`, `docs/plans/`, `findings.md`, `progress.md`, `task_plan.md` are gitignored — they're internal planning artifacts that contain personal absolute paths. Never commit them.

## Configuration model — important behavior

`.lusterpass.yaml` has `common:` (vars + secret refs shared everywhere) and optional `profiles:` (per-environment overlays).

- **`--profile` is OPTIONAL** on `pull`, `env`, and `exec`. Omitted → resolves common only. Don't reintroduce a "profile required" gate.
- **Profile names cannot be `common`** — `Load()` rejects this because the cache file slot for "no profile" is keyed `common`.
- **Unknown profile name errors out** — `ResolveProfile` returns an error listing available profiles. It does NOT silently fall back to common; that fallback was a bug, the explicit error is now load-bearing for catching typos.

The cache file is at `~/.lusterpass/accounts/<account>/cache/<project>/<profile-or-common>.enc`, encrypted with AES-GCM keyed off the access token.

## Demos and how to re-render

Each demo has:
- `demo.tape` (vhs script)
- `fake_lusterpass.sh` (mock binary so the demo doesn't need a Bitwarden account)
- Sometimes `fake_claude.sh` (mock REPL)
- `record.sh` — auto-installs vhs+ttyd via Homebrew, runs vhs

To re-render: `cd demos/<name> && ./record.sh`. Output GIF lands in the same directory and is referenced by README via relative path and by Pages via `raw.githubusercontent.com` URL. Don't duplicate the GIF into `docs/`.

Both demos use `Set FontSize 20` (bumped from 14 for legibility on the README header). Catppuccin Mocha theme. 1200x720.

## Security positioning — what to be careful about

The whole project leans on a security claim. Things that matter:

- The `lusterpass env` command has a TTY guard — refuses to print to an interactive terminal. The guard variable `stdoutIsTerminal` in `cmd/env.go` is package-level so tests can override it. Don't move it; the test in `cmd/env_test.go` depends on the swap.
- `lusterpass exec` uses `syscall.Exec` on Unix (process replacement, zero overhead) and `exec.Cmd.Run` on Windows (fork+wait + signal forwarding). Don't unify these without a clear win — `syscall.Exec` is the right primitive on Unix.
- `docs/security-model.md` is the authoritative threat-model document. If you make security-relevant changes (encryption, cache layout, env merge order, exec mechanic), update this doc. It's linked from README, Pages, and the skill.
- Don't add new commands that print secret values to stdout without an `eval`-style consumer pattern. If you must (e.g., debugging tools), add a TTY guard.

## What lives in the demo GIFs

- `demos/agent-safe/agent-safe-demo.gif` — embedded at top of README and Pages. Shows leak (Act 1) → safe via `lusterpass exec` (Act 2) → grep proof.
- `demos/commands-tour/commands-tour-demo.gif` — embedded under the README's Commands section. Shows login/list/pull/env/enrol with stateful mock binary so the post-enrol list shows the new entry.

If you change command UX, you may need to re-record either or both. Both demos use 20pt font; preserve that.

## Known sharp edges

- **`gh release create` is not idempotent.** Force-pushing an existing tag will fail the release step. If history needs rewriting, delete the release+tag in the right order, then re-tag.
- **GitHub push-protection** scans for known secret patterns (Stripe test keys, AWS keys, etc.). If the diff contains an example value that matches a real-world prefix, the push is blocked. Use obviously fake placeholders like `sk_test_EXAMPLE`, `AKIA_EXAMPLE_KEY_ID`. Don't use the official AWS-docs example string (the one starting with `AKIAIOSFODNN`) — push-protection sometimes flags it.
- **Cross-compile fails locally.** The Bitwarden SDK requires CGO + platform-native builds. `GOOS=windows go build` doesn't work. CI handles this via per-platform runners. Don't try to build releases locally except for your native platform.
- **The hidden `lusterpass test` command writes to your real Bitwarden vault.** It seeds two test secrets in a project named `testing`, fetches them, then deletes them in defer. If you Ctrl-C it mid-run, the seeds may be orphaned in the vault.
- **Reflog retention.** Force-pushed-over commits remain accessible via direct hash URL on GitHub for ~30-90 days. If true secret-purge is needed, GitHub Support has to do it.
- **Privacy:** the maintainer values keeping their public OSS identity scoped to `lustertools`. Don't write copy (in commit messages, README, CLAUDE.md, or any tracked file) that names previous repo locations, prior usernames, or earlier branding. If you find such content, scrub it in the same commit you make the substantive change.

## Recent state

- `lusterpass exec -- <cmd>` exists and is the recommended default for one-shot command execution
- `lusterpass env` has a TTY guard
- `--profile` is optional everywhere it's accepted
- `lusterpass test` is `Hidden: true`
- Pages site at <https://lustertools.github.io/lusterpass/> with a security-model page
- Brand image at `branding/lustertools-brand.png`, embedded in README + Pages

## Workflow tips for future you

- Use absolute paths in Bash invocations after a `cd` — the harness preserves cwd between commands, but it's easy to forget which subdir you're in.
- Verify CI on `main` after every push by checking `gh run list --limit 3`. The release workflow needs a tag push to trigger.
- The user-global `~/.config/gitignore_global` will silently exclude `.envrc.example` and `local-release.sh` from staging unless the project `.gitignore` has explicit `!` overrides. They're already there; don't remove them.
- When updating README and Pages docs, make the same change in both files — they intentionally mirror each other for the headline content.
- Before pushing a CLAUDE.md change, scan it for any prior-username/prior-brand content. Same for any commit message.
