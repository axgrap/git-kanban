#!/bin/bash
# Integration test for git-kanban-lanes.sh
# Tests owner inference with git blame and "kanban:" prefix filtering

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_DIR="$SCRIPT_DIR/test-lanes-tmp"

echo "Setting up test environment..."

# Clean up previous test run
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Initialize git repo
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Create initial README with Kanban board
cat > README.md <<'EOF'
# Test Project

Some content before the board.

---

## Kanban Board

**To Do**

- [ ] Task 1
- [ ] Task 2

**In Progress**

- [ ] Task 3

**Done**

- [x] Task 4
EOF

# Initial commit
git add README.md
git commit -m "Initial commit"

# Simulate Alice claiming Task 1 with kanban: prefix
export GIT_AUTHOR_NAME="Alice"
export GIT_COMMITTER_NAME="Alice"
export GIT_AUTHOR_EMAIL="alice@example.com"
export GIT_COMMITTER_EMAIL="alice@example.com"
sed -i 's/- \[ \] Task 1/- [x] Task 1/' README.md
git add README.md
git commit -m "kanban: Alice claims Task 1"

# Simulate Bob claiming Task 2 with kanban: prefix
export GIT_AUTHOR_NAME="Bob"
export GIT_COMMITTER_NAME="Bob"
export GIT_AUTHOR_EMAIL="bob@example.com"
export GIT_COMMITTER_EMAIL="bob@example.com"
sed -i 's/- \[ \] Task 2/- [x] Task 2/' README.md
git add README.md
git commit -m "kanban: Bob claims Task 2"

# Simulate a non-kanban commit (should be ignored)
export GIT_AUTHOR_NAME="Charlie"
export GIT_COMMITTER_NAME="Charlie"
export GIT_AUTHOR_EMAIL="charlie@example.com"
export GIT_COMMITTER_EMAIL="charlie@example.com"
sed -i 's/- \[ \] Task 3/- [x] Task 3/' README.md
git add README.md
git commit -m "Fixed Task 3 (no kanban prefix)"

# Then Alice updates Task 3 description with kanban: prefix (this should become the owner)
export GIT_AUTHOR_NAME="Alice"
export GIT_COMMITTER_NAME="Alice"
export GIT_AUTHOR_EMAIL="alice@example.com"
export GIT_COMMITTER_EMAIL="alice@example.com"
sed -i 's/- \[x\] Task 3/- [x] Task 3 - updated/' README.md
git add README.md
git commit -m "kanban: Alice updates Task 3"

# Run the integrated git-kanban script with --lanes
echo "Running git-kanban --lanes..."
OUTPUT=$("$PROJECT_ROOT/git-kanban" --lanes)

echo "Output:"
echo "$OUTPUT"

# Validate output format and content
echo "Validating output..."

# Check that we have the expected number of lines (4 tasks)
LINE_COUNT=$(echo "$OUTPUT" | wc -l)
if [ "$LINE_COUNT" -ne 4 ]; then
  echo "FAIL: Expected 4 lines, got $LINE_COUNT"
  exit 1
fi

# Check Task 1 has owner Alice
TASK1=$(echo "$OUTPUT" | grep "Task 1")
if ! echo "$TASK1" | grep -q "Alice"; then
  echo "FAIL: Task 1 should have owner Alice"
  echo "Got: $TASK1"
  exit 1
fi

# Check Task 2 has owner Bob
TASK2=$(echo "$OUTPUT" | grep "Task 2")
if ! echo "$TASK2" | grep -q "Bob"; then
  echo "FAIL: Task 2 should have owner Bob"
  echo "Got: $TASK2"
  exit 1
fi

# Check Task 3 has owner Alice (should skip Charlie's non-kanban commit)
TASK3=$(echo "$OUTPUT" | grep "Task 3")
if ! echo "$TASK3" | grep -q "Alice"; then
  echo "FAIL: Task 3 should have owner Alice (Charlie's commit should be ignored)"
  echo "Got: $TASK3"
  exit 1
fi

# Check Task 4 has no owner (only initial commit)
TASK4=$(echo "$OUTPUT" | grep "Task 4")
TASK4_OWNER=$(echo "$TASK4" | awk -F'\t' '{print $5}')
if [ -n "$TASK4_OWNER" ]; then
  echo "FAIL: Task 4 should have no owner (empty string)"
  echo "Got: $TASK4"
  exit 1
fi

# Check TSV format (5 fields separated by tabs)
if ! echo "$OUTPUT" | head -1 | grep -qP '^\d+\t.+\t\d+\t.+\t.*$'; then
  echo "FAIL: Output not in expected TSV format"
  exit 1
fi

# Clean up
cd "$PROJECT_ROOT"
rm -rf "$TEST_DIR"

echo "PASS: All tests passed!"
exit 0
