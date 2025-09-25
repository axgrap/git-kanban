#!/bin/bash
# Usage: ./install-git-kanban.sh /path/to/git-kanban

SCRIPT="$1"
TARGET="git-kanban"

if [ -z "$SCRIPT" ] || [ ! -f "$SCRIPT" ]; then
  echo "Usage: $0 /path/to/git-kanban"
  exit 1
fi

chmod +x "$SCRIPT"

# Try to install to ~/bin, fallback to /usr/local/bin
if [ -d "$HOME/bin" ]; then
  cp "$SCRIPT" "$HOME/bin/$TARGET"
  echo "Installed to $HOME/bin/$TARGET"
elif [ -w /usr/local/bin ]; then
  sudo cp "$SCRIPT" /usr/local/bin/$TARGET
  echo "Installed to /usr/local/bin/$TARGET"
else
  echo "Could not find a suitable bin directory. Please copy manually."
  exit 2
fi

echo "You can now run: git kanban"
