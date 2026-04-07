Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1000
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 24
Set PlaybackSpeed 1.25
Set TypingSpeed 60ms

# ── Hidden setup: build binary silently ──────────────────────────────
Hide
Type "go build -o kubectl-why . 2>/dev/null && clear"
Enter
Sleep 6s
Show

# ── Part 1: Show the problem (wall-of-text kubectl commands) ──────────
Type "# The old way: multiple commands, walls of text..."
Sleep 1.5s
Enter
Sleep 500ms

Type "kubectl get pods"
Sleep 300ms
Enter
Sleep 2.5s

Type "kubectl describe pod oom-test | tail -30"
Sleep 300ms
Enter
Sleep 2.5s

Type "kubectl logs oom-test --previous | tail -15"
Sleep 300ms
Enter
Sleep 3s
Ctrl+C

# ── Part 2: The kubectl-why solution ─────────────────────────────────
Type "# The kubectl-why way: one command, instant answer."
Sleep 1.5s
Enter
Sleep 500ms

Type "./kubectl-why pod oom-test"
Sleep 400ms
Enter
Sleep 4s

# ── Part 3: Deployment support ───────────────────────────────────────
Type "./kubectl-why deployment bad-deploy"
Sleep 400ms
Enter
Sleep 4s

# ── Part 4: CrashLoopBackOff ─────────────────────────────────────────
Type "./kubectl-why pod crash-test"
Sleep 400ms
Enter
Sleep 4s

# Final pause so viewers can read
Sleep 2s
