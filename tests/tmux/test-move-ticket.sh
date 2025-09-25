#!/bin/bash
# Test: Move ticket from one lane to another
TESTDIR="$(dirname "$0")/log"
mkdir -p "$TESTDIR"
cp ../../test-fixtures/README.md "$TESTDIR/README.md"
cd "$TESTDIR"
SESSION="kanban-move-$$"
tmux new-session -d -s "$SESSION" "../../git-kanban"
sleep 0.5
# Move selection to first lane (Down twice)
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
# Press Enter to select lane
# Now in ticket selection for 'To Do'
tmux send-keys -t "$SESSION" C-m
sleep 0.5

# Select first ticket (Down three times)
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
# Press Enter to select ticket
tmux send-keys -t "$SESSION" C-m
sleep 0.5
# Move to 'In Progress' (Down twice)
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
tmux send-keys -t "$SESSION" -l $'\e[B'
sleep 0.2
# Press Enter to move ticket
tmux send-keys -t "$SESSION" C-m
sleep 1
# End first session
tmux kill-session -t "$SESSION"

# Restart script and select 'In Progress' lane
SESSION2="kanban-move2-$$"
tmux new-session -d -s "$SESSION2" "../../git-kanban"
sleep 0.5
tmux send-keys -t "$SESSION2" -l $'\e[B'
sleep 0.2
tmux send-keys -t "$SESSION2" -l $'\e[B'
sleep 0.2
tmux send-keys -t "$SESSION2" -l $'\e[B'
sleep 0.2
# Press Enter to select 'In Progress' lane
tmux send-keys -t "$SESSION2" C-m
sleep 0.5
tmux capture-pane -pe -t "$SESSION2" > output-move-ticket.txt
tmux kill-session -t "$SESSION2"
grep -q "1|Example ticket 1" output-move-ticket.txt && echo "test-move-ticket: PASS" || echo "test-move-ticket: FAIL"
