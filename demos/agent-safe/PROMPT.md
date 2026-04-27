# Driving prompt for the agent-safe demo

Paste the text below into a fresh AI coding agent session to reproduce the demo's storyline manually (without `fake_claude.sh`).

The setup script that the agent expects:
- `.env` containing `DB_USER`, `DB_PASSWORD`, `OPENAI_API_KEY`
- `.lusterpass.yaml` with the same names mapped to Bitwarden references
- `run-migration.sh` that prints "applied" if connection works

---

> I have pending DB migrations. First show me the **broken** way (read `.env` and run the migration), then show me the **safe** way using lusterpass.
>
> For the broken way: read `.env`, print the values you see, and run `./run-migration.sh` with the inherited environment.
>
> For the safe way: run `eval "$(lusterpass env --profile dev)"` and then `./run-migration.sh`. After the run, confirm in one line that no secret value appeared in your output.
>
> Finally, `grep` the transcript of just your second response for any of the literal values from `.env` and confirm it returns nothing.
