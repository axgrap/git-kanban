# Git Kanban Web Server - Integration Documentation

This document describes the Go web server implementation that provides a browser-based GUI for managing the git-kanban board.

## Overview

The `git-kanban-web` server provides a novice-friendly web interface for managing Kanban boards stored in `README.md`. It integrates with the `git-kanban-lanes.sh` shell script to infer ticket ownership from git history.

## Architecture

### Components

1. **Shell Script (`git-kanban-lanes.sh`)**: Parses README.md and uses git blame to determine ticket owners
2. **Go Web Server**: Provides HTTP API and serves the browser UI
3. **Browser UI**: Single-page application with drag-and-drop, claim/unclaim features

### Data Flow

```
README.md → parseBoard() → Board object
                ↓
Board object → git-kanban-lanes.sh --lanes → TSV output
                ↓
TSV → runLanesScript() → []TicketInfo
                ↓
mergeOwnersIntoBoard() → Enriched Board with owners
                ↓
JSON → Browser UI
```

## Building and Running

### Prerequisites

- Go 1.21 or later
- Git (for owner inference)
- Bash (for the shell script)

### Build

```bash
cd cmd/git-kanban-web
go build -o git-kanban-web
```

### Run

From the repository root:

```bash
./cmd/git-kanban-web/git-kanban-web
```

The server will start on `http://localhost:8080` by default.

## Configuration

Configuration is currently done via constants in `main.go`:

| Setting | Default | Description |
|---------|---------|-------------|
| `DefaultScriptPath` | `./git-kanban-lanes.sh` | Path to the lanes script |
| `DefaultTimeout` | 10 seconds | Timeout for script execution |
| `DefaultCacheTTL` | 5 seconds | Cache duration for owner info |
| `DefaultPort` | 8080 | HTTP server port |

### Environment Variables

The shell script respects these environment variables:

- `README_FILE`: Path to README file (default: `README.md`)
- `DEBUG`: Set to `1` to enable debug logging in the script

## API Endpoints

### GET /api/load

Loads the Kanban board from README.md and enriches it with owner information.

**Response:**
```json
{
  "columns": [
    {
      "name": "To Do",
      "cards": [
        {
          "id": "To Do-0",
          "title": "Task 1",
          "checked": false,
          "assignee": "Alice"
        }
      ]
    }
  ]
}
```

**Error Handling:**
- Returns 500 if README.md cannot be parsed
- Owner info failures are logged but don't fail the request (cards will have empty assignee)

### POST /api/save

Saves the board state back to README.md and creates a git commit.

**Request Body:**
```json
{
  "columns": [...]
}
```

**Response:**
```json
{
  "status": "ok"
}
```

**Behavior:**
- Rewrites the Kanban section in README.md
- Preserves content before and after the board
- Creates a git commit with message: "kanban: updated board via web UI"
- Invalidates the owner cache

### POST /api/claim

Claims or unclaims a ticket (creates a commit with "kanban:" prefix for owner tracking).

**Request Body:**
```json
{
  "lane_index": 0,
  "ticket_index": 1,
  "owner": "Alice",
  "action": "claim"
}
```

**Response:**
Returns the updated board with fresh owner information.

**Behavior:**
- Creates a commit with subject: "kanban: Alice claims 'Task Name' in To Do"
- For unclaim: "kanban: unclaim 'Task Name' from To Do"
- The "kanban:" prefix ensures the commit is recognized by git blame owner inference

## Owner Inference

### How It Works

1. The shell script runs `git blame` on each ticket line in README.md
2. For each commit found, it checks if the subject starts with "kanban:" (case-insensitive)
3. If not, it adds the SHA to an ignore list and re-runs git blame (requires Git 2.23+)
4. The first commit with "kanban:" prefix found is the owner
5. If no "kanban:" commit is found, the owner is empty

### Commit Message Convention

**All commits that should count for ownership MUST start with "kanban:"**

Examples:
- ✅ `kanban: Alice claims 'Implement feature X'`
- ✅ `kanban: moved 'Fix bug Y' from Backlog to In Progress`
- ✅ `kanban: unclaim 'Task Z' from Done`
- ❌ `Fixed Task X` (will be ignored by owner inference)
- ❌ `Update board` (will be ignored by owner inference)

### Limitations

- Requires Git 2.23+ for `--ignore-revs-file` support
- On older Git versions, only the most recent commit is checked
- Owner inference can be slow on large repositories (mitigated by caching)

## Performance and Caching

### Owner Cache

The server caches owner information for `DefaultCacheTTL` (5 seconds) to avoid repeated script executions.

- Cache is invalidated on save/claim operations
- Multiple rapid API calls will reuse cached data
- Cache stores `[]TicketInfo` in memory

### Script Performance

