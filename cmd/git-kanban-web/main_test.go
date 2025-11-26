package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunLanesScript(t *testing.T) {
	// Create a temporary test directory
	tmpDir, err := os.MkdirTemp("", "git-kanban-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set git config
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create README.md
	readme := `# Test Project

---

## Kanban Board

**To Do**

- [ ] Task 1
- [ ] Task 2

**Done**

- [x] Task 3
`
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	// Initial commit
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add README: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Copy integrated git-kanban script to temp dir
	scriptSrc := "../../git-kanban"
	scriptDst := filepath.Join(tmpDir, "git-kanban")
	input, err := os.ReadFile(scriptSrc)
	if err != nil {
		t.Fatalf("Failed to read script: %v", err)
	}
	if err := os.WriteFile(scriptDst, input, 0755); err != nil {
		t.Fatalf("Failed to write script: %v", err)
	}

	// Change to temp dir
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	// Run the script
	ctx := context.Background()
	tickets, err := runLanesScript(ctx, scriptDst, 10*time.Second)
	if err != nil {
		t.Fatalf("runLanesScript failed: %v", err)
	}

	// Verify results
	if len(tickets) != 3 {
		t.Errorf("Expected 3 tickets, got %d", len(tickets))
	}

	// Check first ticket
	if len(tickets) > 0 {
		if tickets[0].LaneIndex != 0 {
			t.Errorf("Expected lane index 0, got %d", tickets[0].LaneIndex)
		}
		if tickets[0].LaneName != "To Do" {
			t.Errorf("Expected lane name 'To Do', got '%s'", tickets[0].LaneName)
		}
		if tickets[0].Text != "Task 1" {
			t.Errorf("Expected text 'Task 1', got '%s'", tickets[0].Text)
		}
	}
}

func TestMergeOwnersIntoBoard(t *testing.T) {
	board := &Board{
		Columns: []Column{
			{
				Name: "To Do",
				Cards: []Card{
					{ID: "1", Title: "Task 1", Checked: false},
					{ID: "2", Title: "Task 2", Checked: false},
				},
			},
			{
				Name: "Done",
				Cards: []Card{
					{ID: "3", Title: "Task 3", Checked: true},
				},
			},
		},
	}

	infos := []TicketInfo{
		{LaneIndex: 0, LaneName: "To Do", TicketIndex: 0, Text: "Task 1", Owner: "Alice"},
		{LaneIndex: 0, LaneName: "To Do", TicketIndex: 1, Text: "Task 2", Owner: "Bob"},
		{LaneIndex: 1, LaneName: "Done", TicketIndex: 0, Text: "Task 3", Owner: "Charlie"},
	}

	mergeOwnersIntoBoard(board, infos)

	// Check owners were assigned
	if board.Columns[0].Cards[0].Assignee != "Alice" {
		t.Errorf("Expected Task 1 owner 'Alice', got '%s'", board.Columns[0].Cards[0].Assignee)
	}
	if board.Columns[0].Cards[1].Assignee != "Bob" {
		t.Errorf("Expected Task 2 owner 'Bob', got '%s'", board.Columns[0].Cards[1].Assignee)
	}
	if board.Columns[1].Cards[0].Assignee != "Charlie" {
		t.Errorf("Expected Task 3 owner 'Charlie', got '%s'", board.Columns[1].Cards[0].Assignee)
	}
}

func TestMergeOwnersTitleFallback(t *testing.T) {
	board := &Board{
		Columns: []Column{
			{
				Name: "To Do",
				Cards: []Card{
					{ID: "1", Title: "Task 1", Checked: false},
					{ID: "2", Title: "Task 2", Checked: false},
				},
			},
		},
	}

	// TicketInfo with wrong index but correct title
	infos := []TicketInfo{
		{LaneIndex: 0, LaneName: "To Do", TicketIndex: 99, Text: "Task 2", Owner: "Alice"},
	}

	mergeOwnersIntoBoard(board, infos)

	// Should fall back to title matching
	if board.Columns[0].Cards[1].Assignee != "Alice" {
		t.Errorf("Expected Task 2 owner 'Alice' via fallback, got '%s'", board.Columns[0].Cards[1].Assignee)
	}
}

func TestParseBoard(t *testing.T) {
	// Create a temporary README
	tmpDir, err := os.MkdirTemp("", "git-kanban-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	readme := `# Test Project

---

## Kanban Board

**To Do**

- [ ] Task 1
- [ ] Task 2

**In Progress**

- [ ] Task 3

**Done**

- [x] Task 4
- [x] Task 5
`
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	board, err := parseBoard(readmePath)
	if err != nil {
		t.Fatalf("parseBoard failed: %v", err)
	}

	// Verify structure
	if len(board.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(board.Columns))
	}

	if board.Columns[0].Name != "To Do" {
		t.Errorf("Expected column 0 name 'To Do', got '%s'", board.Columns[0].Name)
	}

	if len(board.Columns[0].Cards) != 2 {
		t.Errorf("Expected 2 cards in To Do, got %d", len(board.Columns[0].Cards))
	}

	if board.Columns[0].Cards[0].Title != "Task 1" {
		t.Errorf("Expected card title 'Task 1', got '%s'", board.Columns[0].Cards[0].Title)
	}

	if board.Columns[0].Cards[0].Checked {
		t.Error("Expected Task 1 to be unchecked")
	}

	if len(board.Columns[2].Cards) != 2 {
		t.Errorf("Expected 2 cards in Done, got %d", len(board.Columns[2].Cards))
	}

	if !board.Columns[2].Cards[0].Checked {
		t.Error("Expected Task 4 to be checked")
	}
}

func TestWriteBoard(t *testing.T) {
	// Create a temporary README
	tmpDir, err := os.MkdirTemp("", "git-kanban-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	readme := `# Test Project

Some content before.

---

## Kanban Board

**To Do**

- [ ] Task 1

**Done**

## Next Section

Content after.
`
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readme), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	// Create a modified board
	board := &Board{
		Columns: []Column{
			{
				Name: "To Do",
				Cards: []Card{
					{ID: "1", Title: "Task 1", Checked: false},
					{ID: "2", Title: "Task 2", Checked: false},
				},
			},
			{
				Name: "Done",
				Cards: []Card{
					{ID: "3", Title: "Task 3", Checked: true},
				},
			},
		},
	}

	// Write board
	if err := writeBoard(readmePath, board); err != nil {
		t.Fatalf("writeBoard failed: %v", err)
	}

	// Read back and verify
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read back README: %v", err)
	}

	contentStr := string(content)

	// Check that content before is preserved
	if !strings.Contains(contentStr, "# Test Project") {
		t.Error("Expected '# Test Project' to be preserved")
	}
	if !strings.Contains(contentStr, "Some content before.") {
		t.Error("Expected content before to be preserved")
	}

	// Check that content after is preserved
	if !strings.Contains(contentStr, "## Next Section") {
		t.Error("Expected '## Next Section' to be preserved")
	}
	if !strings.Contains(contentStr, "Content after.") {
		t.Error("Expected content after to be preserved")
	}

	// Check that new tasks are present
	if !strings.Contains(contentStr, "Task 2") {
		t.Error("Expected 'Task 2' to be added")
	}
	if !strings.Contains(contentStr, "Task 3") {
		t.Error("Expected 'Task 3' to be added")
	}

	// Check formatting
	if !strings.Contains(contentStr, "**To Do**") {
		t.Error("Expected '**To Do**' lane header")
	}
	if !strings.Contains(contentStr, "- [ ] Task 1") {
		t.Error("Expected unchecked Task 1")
	}
	if !strings.Contains(contentStr, "- [x] Task 3") {
		t.Error("Expected checked Task 3")
	}
}
