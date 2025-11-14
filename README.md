# git-kanban

A POSIX shell-based git extension for managing a Markdown Kanban board in your repository (e.g., in README.md), with both a text-based interface and a web-based GUI.

## Features

- Parse Markdown Kanban boards (list format) from README.md
- **Text-based TUI** for selecting and moving tickets (no dependencies required)
- **Web-based GUI** with drag-and-drop, claim/unclaim, and visual board management
- **Ticket ownership tracking** via git blame with "kanban:" commit prefix filtering
- Auto-commit changes with descriptive messages
- Owner inference from git history

## Usage

### Terminal UI (TUI)

```sh
git kanban
```

- Launches the TUI to move tickets between columns.
- All changes are committed automatically.

### Web GUI

```sh
cd cmd/git-kanban-web
go build -o git-kanban-web
./git-kanban-web
```

Then open http://localhost:8080/static/index.html in your browser.

Features:
- **Drag & Drop**: Move tickets between lanes
- **Claim/Unclaim**: Double-click cards to claim ownership
- **Save & Commit**: Saves changes to README.md and creates git commits
- **Owner Display**: Shows who owns each ticket based on git history
- **Keyboard Shortcuts**: Ctrl+S to save, Ctrl+R to reload, ? for help

See [cmd/git-kanban-web/README_go_integration.md](cmd/git-kanban-web/README_go_integration.md) for detailed documentation.

### Owner Inference CLI

The `git-kanban-lanes.sh` script can be used standalone to query ticket ownership:

```sh
./git-kanban-lanes.sh --lanes
```

Outputs TSV format:
```
0    To Do    0    Task 1    Alice
0    To Do    1    Task 2    Bob
1    Done     0    Task 3    Charlie
```

Fields: `lane_index`, `lane_name`, `ticket_index`, `ticket_text`, `owner`

**Owner Inference Rules:**
- Only commits with subjects starting with "kanban:" (case-insensitive) count for ownership
- The script uses `git blame` with an ignore list to skip non-kanban commits
- Requires Git 2.23+ for full functionality (--ignore-revs-file support)

## Requirements

- POSIX shell (sh, bash, zsh) - for TUI and owner inference
- Git - for version control and blame-based ownership
- Go 1.21+ - for web GUI (optional)

## Commit Message Convention

For ticket ownership tracking to work, use the "kanban:" prefix in commit messages:

**Examples:**
- ✅ `kanban: Alice claims 'Implement feature X'`
- ✅ `kanban: moved 'Fix bug Y' from Backlog to In Progress`
- ✅ `kanban: unclaim 'Task Z'`
- ❌ `Fixed Task X` (will be ignored by owner inference)

The web GUI automatically creates commits with the "kanban:" prefix.

## Status

This is a prototype for discussion and extension. Not yet part of git-extras.

## How to Run (Current State)

### TUI

1. Make the script executable:
   ```sh
   chmod +x git-kanban
   ```
2. From your repo root (with a Markdown Kanban in `README.md`), run:
   ```sh
   ./git-kanban
   ```
   Or symlink it into your PATH as `git-kanban` to use as `git kanban`.

### Web GUI

1. Build the server:
   ```sh
   cd cmd/git-kanban-web
   go build -o git-kanban-web
   ```
2. Run from the repository root:
   ```sh
   cd ../..
   ./cmd/git-kanban-web/git-kanban-web
   ```
3. Open http://localhost:8080/static/index.html in your browser

## Limitations (Prototype)

- Only supports list-style Kanban boards (not table format)
- Web GUI is a basic prototype (no authentication, single-user)

## Kanban Board Format Requirements

To work with `git-kanban`, your Markdown Kanban board must follow these conventions:

- **Board Definition:** The board must be present in a file named `README.md` in your repository root.
- **Board Detection:** The script will look for a horizontal rule (`---`), followed by an H2 header containing the word `Kanban` (e.g., `## Kanban Board`).
- **Lane/Column Delimiters:** All bolded elements (e.g., `**Backlog**`, `**Ready**`) that appear under the detected Kanban board header are treated as lanes/columns. The order in the file determines the order in the UI.
- **Ticket Format:** Each ticket is a Markdown list item in the format `- [ ] Ticket text` (unchecked box). Only list-style boards are supported.
- **Ticket Parsing:** Tickets are parsed from all detected lanes. Tickets must be directly under the lane heading, not nested or in tables.
- **Moving Tickets:** When you move a ticket, it is removed from its current lane and added to the selected destination lane, preserving the list format.
  Below is a sample (but also used to track this project!) Markdown Kanban board compatible with the script:

---

## Kanban Board

**Parking Lot**

- [ ] Richer TUI: create, edit, delete, and assign tickets; filter and search
- [ ] Undo/redo and better error handling
- [ ] Optional web or desktop UI
- [ ] VS Code plugin
- [ ] Integration with other tools (e.g., notifications, metrics)
- [ ] Pluggable event hooks (e.g., run scripts on ticket move)

**Backlog**

- [ ] Board movement history by user (parse git log)
- [ ] Ticket ownership (parse git log for last touched)
- [ ] Claim ticket without it switching lanes.

**Ready**

- [ ] Refactor codebase, cleanup, DRY

**In Progress**

- [ ] Test suite and man page for git-extras inclusion
- [ ] Write documentation
- [ ] Add verbose mode with logging
- [ ] Add unit tests for all workflows

**Review/Demo**

**Done**

- [x] Parse and move tickets in all columns, not just Backlog
- [x] Configurable board file/location (not just README.md)
- [x] Finish ticket addition
- [x] Ensure ticket lists in lanes have correct linebreaks to ensure formatting stays consistent
