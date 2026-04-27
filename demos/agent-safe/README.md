# Agent-safe demo

Showcase for lusterpass: an AI coding agent runs the same DB-migration task twice — once reading `.env` directly (secrets leak into the transcript), once via `eval "$(lusterpass env --profile dev)"` (transcript stays clean).

## Files

- `fake_claude.sh` — canned AI-agent REPL (so the demo doesn't depend on a real LLM session)
- `fake_lusterpass.sh` — mock `lusterpass` binary (so the demo doesn't need a Bitwarden account)
- `demo.tape` — [vhs](https://github.com/charmbracelet/vhs) script that renders the GIF
- `record.sh` — one-shot automation: installs vhs/ttyd if needed, renders `agent-safe-demo.gif`
- `PROMPT.md` — the prompt to drive a real coding-agent session in the same scenario

## Generate the GIF

```bash
./record.sh
```

Produces `agent-safe-demo.gif` (~22s, single loop) in this directory. Embedding the rendered GIF in the README is the intended use.

## What viewers see

1. **Act 1 (the problem)** — agent reads `.env`, password value `p4ssw0rd!2026` and OpenAI key prefix appear in the agent's transcript. Annotated as a leak.
2. **Act 2 (with lusterpass)** — agent runs `lusterpass exec -- ./run-migration.sh`, the migration succeeds, no secret value enters the agent's transcript.
3. **Closer** — `grep` over the saved transcript returns zero matches against the leaked tokens.

### Note on the chosen path

The demo uses `lusterpass exec` for visual clarity (single command, no `eval` to explain). The `eval "$(lusterpass env)"` path is equally supported and equally agent-safe in a captured-pipe context — neither is deprecated. The project's [security model](../../docs/security-model.md) compares the two. The demo's purpose is to contrast "agent reads `.env`" (broken) against "agent uses lusterpass" (safe), not to argue that `exec` is the only valid lusterpass invocation.
