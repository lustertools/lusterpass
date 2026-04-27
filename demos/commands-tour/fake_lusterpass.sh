#!/usr/bin/env bash
# Mock lusterpass binary for the commands-tour demo.
# Stands in for the real CLI so the demo doesn't need a Bitwarden account.
# Output mirrors the real binary's command shapes (login, list, pull, env, enrol).

set -u

DIM=$'\033[2m'
GREEN=$'\033[32m'
BOLD=$'\033[1m'
CYAN=$'\033[36m'
RESET=$'\033[0m'

STATE_FILE="/tmp/lusterpass-demo-enrolled"

case "${1:-}" in
  login)
    printf "Enter Bitwarden access token: "
    read -rs _TOKEN
    echo "${DIM}[hidden]${RESET}"
    printf "Enter Bitwarden organization ID: "
    read -r _ORG
    sleep 0.3
    echo "${GREEN}✓${RESET} Saved access token and org ID for account ${BOLD}default${RESET}"
    ;;

  list)
    printf "${BOLD}%-12s  %s${RESET}\n" "PROJECT" "SECRET REFERENCE"
    cat <<EOF
myapp         db-pass--myapp--dev
myapp         openai-key--myapp--dev
myapp         stripe-key--myapp--dev
myapp         db-pass--myapp--prod
shared        github-token--ci
EOF
    if [[ -f "$STATE_FILE" ]]; then
      while IFS= read -r line; do
        printf "%s  ${CYAN}← new${RESET}\n" "$line"
      done < "$STATE_FILE"
    fi
    ;;

  pull)
    echo "Fetching 4 secrets from Bitwarden..."
    sleep 0.4
    echo "${GREEN}✓${RESET} Cached 4 secrets to ~/.lusterpass/cache/myapp-dev"
    ;;

  env)
    cat <<'EOF'
export DB_USER='app'
export DB_PASSWORD='p4ssw0rd!2026'
export OPENAI_API_KEY='sk-proj-EXAMPLE-XYZ'
export APP_NAME='myapp'
EOF
    ;;

  enrol)
    printf "Reference name: "
    read -r REFNAME
    printf "Project: "
    read -r PROJ
    printf "Secret value: "
    read -rs _VAL
    echo "${DIM}[hidden]${RESET}"
    printf "Note (optional): "
    read -r _NOTE
    sleep 0.3
    echo "${GREEN}✓${RESET} Created reference ${BOLD}${REFNAME}${RESET} in project ${BOLD}${PROJ}${RESET}"
    printf "%-12s  %s\n" "$PROJ" "$REFNAME" >> "$STATE_FILE"
    ;;

  --version)
    echo "lusterpass v0.1.0"
    ;;

  *)
    echo "lusterpass: demo mock"
    ;;
esac
