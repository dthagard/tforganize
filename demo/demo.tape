# tforganize demo — README gif
# Run: vhs demo/demo.tape

Output demo/demo.gif
Output demo/demo.mp4

Set Shell "zsh"
Set FontSize 14
Set FontFamily "JetBrains Mono"
Set Width 900
Set Height 550
Set Theme "Catppuccin Mocha"
Set Padding 16

Hide
Type "cd /Users/dthagard/.openclaw/workspace-development/tforganize && cp demo/main.tf.orig demo/main.tf && clear"
Enter
Sleep 500ms
Show

# ---------- BEFORE ----------
Type "# Before: unsorted Terraform"
Sleep 600ms
Enter
Sleep 300ms

Type "cat demo/main.tf"
Sleep 500ms
Enter
Sleep 2.5s

# ---------- RUN ----------
Enter
Type "# Sort with tforganize"
Sleep 600ms
Enter
Sleep 300ms

Type "tforganize sort --inline demo/"
Sleep 500ms
Enter
Sleep 1.5s

# ---------- AFTER ----------
Enter
Type "# After: organized ✓"
Sleep 600ms
Enter
Sleep 300ms

Type "cat demo/main.tf"
Sleep 500ms
Enter
Sleep 3s
