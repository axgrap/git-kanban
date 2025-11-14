package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var staticFiles embed.FS

// Configuration
const (
	DefaultScriptPath = "./git-kanban-lanes.sh"
	DefaultTimeout    = 10 * time.Second
	DefaultCacheTTL   = 5 * time.Second
	DefaultPort       = 8080
)

// TicketInfo represents enriched ticket data from the shell script
type TicketInfo struct {
	LaneIndex   int    `json:"lane_index"`
	LaneName    string `json:"lane_name"`
	TicketIndex int    `json:"ticket_index"`
	Text        string `json:"text"`
	Owner       string `json:"owner"`
}

// Card represents a Kanban card/ticket
type Card struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Checked  bool   `json:"checked"`
	Assignee string `json:"assignee,omitempty"` // Owner from git blame
}

// Column represents a Kanban lane/column
type Column struct {
	Name  string `json:"name"`
	Cards []Card `json:"cards"`
}

// Board represents the complete Kanban board
type Board struct {
	Columns []Column `json:"columns"`
}

// Cache for owner information
type ownerCache struct {
	mu         sync.RWMutex
	data       []TicketInfo
	lastUpdate time.Time
	ttl        time.Duration
}

func (c *ownerCache) get() ([]TicketInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Since(c.lastUpdate) < c.ttl {
		return c.data, true
	}
	return nil, false
}

func (c *ownerCache) set(data []TicketInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
	c.lastUpdate = time.Now()
}

var cache = &ownerCache{ttl: DefaultCacheTTL}

// runLanesScript executes the shell script and parses TSV output
func runLanesScript(ctx context.Context, scriptPath string, timeout time.Duration) ([]TicketInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, scriptPath, "--lanes")
	cmd.Dir = "." // Run in current directory (repo root)
	
	// Sanitize environment
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start script: %w", err)
	}

	// Read stdout
	var tickets []TicketInfo
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.SplitN(line, "\t", 5)
		
		if len(fields) < 5 {
			log.Printf("Warning: skipping malformed line (expected 5 fields, got %d): %s", len(fields), line)
			continue
		}

		laneIndex, err := strconv.Atoi(fields[0])
		if err != nil {
			log.Printf("Warning: invalid lane_index: %s", fields[0])
			continue
		}

		ticketIndex, err := strconv.Atoi(fields[2])
		if err != nil {
			log.Printf("Warning: invalid ticket_index: %s", fields[2])
			continue
		}

		tickets = append(tickets, TicketInfo{
			LaneIndex:   laneIndex,
			LaneName:    fields[1],
			TicketIndex: ticketIndex,
			Text:        fields[3],
			Owner:       fields[4],
		})
	}

	// Capture stderr
	stderrBytes, _ := io.ReadAll(stderr)

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("script failed: %w\nstderr: %s", err, string(stderrBytes))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading script output: %w", err)
	}

	return tickets, nil
}

// mergeOwnersIntoBoard attaches owner information to board cards
func mergeOwnersIntoBoard(board *Board, infos []TicketInfo) {
	for _, info := range infos {
		// Try index-based matching first
		if info.LaneIndex >= 0 && info.LaneIndex < len(board.Columns) {
			column := &board.Columns[info.LaneIndex]
			if info.TicketIndex >= 0 && info.TicketIndex < len(column.Cards) {
				board.Columns[info.LaneIndex].Cards[info.TicketIndex].Assignee = info.Owner
				continue
			}
		}

		// Fallback: exact title match in the correct column
		if info.LaneIndex >= 0 && info.LaneIndex < len(board.Columns) {
			column := &board.Columns[info.LaneIndex]
			matchCount := 0
			matchIndex := -1
			for i, card := range column.Cards {
				if card.Title == info.Text {
					matchCount++
					matchIndex = i
				}
			}
			// Only assign if there's exactly one match (not ambiguous)
			if matchCount == 1 {
				column.Cards[matchIndex].Assignee = info.Owner
			}
		}
	}
}