The shell script has several performance safeguards:

- `MAX_ITER=200`: Limits blame iterations to prevent infinite loops
- Efficient AWK parsing for README.md
- Conditional git blame (only runs if in a git repository)

### Optimization Tips

1. **Increase Cache TTL**: For large teams, increase `DefaultCacheTTL` to reduce script runs
2. **Async Owner Loading**: Future enhancement could load board immediately and fetch owners asynchronously
3. **Pre-commit Hook**: Use a git hook to ensure all commits have "kanban:" prefix

## Security Considerations

### Script Execution

1. **Fixed Script Path**: The script path is hardcoded or configured, not user-provided
2. **Sanitized Environment**: Script runs with minimal environment variables
3. **Timeout Enforcement**: Script execution is limited by `DefaultTimeout`
4. **No User Input to Script**: The script only reads README.md from the working directory

### Git Operations

1. **Working Directory**: Server must run from repository root
2. **Git Credentials**: Uses the current user's git configuration
3. **Commit Author**: Uses `git config user.name` and `user.email`

### File Access

1. **README.md Only**: Server only reads/writes README.md in current directory
2. **No Path Traversal**: All paths are relative to working directory
3. **Git Tracking**: Only git-tracked files are used by blame

**⚠️ Security Notice:**
- Run the server only in trusted repositories
- Do not expose the server to the public internet without authentication
- Ensure proper git configuration (user.name, user.email) before running

## Browser UI Features

### Drag and Drop

- Click and drag cards between columns
- Visual feedback during drag (opacity, cursor changes)
- Drop zones highlight on hover
- Cards automatically check when moved to "Done" column

### Claim/Unclaim

- Double-click a card to claim or unclaim
- Prompts for owner name on claim
- Creates a git commit with "kanban:" prefix
- Owner updates appear immediately after claim

### Keyboard Shortcuts

- **Ctrl+S**: Save and commit changes
- **Ctrl+R / F5**: Reload board
- **?**: Show help modal
- **Tab**: Navigate between cards
- **Enter**: Claim/unclaim focused card
- **Escape**: Close modals

### Accessibility

- Semantic HTML structure
- Focus indicators for keyboard navigation
- ARIA labels for screen readers
- Keyboard-only operation supported

## Testing

### Go Tests

Run the test suite:

```bash
cd cmd/git-kanban-web
go test -v
```

Tests cover:
- TSV parsing (`TestRunLanesScript`)
- Owner merging (`TestMergeOwnersIntoBoard`, `TestMergeOwnersTitleFallback`)
- Board parsing (`TestParseBoard`)
- Board writing (`TestWriteBoard`)

### Shell Script Tests

Run the shell integration test:

```bash
./tests/test-lanes.sh
```

Tests cover:
- Owner inference with "kanban:" prefix filtering
- TSV output format validation
- Git blame with ignore list

### Manual Testing

1. Start the server: `./cmd/git-kanban-web/git-kanban-web`
2. Open http://localhost:8080/static/index.html in a browser
3. Test drag-and-drop by moving cards
4. Test claim by double-clicking a card
5. Verify commits with `git log`

## Troubleshooting

### Server won't start

- Ensure you're in the repository root
- Check that port 8080 is available
- Verify `git-kanban-lanes.sh` exists and is executable

### Owner info not showing

- Check server logs for script errors
- Verify git blame works: `git blame README.md`
- Ensure commits have "kanban:" prefix
- Check Git version: `git --version` (need 2.23+ for full functionality)

### Board not loading

- Check browser console for errors
- Verify README.md has a Kanban section (see README.md for format)
- Check server logs: `tail -f /tmp/server.log`

### Drag-and-drop not working

- Ensure JavaScript is enabled
- Try a different browser (Chrome, Firefox, Safari supported)
- Check browser console for errors

## Development

### Adding New Features

1. **New API Endpoint**: Add handler function and register in `main()`
2. **Board Logic**: Modify `parseBoard()` or `writeBoard()` functions
3. **Owner Logic**: Modify `git-kanban-lanes.sh` or `runLanesScript()`
4. **UI Changes**: Edit files in `static/` directory (changes require rebuild)

### Embedding Static Files

Static files are embedded using `go:embed`:

```go
//go:embed static/*
var staticFiles embed.FS
```

After modifying static files, rebuild the binary to include changes.

## Future Enhancements

- [ ] Async owner loading (load board fast, fetch owners in background)
- [ ] WebSocket for real-time updates
- [ ] User authentication
- [ ] Multi-file board support
- [ ] Customizable commit message templates
- [ ] Board statistics and analytics
- [ ] Export to other formats (JSON, CSV)
- [ ] Integration with GitHub/GitLab APIs

## License

Same as the main git-kanban project.
