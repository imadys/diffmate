# diffmate

Review your working tree from the terminal before committing.

`diffmate` is a focused Git TUI for people whose daily coding routine lives in the
terminal. Run it inside a repository, scan changed and untracked files, inspect
diffs, stage or unstage files, and jump into your editor without leaving the
keyboard.

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

## Keybindings

| Key | Action |
| --- | --- |
| `j`, `down` | Move to next file |
| `k`, `up` | Move to previous file |
| `space`, `pgdown` | Scroll diff down |
| `b`, `pgup` | Scroll diff up |
| `s` | Stage selected file |
| `u` | Unstage selected file |
| `S` | Stage all changes |
| `U` | Unstage all changes |
| `c` | Open commit message box |
| `ctrl+s` | Create commit from the commit message box |
| `esc` | Cancel commit message box |
| `e`, `enter` | Open selected file in `$VISUAL`, `$EDITOR`, or `vi` |
| `r` | Refresh |
| `q`, `esc` | Quit |

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
