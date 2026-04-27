# Commands tour demo

Walks a developer through the everyday lusterpass workflow in ~30 seconds: `login`, `list`, `pull`, `env`, `enrol`.

## Files

- `fake_lusterpass.sh` — mock `lusterpass` binary that mirrors the real CLI's command shapes (so the demo doesn't need a Bitwarden account)
- `demo.tape` — [vhs](https://github.com/charmbracelet/vhs) script
- `record.sh` — one-shot automation: installs vhs/ttyd if needed, renders `commands-tour-demo.gif`

## Generate the GIF

```bash
./record.sh
```

Produces `commands-tour-demo.gif` (~30s, single loop) in this directory.

## What viewers see

1. **`lusterpass login`** — prompts for Bitwarden access token (hidden) and organization ID, confirms saved.
2. **`lusterpass list`** — table of secret reference names grouped by project (values never shown).
3. **`lusterpass pull --profile dev`** — fetches and caches the profile's secrets into `~/.lusterpass/cache/`.
4. **`eval "$(lusterpass env --profile dev)"`** — loads everything into the current shell. A trailing `printenv | grep` shows non-secret vars (`DB_USER`, `APP_NAME`) are present, while the secret values stay out of the visible filter.
5. **`lusterpass enrol`** — interactive prompts (reference name, project, hidden value, optional note) add a new secret to the vault.
6. **`lusterpass list` again** — the new entry appears, marked `← new`.
