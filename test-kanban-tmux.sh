#!/bin/bash
# tmux test for git-kanban: checks arrow key navigation

# Setup test environment
test_dir="$(dirname "$0")"
cd "$test_dir"
cp test-fixtures/README.md README.md

# Start tmux session and run git-kanban
SESSION="kanban-test-$$"
tmux new-session -d -s "$SESSION" "./git-kanban"

# Wait for TUI to load
sleep 0.2

# Send Down arrow key
# tmux send-keys uses C-m for Enter, and Escape sequences for arrows
tmux send-keys -t "$SESSION" C-down

# Wait for redraw
sleep 0.2

# Capture pane output
tmux capture-pane -pt "$SESSION" > tmux-kanban-output.txt

# Kill session
tmux kill-session -t "$SESSION"

# Print output for inspection
cat tmux-kanban-output.txt

# Check if highlight moved to '＋ Create lane'
grep -q "＋ Create lane" tmux-kanban-output.txt && echo "Arrow key highlight test passed" || echo "Arrow key highlight test failed"
