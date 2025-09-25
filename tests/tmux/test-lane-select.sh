#!/bin/bash
# Test: Lane selection and exit
TESTDIR="$(dirname "$0")/log"
mkdir -p "$TESTDIR"
cp ../../test-fixtures/README.md "$TESTDIR/README.md"
cd "$TESTDIR"
SESSION="kanban-lane-$$"
tmux new-session -d -s "$SESSION" "../../git-kanban"
sleep 0.5
echo "[LOG] After session start" > lane-select.txt
tmux capture-pane -pe -t "$SESSION" >> lane-select.txt
# Move selection to first lane (Down twice)
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
echo "[LOG] After first Down" >> lane-select.txt
tmux capture-pane -pe -t "$SESSION" >> lane-select.txt
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
echo "[LOG] After second Down" >> lane-select.txt
tmux capture-pane -pe -t "$SESSION" >> lane-select.txt
# Press Enter to select lane
tmux send-keys -t "$SESSION" C-m
sleep 0.5
echo "[LOG] After Enter" >> lane-select.txt
tmux capture-pane -pe -t "$SESSION" >> lane-select.txt
sleep 2
echo "[LOG] Final capture" >> lane-select.txt
tmux capture-pane -pe -t "$SESSION" > output-lane-select-after.txt
tmux kill-session -t "$SESSION"
grep -q "Use ↑/↓ arrows to select ticket" output-lane-select-after.txt && echo "test-lane-select: PASS" || echo "test-lane-select: FAIL"