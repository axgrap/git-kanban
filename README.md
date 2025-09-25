# git-kanban

A POSIX shell-based git extension for managing a Markdown Kanban board in your repository (e.g., in README.md), with a simple text-based interface for moving tickets between columns.

## Features

- Parse Markdown Kanban boards (list format) from README.md
- Text-based UI for selecting and moving tickets (no dependencies required)
- Auto-commit changes with descriptive messages

## Usage

```sh
git kanban
```

- Launches the TUI to move tickets between columns.
- All changes are committed automatically.

## Requirements

- POSIX shell (sh, bash, zsh)

## Status

This is a prototype for discussion and extension. Not yet part of git-extras.

## How to Run (Current State)

1. Make the script executable:
   ```sh
   chmod +x proposed/git-kanban/git-kanban
   ```
2. From your repo root (with a Markdown Kanban in `README.md`), run:
   ```sh
   ./proposed/git-kanban/git-kanban
   ```
   Or symlink it into your PATH as `git-kanban` to use as `git kanban`.

## Limitations (Prototype)

- Only supports list-style Kanban boards (not table format)
- No gui for PM to manage

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
