#!/bin/bash
# Test: Arrow key highlights next selection
TESTDIR="$(dirname "$0")/log"
mkdir -p "$TESTDIR"
cp ../../test-fixtures/README.md "$TESTDIR/README.md"
cd "$TESTDIR"
SESSION="kanban-arrow-$$"
tmux new-session -d -s "$SESSION" "../../git-kanban"
sleep 0.2
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
tmux capture-pane -pe -t "$SESSION" > output-arrow-key.txt
tmux kill-session -t "$SESSION"
grep -q "＋ Create lane" output-arrow-key.txt && echo "test-arrow-key: PASS" || echo "test-arrow-key: FAIL"