# tforganize demo — README gif
# Requires: vhs, tmux, tforganize
# Run from project root: vhs demo/demo.tape

Output demo/demo.gif
Output demo/demo.mp4

Set Shell "zsh"

Set TypingSpeed 75ms
Set FontSize 22
Set Width 1600
Set Height 1100
Set Theme "Dracula"

# ---------- Hidden setup ----------

Hide
Set TypingSpeed 1ms

# Reset fixture
Type "cp demo/main.tf.orig demo/main.tf && clear"
Enter
Sleep 500ms

# Launch an isolated tmux (no user config, dedicated socket)
Type "tmux -f /dev/null -L demo kill-server 2>/dev/null; true"
Enter
Sleep 300ms

# Left pane: run cat directly (no interactive shell = no prompt clutter)
Type "tmux -f /dev/null -L demo new-session -d -s demo 'tput cup 1 0 && cat demo/main.tf && sleep 300'"
Enter
Sleep 500ms

# Right pane: bash with clean "$ " prompt
Type "tmux -L demo set-environment -t demo PS1 '$ '"
Enter
Sleep 100ms
Type "tmux -L demo split-window -h -t demo 'bash --norc --noprofile'"
Enter
Sleep 300ms

# Disable status bar
Type "tmux -L demo set -t demo status off"
Enter
Sleep 200ms

# Select the right pane and attach
Type "tmux -L demo select-pane -t demo:0.1 && tmux -L demo attach -t demo"
Enter
Sleep 1s

# ---------- Visible recording ----------

Show
Set TypingSpeed 100ms

Sleep 2s

# Type a comment
Type "# Sort with tforganize"
Sleep 800ms
Enter
Sleep 500ms

# Type and run the sort command
Type "tforganize sort --inline demo/"
Sleep 1.5s
Enter
Sleep 1s

# Display the sorted result
Type "cat demo/main.tf"
Sleep 500ms
Enter
Sleep 10s