// parseBoard reads README.md and constructs a Board object
func parseBoard(readmePath string) (*Board, error) {
	file, err := os.Open(readmePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open README: %w", err)
	}
	defer file.Close()

	board := &Board{Columns: []Column{}}
	scanner := bufio.NewScanner(file)
	
	// Find Kanban section
	foundSeparator := false
	foundKanban := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			foundSeparator = true
			continue
		}
		if foundSeparator && strings.HasPrefix(line, "##") && strings.Contains(strings.ToLower(line), "kanban") {
			foundKanban = true
			break
		}
	}

	if !foundKanban {
		return nil, fmt.Errorf("Kanban board section not found in README")
	}

	var currentColumn *Column
	
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Stop at next H2 section
		if strings.HasPrefix(line, "##") {
			break
		}

		// Detect lane (bolded text)
		if strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**") {
			laneName := strings.Trim(trimmed, "*")
			laneName = strings.TrimSpace(laneName)
			currentColumn = &Column{Name: laneName, Cards: []Card{}}
			board.Columns = append(board.Columns, *currentColumn)
			continue
		}

		// Detect ticket
		if strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") {
			if currentColumn == nil {
				continue
			}
			
			checked := strings.HasPrefix(trimmed, "- [x]")
			title := strings.TrimSpace(trimmed[5:]) // Skip "- [ ] " or "- [x] "
			
			card := Card{
				ID:      fmt.Sprintf("%s-%d", currentColumn.Name, len(currentColumn.Cards)),
				Title:   title,
				Checked: checked,
			}
			
			// Update the current column in the board
			board.Columns[len(board.Columns)-1].Cards = append(board.Columns[len(board.Columns)-1].Cards, card)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading README: %w", err)
	}

	return board, nil
}

// writeBoard updates the Kanban section in README.md
func writeBoard(readmePath string, board *Board) error {
	// Read entire file
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return fmt.Errorf("failed to read README: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	
	// Find Kanban section bounds
	startLine := -1
	endLine := -1
	foundSeparator := false
	
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			foundSeparator = true
			continue
		}
		if foundSeparator && strings.HasPrefix(line, "##") && strings.Contains(strings.ToLower(line), "kanban") {
			startLine = i
			continue
		}
		if startLine != -1 && strings.HasPrefix(line, "##") {
			endLine = i
			break
		}
	}

	if startLine == -1 {
		return fmt.Errorf("Kanban board section not found")
	}

	if endLine == -1 {
		endLine = len(lines)
	}

	// Build new Kanban section
	var newSection []string
	newSection = append(newSection, lines[startLine]) // Keep the ## Kanban header
	newSection = append(newSection, "")

	for _, column := range board.Columns {
		newSection = append(newSection, fmt.Sprintf("**%s**", column.Name))
		newSection = append(newSection, "")
		
		for _, card := range column.Cards {
			checkbox := "[ ]"
			if card.Checked {
				checkbox = "[x]"
			}
			newSection = append(newSection, fmt.Sprintf("- %s %s", checkbox, card.Title))
		}
		newSection = append(newSection, "")
	}

	// Reconstruct file
	var result []string
	result = append(result, lines[:startLine]...)
	result = append(result, newSection...)
	if endLine < len(lines) {
		result = append(result, lines[endLine:]...)
	}

	// Write back
	output := strings.Join(result, "\n")
	if err := os.WriteFile(readmePath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write README: %w", err)
	}

	return nil
}

// gitCommit creates a git commit with the given message
func gitCommit(message string) error {
	cmd := exec.Command("git", "add", "README.md")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %w\noutput: %s", err, string(output))
	}

	return nil
}

// HTTP Handlers

func handleLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse board from README.md
	board, err := parseBoard("README.md")
	if err != nil {
		log.Printf("Error parsing board: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse board: %v", err), http.StatusInternalServerError)
		return
	}

	// Try to get owner info from cache
	ownerInfos, cached := cache.get()
	if !cached {
		// Run script to get fresh owner info
		ctx := context.Background()
		ownerInfos, err = runLanesScript(ctx, DefaultScriptPath, DefaultTimeout)
		if err != nil {
			log.Printf("Warning: failed to get owner info: %v", err)
			// Continue without owner info rather than failing
			ownerInfos = nil
		} else {
			cache.set(ownerInfos)
		}
	}

	// Merge owner info into board
	if ownerInfos != nil {
		mergeOwnersIntoBoard(board, ownerInfos)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(board)
}

func handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var board Board
	if err := json.NewDecoder(r.Body).Decode(&board); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Write board to README.md
	if err := writeBoard("README.md", &board); err != nil {
		log.Printf("Error writing board: %v", err)
		http.Error(w, fmt.Sprintf("Failed to write board: %v", err), http.StatusInternalServerError)
		return
	}

	// Create git commit
	if err := gitCommit("kanban: updated board via web UI"); err != nil {
		log.Printf("Warning: git commit failed: %v", err)
		// Don't fail the request - board was written successfully
	}

	// Invalidate cache
	cache.set(nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type ClaimRequest struct {
	LaneIndex   int    `json:"lane_index"`
	TicketIndex int    `json:"ticket_index"`
	Owner       string `json:"owner"`
	Action      string `json:"action"` // "claim" or "unclaim"
}

func handleClaim(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Parse current board
	board, err := parseBoard("README.md")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse board: %v", err), http.StatusInternalServerError)
		return
	}

	// Validate indices
	if req.LaneIndex < 0 || req.LaneIndex >= len(board.Columns) {
		http.Error(w, "Invalid lane index", http.StatusBadRequest)
		return
	}
	if req.TicketIndex < 0 || req.TicketIndex >= len(board.Columns[req.LaneIndex].Cards) {
		http.Error(w, "Invalid ticket index", http.StatusBadRequest)
		return
	}

	// Update owner (just for the commit message, actual owner comes from git blame)
	card := &board.Columns[req.LaneIndex].Cards[req.TicketIndex]
	
	// Write board back (no changes to content, just to trigger a commit)
	if err := writeBoard("README.md", board); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write board: %v", err), http.StatusInternalServerError)
		return
	}

	// Create appropriate commit message with kanban: prefix
	var commitMsg string
	if req.Action == "unclaim" {
		commitMsg = fmt.Sprintf("kanban: unclaim '%s' from %s", card.Title, board.Columns[req.LaneIndex].Name)
	} else {
		commitMsg = fmt.Sprintf("kanban: %s claims '%s' in %s", req.Owner, card.Title, board.Columns[req.LaneIndex].Name)
	}

	if err := gitCommit(commitMsg); err != nil {
		log.Printf("Warning: git commit failed: %v", err)
	}

	// Invalidate cache
	cache.set(nil)

	// Return updated board with fresh owner info
	time.Sleep(100 * time.Millisecond) // Give git a moment
	ownerInfos, err := runLanesScript(context.Background(), DefaultScriptPath, DefaultTimeout)
	if err != nil {
		log.Printf("Warning: failed to get updated owner info: %v", err)
	} else {
		mergeOwnersIntoBoard(board, ownerInfos)
		cache.set(ownerInfos)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(board)
}

func main() {
	// Verify we're in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		log.Println("Warning: Not in a git repository root. Some features may not work.")
	}

	// Verify script exists
	scriptPath := DefaultScriptPath
	if absPath, err := filepath.Abs(scriptPath); err == nil {
		scriptPath = absPath
	}
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Printf("Warning: Script not found at %s. Owner inference will not work.", scriptPath)
	}

	// Set up HTTP routes
	http.HandleFunc("/api/load", handleLoad)
	http.HandleFunc("/api/save", handleSave)
	http.HandleFunc("/api/claim", handleClaim)
	
	// Serve static files
	http.Handle("/", http.FileServer(http.FS(staticFiles)))

	port := DefaultPort
	log.Printf("Starting git-kanban web server on http://localhost:%d", port)
	log.Printf("Script path: %s", scriptPath)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
