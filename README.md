```text
████▄  ▄▄ ▄▄▄▄▄ ▄▄▄▄▄ ▄▄   ▄▄  ▄▄▄ ▄▄▄▄▄▄ ▄▄▄▄▄ 
██  ██ ██ ██▄▄  ██▄▄  ██▀▄▀██ ██▀██  ██   ██▄▄  
████▀  ██ ██    ██    ██   ██ ██▀██  ██   ██▄▄▄
```

Review your working tree from the terminal before committing.

`diffmate` is a focused Git TUI for people whose daily coding routine lives in the
terminal. Run it inside a repository, scan changed and untracked files, inspect
diffs, stage or unstage files, commit, push, and jump into your editor or coding
agent without leaving the keyboard.

## Status

Early MVP. The first goal is a small, useful pre-commit review screen rather than
a full Git client.

## Install

For now:

```sh
go install github.com/imadys/diffmate/cmd/diffmate@latest
```

From a local checkout:

```sh
make install
diffmate review
```

Planned release paths:

```sh
brew install diffmate
```

## Usage

```sh
diffmate review
```

Running `diffmate` with no arguments also opens the review screen.

If the current directory is not a Git repository, `diffmate` opens a setup screen
and can initialize Git for that directory.

The review screen uses a bento-style terminal layout:

- Changes, branches, commits, and stash live in numbered sidebar cards.
- The diff panel fills the remaining terminal width.
- The footer keeps the main workflow shortcuts visible.

## Config

Press `,` inside the app to configure:

- Visible sidebar cards.
- Preferred editor: VS Code, Zed, Cursor, or Neovim.
- Preferred coding agent: Codex, Claude, Antigravity, or Gemini.

Config is saved in your OS config directory under `diffmate/config.json`.

## Commit Suggestions

In the commit modal, press `ctrl+g` to ask the configured coding agent for a
Conventional Commit message from the current diff.

Supported headless suggestion commands:

- Codex: `codex exec` using the account-supported default model.
- Claude: `claude -p` with `haiku`.
- Gemini: `gemini -p` with `gemini-2.5-flash`.

If the selected agent is not installed or does not support a verified headless
prompt mode yet, `diffmate` shows a placeholder instead of blocking the modal.

## Keybindings

| Key                              | Action                                              |
| -------------------------------- | --------------------------------------------------- |
| `j`, `down`                      | Move to next file                                   |
| `k`, `up`                        | Move to previous file                               |
| `1`-`4`                          | Focus sidebar cards                                 |
| `5`                              | Focus diff                                          |
| `tab`                            | Cycle cards and diff                                |
| `,`                              | Open config                                         |
| `t`                              | Open config sections                                |
| `left`, `right`                  | Switch sidebar sections when sidebar is focused     |
| `]`, `right`                     | Scroll diff down one line                           |
| `[`, `left`                      | Scroll diff up one line                             |
| `space`, `f`, `pgdown`, `ctrl+d` | Scroll diff down one page                           |
| `b`, `pgup`, `ctrl+u`            | Scroll diff up one page                             |
| `g`                              | Jump to top of diff                                 |
| `G`                              | Jump to bottom of diff                              |
| `s`                              | Stage selected file                                 |
| `u`                              | Unstage selected file                               |
| `S`                              | Stage all changes                                   |
| `U`                              | Unstage all changes                                 |
| `c`                              | Open commit message box                             |
| `ctrl+g`                         | Suggest a commit message with preferred agent       |
| `ctrl+s`                         | Create commit from the commit message box           |
| `ctrl+d`                         | Clear commit message and modal errors               |
| `esc`                            | Cancel commit message box                           |
| `p`                              | Push current branch                                 |
| `o`                              | Open project in preferred editor                    |
| `a`                              | Open preferred coding agent                         |
| `e`, `enter`                     | Open selected file in `$VISUAL`, `$EDITOR`, or `vi` |
| `r`                              | Refresh                                             |
| `?`                              | Show full keymap                                    |
| `q`, `esc`                       | Quit                                                |

## Development

```sh
go test
go test ./...
go run ./cmd/diffmate review
```

## Roadmap

- Stage and unstage individual hunks.
- Toggle staged, unstaged, and all-changes views.
- Watch the repository and refresh automatically.
- Generate release binaries.
- Add Homebrew packaging.

## License

MIT
