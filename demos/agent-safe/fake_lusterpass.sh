#!/usr/bin/env bash
# Mock lusterpass binary — demo GIF only.
# Stands in for the real CLI so the demo doesn't need a Bitwarden account.
# Output deliberately matches the real binary's `env --profile X` shape.

case "${1:-}" in
  env)
    cat <<'EOF'
export DB_USER='app'
export DB_PASSWORD='p4ssw0rd!2026'
export OPENAI_API_KEY='sk-proj-EXAMPLE-XYZ'
export STRIPE_SECRET_KEY='sk_test_EXAMPLE'
EOF
    ;;
  --version)
    echo "lusterpass v0.1.0"
    ;;
  *)
    echo "lusterpass: demo mock"
    ;;
esac
