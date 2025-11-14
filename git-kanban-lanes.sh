#!/usr/bin/env bash
# git-kanban-lanes.sh: Parse README.md Kanban board and infer ticket owners via git blame
#
# Usage: git-kanban-lanes.sh --lanes
#
# Output format (TSV): lane_index \t lane_name \t ticket_index \t ticket_text \t owner

set -eo pipefail

# Configuration
README_FILE="${README_FILE:-README.md}"
IGNORE_PREFIX="kanban:"
MAX_ITER=200
DEBUG="${DEBUG:-0}"

# Debug logging
debug() {
  if [ "$DEBUG" = "1" ]; then
    echo "[DEBUG] $*" >&2
  fi
}

# Show help
show_help() {
  cat <<EOF
Usage: git-kanban-lanes.sh [OPTIONS]

Options:
  --lanes    Output TSV with lane_index, lane_name, ticket_index, ticket_text, owner
  --help     Show this help message
  --debug    Enable debug logging (or set DEBUG=1)

Environment:
  README_FILE    Path to README file (default: README.md)

Description:
  Parses the Kanban board in README.md and infers ticket owners by analyzing
  git blame history. Only commits with subject lines beginning with "kanban:"
  (case-insensitive) are considered for ownership.

Exit codes:
  0  Success
  1  Error (missing file, no board found, etc.)
EOF
}

# Find Kanban board bounds in README.md
# Returns: "start_line end_line" or exits with error
find_board_bounds() {
  if [ ! -f "$README_FILE" ]; then
    echo "Error: $README_FILE not found" >&2
    exit 1
  fi

  # Find start: first H2 with "Kanban" after a "---" line
  local start_line
  start_line=$(awk '/^---$/ {f=1} f && /^## .*[Kk]anban/ {print NR; exit}' "$README_FILE")
  
  if [ -z "$start_line" ]; then
    echo "Error: Kanban board section not found in $README_FILE" >&2
    exit 1
  fi

  # Find end: next H2 or EOF
  local end_line
  end_line=$(awk -v s="$start_line" 'NR > s && /^## / {print NR-1; exit}' "$README_FILE")
  
  if [ -z "$end_line" ]; then
    # No next H2, use EOF
    end_line=$(wc -l < "$README_FILE")
  fi

  echo "$start_line $end_line"
}

# Infer owner for a ticket line using iterative git blame
# Args: $1=file, $2=line_number
# Returns: owner name or empty string
blame_owner_for_line() {
  local file="$1"
  local lineno="$2"
  
  # Check if we're in a git repository
  if ! git rev-parse --git-dir >/dev/null 2>&1; then
    debug "Not in a git repository"
    echo ""
    return
  fi

  # Check if file is tracked
  if ! git ls-files --error-unmatch "$file" >/dev/null 2>&1; then
    debug "File $file is not tracked by git"
    echo ""
    return
  fi

  # Create temporary ignore file
  local ignore_file
  ignore_file=$(mktemp)
  trap "rm -f '$ignore_file'" RETURN

  # Check if git supports --ignore-revs-file
  local ignore_flag=""
  if git blame --help 2>&1 | grep -q -- '--ignore-revs-file'; then
    ignore_flag="--ignore-revs-file"
    debug "Git supports --ignore-revs-file"
  else
    debug "Git does not support --ignore-revs-file (requires Git 2.23+)"
    # Fallback: without --ignore-revs-file, we'll just get the first blame result
  fi

  local iteration=0
  local prev_sha=""
  
  while [ $iteration -lt $MAX_ITER ]; do
    iteration=$((iteration + 1))
    
    # Run git blame
    local blame_cmd="git blame -L $lineno,$lineno --porcelain"
    if [ -n "$ignore_flag" ] && [ -s "$ignore_file" ]; then
      blame_cmd="$blame_cmd $ignore_flag '$ignore_file'"
    fi
    blame_cmd="$blame_cmd -- '$file'"
    
    debug "Running: $blame_cmd"
    local blame_output
    if ! blame_output=$(eval $blame_cmd 2>&1); then
      debug "Git blame failed: $blame_output"
      echo ""
      return
    fi

    # Extract SHA and author from porcelain output
    local sha
    sha=$(echo "$blame_output" | head -1 | awk '{print $1}')
    
    if [ -z "$sha" ] || [ "$sha" = "0000000000000000000000000000000000000000" ]; then
      debug "No valid SHA found in blame output"
      echo ""
      return
    fi

    # Detect infinite loop
    if [ "$sha" = "$prev_sha" ]; then
      debug "Same SHA encountered twice: $sha"
      echo ""
      return
    fi
    prev_sha="$sha"

    # Get commit subject
    local subject
    subject=$(git show -s --format=%s "$sha" 2>/dev/null || echo "")
    
    debug "SHA: $sha, Subject: $subject"

    # Check if subject starts with ignore prefix (case-insensitive)
    if echo "$subject" | grep -iq "^${IGNORE_PREFIX}"; then
      # This is a kanban commit - get the author
      local author
      author=$(git show -s --format=%an "$sha" 2>/dev/null || echo "")
      debug "Found kanban commit by: $author"
      echo "$author"
      return
    else
      # Not a kanban commit - add to ignore list and retry
      debug "Not a kanban commit, ignoring SHA: $sha"
      echo "$sha" >> "$ignore_file"
      
      # If we don't have --ignore-revs-file support, we can't continue
      if [ -z "$ignore_flag" ]; then
        debug "Cannot continue without --ignore-revs-file support"
        echo ""
        return
      fi
    fi
  done

  debug "Max iterations ($MAX_ITER) reached"
  echo ""
}

