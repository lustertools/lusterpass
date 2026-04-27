# vhs tape — lusterpass agent-safe demo.
# Render with: ./record.sh   (sets DEMO_DIR and runs vhs)
#
# Two-act structure:
#   Act 1 — "before": agent reads .env, secret values leak into transcript.
#   Act 2 — "after":  agent runs eval $(lusterpass env), transcript stays clean.
#   Closer: grep proof on the saved transcript.

Output agent-safe-demo.gif

Set Shell "zsh"
Set FontSize 20
Set Width 1200
Set Height 720
Set Theme "Catppuccin Mocha"
Set TypingSpeed 30ms
Set PlaybackSpeed 1.0

# ---------- Setup (hidden) ----------
Hide
Type "cd $(mktemp -d)"
Enter
Type "cp $DEMO_DIR/fake_claude.sh ./claude && chmod +x ./claude"
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
Type "# lusterpass — secrets your AI agent uses, but never sees"
Sleep 800ms
Enter
Sleep 1500ms

# ---------- Act 1: the problem ----------
Type "# Act 1 — the broken default: agent reads .env"
Enter
Sleep 800ms

Type "./claude | tee transcript.log"
Sleep 300ms
Enter
Sleep 700ms

Type "I need to apply pending DB migrations. Read the .env and run them."
Sleep 400ms
Enter
Sleep 5500ms

Type "exit"
Enter
Sleep 800ms

# ---------- Act 2: with lusterpass ----------
Type "clear"
Enter
Type "# Act 2 — same task with lusterpass: secrets enter the env, not the transcript"
Enter
Sleep 1000ms

Type "./claude | tee transcript.log"
Sleep 300ms
Enter
Sleep 700ms

Type "Now do it the safe way using lusterpass."
Sleep 400ms
Enter
Sleep 6500ms

Type "exit"
Enter
Sleep 800ms

# ---------- Closer: grep proof ----------
Type "# Did any secret value leak into the agent's transcript?"
Enter
Sleep 600ms

Type "grep -E 'p4ssw0rd|REAL_VALUE|sk-proj-EX' transcript.log || echo '  (zero matches — the LLM never saw a value)'"
Sleep 300ms
Enter
Sleep 4000ms

# Hold final frame
Sleep 2500ms
