#!/usr/bin/env bash

if command -v tmuxhelp >/dev/null 2>&1; then
  exec tmuxhelp
fi

cat <<'ROWS' | column -t -s $'\t'
Cmd Shortcut	Action
Cmd+H / Cmd+J / Cmd+K / Cmd+L	Move between tmux panes
Cmd+[ / Cmd+]	Switch tmux window
Cmd+T	Create new tmux window
Cmd+W	Close current tmux window
Cmd+1 … Cmd+9	Jump to window by index
Cmd+Shift+W	Close all windows in the session
Cmd+Shift+H/J/K/L	Resize splits (prefix + arrow)
Cmd+Backspace	Clear line (Ctrl+U)
Cmd+Shift+P	Send prefix + Ctrl+P (command palette)
Cmd+I	Show this help popup
ROWS
printf "\nCmd shortcuts mirror tmux bindings: the prefix is sent for you."
printf "\nPress any key to close..."
read -n1 -s -r