# Output lanes in TSV format
output_lanes() {
  local bounds
  bounds=$(find_board_bounds)
  local start_line end_line
  read -r start_line end_line <<< "$bounds"
  
  debug "Board bounds: $start_line to $end_line"

  # Parse lanes and tickets
  local lane_index=-1
  local current_lane=""
  local ticket_index
  
  while IFS= read -r line; do
    # Detect lane (bolded text)
    if echo "$line" | grep -qE '^[[:space:]]*\*\*[^*]+\*\*[[:space:]]*$'; then
      # Extract lane name
      current_lane=$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//;s/^\*\*//;s/\*\*$//')
      lane_index=$((lane_index + 1))
      ticket_index=0
      debug "Found lane $lane_index: $current_lane"
      continue
    fi
    
    # Detect ticket (checked or unchecked)
    if echo "$line" | grep -qE '^- \[([ x])\]'; then
      if [ -z "$current_lane" ]; then
        debug "Warning: ticket found before any lane"
        continue
      fi
      
      # Extract ticket text
      local ticket_text
      ticket_text=$(echo "$line" | sed 's/^- \[[x ]\] //')
      
      # Get line number in file for git blame
      local file_line_num
      file_line_num=$(awk -v s="$start_line" -v text="$line" 'NR >= s && $0 == text {print NR; exit}' "$README_FILE")
      
      # Infer owner
      local owner=""
      if [ -n "$file_line_num" ]; then
        owner=$(blame_owner_for_line "$README_FILE" "$file_line_num")
      fi
      
      debug "Ticket $ticket_index in lane $lane_index: $ticket_text (owner: $owner)"
      
      # Output TSV line
      printf "%d\t%s\t%d\t%s\t%s\n" "$lane_index" "$current_lane" "$ticket_index" "$ticket_text" "$owner"
      
      ticket_index=$((ticket_index + 1))
    fi
  done < <(sed -n "${start_line},${end_line}p" "$README_FILE")
}

# Main
case "${1:-}" in
  --lanes)
    output_lanes
    ;;
  --debug)
    DEBUG=1
    shift
    exec "$0" "$@"
    ;;
  --help|-h)
    show_help
    exit 0
    ;;
  "")
    echo "Error: No option specified. Use --lanes or --help" >&2
    exit 1
    ;;
  *)
    echo "Error: Unknown option: $1" >&2
    show_help
    exit 1
    ;;
esac
