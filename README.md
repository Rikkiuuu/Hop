# hop

> A fast terminal directory jumper made in Go, inspired by [zoxide](https://github.com/ajeetdsouza/zoxide).

## Features

- Tracks project roots (`.git`, `go.mod`, `package.json`, `Makefile`, `Cargo.toml`, or custom markers) so random subfolders stay out of history.
- Scans a workspace recursively to index existing project roots with `hop scan`.
- Pins any folder as a bookmark and assigns short `@alias` shortcuts with `hop add --alias`.
- Interactive fuzzy search (`go-fuzzyfinder` + `lipgloss`) with a live preview of tech stack, Git branch, dirty file count, and latest commit.
- Jumps into nested subdirectories of the current folder with `hop .` without adding them to history.
- Opens your configured editor on jump via `hop -e`, managed with `hop editor` and `~/.hop.yaml`.
- Prunes deleted unpinned entries with `hop clean` while warning on missing pinned bookmarks instead of removing them.

## Installation

### Homebrew (macOS & Linux)

```sh
brew install Rikkiuuu/tap/hop
```

### Shell Installer (macOS & Linux)

```sh
curl -fsSL [https://raw.githubusercontent.com/Rikkiuuu/hop/main/install.sh](https://raw.githubusercontent.com/Rikkiuuu/hop/main/install.sh) | bash
```

### PowerShell Installer (Windows)

```powershell
irm [https://raw.githubusercontent.com/Rikkiuuu/hop/main/install.ps1](https://raw.githubusercontent.com/Rikkiuuu/hop/main/install.ps1) | iex
```

### Go Install

```sh
go install [github.com/Rikkiuuu/hop/cmd/hop@latest](https://github.com/Rikkiuuu/hop/cmd/hop@latest)
```


### Build from Source

```sh
git clone [https://github.com/Rikkiuuu/hop.git](https://github.com/Rikkiuuu/hop.git)
cd hop
go mod tidy
make build-all
```

- Linux: `build/linux/hop` (move to `/usr/local/bin/hop`)
- macOS: `build/macos/amd64/hop` and `build/macos/arm64/hop` (move to `/usr/local/bin/hop`)
- Windows: `build/windows/hop.exe` (add the folder to your system `PATH`)

## Shell Setup

Add the hook for your shell (handles `cd:<path>` and `edit:<editor>|<path>` directives automatically), then restart your terminal or reload your shell config.

**Zsh (`~/.zshrc`)**
```zsh
eval "$(hop init zsh)"
```

**Bash (`~/.bashrc`)**
```bash
eval "$(hop init bash)"
```

**Fish (`~/.config/fish/config.fish`)**
```fish
hop init fish | source
```

**PowerShell (`$PROFILE`)**
```powershell
Invoke-Expression (& hop init powershell | Out-String)
```

## Usage

| Command | Description |
| :--- | :--- |
| `hop` | Open interactive fuzzy search with live Git and stack preview |
| `hop <query>` | Jump immediately on a single match, or open a pre-filtered fuzzy menu |
| `hop @<alias>` | Jump directly to a directory by its short alias |
| `hop .` | Fuzzy-jump into subdirectories of the current folder |
| `hop -e <target>` | Jump to `<query>`, `@<alias>`, or `.` and open your configured editor |
| `hop editor [command]` | View or set your default editor in `~/.hop.yaml` (`--clear` to reset) |
| `hop add [path] [--alias <name>]` | Pin a directory as a bookmark with an optional unique alias |
| `hop rm [path \| @alias]` | Remove a directory (defaults to cwd) or `@alias` from history |
| `hop scan [path]` | Scan `[path]` (defaults to cwd) and index new project roots |
| `hop clean` | Prune deleted unpinned entries and warn on missing pinned bookmarks |

## Configuration

User preferences live in `~/.hop.yaml`. The file is optional; `hop` uses these defaults when it is absent:

```yaml
markers:
  - .git
  - go.mod
  - package.json
  - Makefile
  - Cargo.toml
ignore_dirs:
  - .git
  - node_modules
  - vendor
  - .cache
  - dist
  - build
  - target
editor: nvim
```

## License

Distributed under the MIT License. See `LICENSE` for more information.
