# vhs tape — lusterpass commands tour.
# Render with: ./record.sh   (sets DEMO_DIR and runs vhs)
#
# Five-act developer-day-in-the-life:
#   1. login   — store Bitwarden access token + org ID
#   2. list    — see what's in the vault
#   3. pull    — fetch + cache for a profile
#   4. env     — load secrets into shell via eval, prove non-secret vars are set
#   5. enrol   — add a new secret, then re-list to confirm

Output commands-tour-demo.gif

Set Shell "zsh"
Set FontSize 20
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms
Set PlaybackSpeed 1.0

# ---------- Setup (hidden) ----------
Hide
Type "rm -f /tmp/lusterpass-demo-enrolled"
Enter
Type "cd $(mktemp -d)"
Enter
Type "mkdir -p bin && cp $DEMO_DIR/fake_lusterpass.sh bin/lusterpass && chmod +x bin/lusterpass"
Enter
Type "export PATH=$PWD/bin:$PATH"
Enter
Type "setopt INTERACTIVE_COMMENTS"
Enter
Type "clear"
Enter
Show

# ---------- Title ----------
Type "# lusterpass — daily secret workflow"
Sleep 800ms
Enter
Sleep 1500ms

# ---------- Act 1: login ----------
Type "lusterpass login"
Enter
Sleep 1000ms
Type "0YlmS8EXAMPLEACCESSTOKEN"
Sleep 200ms
Enter
Sleep 600ms
Type "0c4f5b40-1234-5678-9012-abcdef012345"
Sleep 200ms
Enter
Sleep 1800ms

# ---------- Act 2: list ----------
Type "lusterpass list"
Enter
Sleep 3500ms

# ---------- Act 3: pull ----------
Type "lusterpass pull --profile dev"
Enter
Sleep 2500ms

# ---------- Act 4: load + verify non-secret vars ----------
Type `eval "$(lusterpass env --profile dev)"`
Enter
Sleep 600ms
Type "printenv | grep -E '^(DB_USER|APP_NAME)='"
Enter
Sleep 2500ms

# ---------- Act 5: enrol a new secret ----------
Type "lusterpass enrol"
Enter
Sleep 800ms
Type "stripe-webhook-secret--myapp--dev"
Sleep 200ms
Enter
Sleep 600ms
Type "myapp"
Sleep 200ms
Enter
Sleep 600ms
Type "whsec_EXAMPLE_VALUE_NEVER_VISIBLE"
Sleep 200ms
Enter
Sleep 600ms
Type "Stripe webhook signing secret"
Sleep 200ms
Enter
Sleep 1800ms

# ---------- Closer: re-list, confirm new entry ----------
Type "lusterpass list"
Enter
Sleep 4000ms

# Hold final frame
Sleep 2500ms
