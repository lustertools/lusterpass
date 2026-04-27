#!/usr/bin/env bash
# Fake AI coding agent REPL — used only for rendering the demo GIF.
# Reads two prompts (the bad approach and the lusterpass approach), prints
# canned "agent" output for each, and exits.

set -u

GREEN=$'\033[32m'
DIM=$'\033[2m'
RED=$'\033[31m'
RESET=$'\033[0m'

prompt() { printf "\n${GREEN}>${RESET} "; }

prompt
while IFS= read -r line; do
  case "$line" in
    *"read .env"*|*"Read .env"*|*"read the .env"*|*"Read the .env"*)
      sleep 0.4
      echo
      echo "● Reading .env"
      sleep 0.3
      echo "  DB_USER=app"
      sleep 0.15
      echo "  DB_PASSWORD=p4ssw0rd!2026   ${RED}← leaked into my context${RESET}"
      sleep 0.15
      echo "  OPENAI_API_KEY=sk-proj-${RED}REAL_VALUE_HERE${RESET}"
      sleep 0.6
      echo "● Running ./run-migration.sh"
      sleep 0.3
      echo "  ${DIM}connecting as app:p4ssw0rd!2026${RESET}   ${RED}← leaked again${RESET}"
      sleep 0.3
      echo "  ✓ 3 migrations applied"
      ;;
    *lusterpass*|*Lusterpass*|*"the safe way"*)
      sleep 0.4
      echo
      echo "● lusterpass exec -- ./run-migration.sh"
      sleep 0.5
      echo "  [lusterpass] resolved 4 secrets, replaced self with ./run-migration.sh"
      sleep 0.4
      echo "  connecting as \$DB_USER ${DIM}(password from env, never read by me)${RESET}"
      sleep 0.4
      echo "  ✓ 3 migrations applied"
      sleep 0.5
      echo "● Done. Migrations applied. No secret values entered my transcript,"
      echo "  no values leaked into your shell, no eval needed."
      ;;
    /exit|exit|quit)
      break
      ;;
    *) ;;
  esac
  prompt
done
echo
