#!/bin/sh
set -eu

[ "$#" -le 1 ] || { echo "Usage: install-completions.sh [completion-directory]" >&2; exit 1; }
[ -n "${HOME:-}" ] || { echo "HOME is not set" >&2; exit 1; }
script_dir="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
completion_dir="${1:-$script_dir/completions}"
for shell in bash zsh fish; do
  [ -f "$completion_dir/$shell" ] || { echo "Missing completion file: $completion_dir/$shell" >&2; exit 1; }
done
failed=0

# Completion installation is best-effort: a missing completion framework or an
# unwritable directory must not undo a successful binary installation. Never
# edit shell startup files or request elevated privileges.
install_completion() {
  source_path="$completion_dir/$1"
  destination="$2"
  [ -f "$source_path" ] || return 0 # Older releases have no completion files.
  if (
    mkdir -p "$(dirname "$destination")" || exit 1
    completion_stage="$(mktemp "$(dirname "$destination")/.toggl-completion.XXXXXX")" || exit 1
    trap 'rm -f "$completion_stage"' EXIT HUP INT TERM
    cp "$source_path" "$completion_stage" &&
    chmod 644 "$completion_stage" &&
    mv "$completion_stage" "$destination"
  ); then
    echo "Installed $1 completion to $destination"
  else
    failed=1
    echo "Could not install $1 completion to $destination; run the completion installer again." >&2
  fi
}

data_dir="${XDG_DATA_HOME:-$HOME/.local/share}"
install_completion bash "${BASH_COMPLETION_USER_DIR:-$data_dir/bash-completion}/completions/toggl"
install_completion fish "${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions/toggl.fish"

if [ -f "$completion_dir/zsh" ]; then
  zsh_completion_dir=""
  if command -v zsh >/dev/null 2>&1; then
    # -f avoids loading user startup files. Choose an existing writable site
    # directory already in zsh's default fpath (including Homebrew builds).
    zsh_paths="$(zsh -fc 'printf "%s\n" "${fpath[@]}"' 2>/dev/null)" || zsh_paths=""
    while IFS= read -r directory; do
      case "$directory" in
        */site-functions)
          if [ -d "$directory" ] && [ -w "$directory" ]; then
            zsh_completion_dir="$directory"
            break
          fi
          ;;
      esac
    done <<EOF
$zsh_paths
EOF
  fi
  if [ -n "$zsh_completion_dir" ]; then
    install_completion zsh "$zsh_completion_dir/_toggl"
  else
    install_completion zsh "$data_dir/zsh/site-functions/_toggl"
    echo "For zsh, place _toggl in a directory on your fpath and enable compinit. No writable default site-functions directory was found."
  fi
  echo "Open a new shell to load completions. Bash requires bash-completion; zsh requires compinit."
fi

exit "$failed"
