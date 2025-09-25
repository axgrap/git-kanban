#!/bin/bash
# Test: Ticket creation
TESTDIR="$(dirname "$0")/log"
mkdir -p "$TESTDIR"
cp ../../test-fixtures/README.md "$TESTDIR/README.md"
cd "$TESTDIR"
SESSION="kanban-create-$$"
tmux new-session -d -s "$SESSION" "../../git-kanban"
sleep 0.5
tmux send-keys -t "$SESSION" -l $'\e[B' \; send-keys -l $'\e[B' \; send-keys C-m
sleep 0.5
tmux send-keys -t "$SESSION" -l $'\e[B' \; send-keys C-m
sleep 0.5
tmux send-keys -t "$SESSION" "Test Ticket" C-m
sleep 0.5
tmux capture-pane -pe -t "$SESSION" > output-ticket-create.txt
tmux kill-session -t "$SESSION"
grep -q "Test Ticket" README.md && echo "test-ticket-create: PASS" || echo "test-ticket-create: FAIL"