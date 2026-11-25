# hop

> A fast terminal A Fast terminal directory jumper made in go inspired by [zoxide](https://github.com/ajeetdsouza/zoxide).

## Features
- **Project-Root Focus:** Automatically tracks folders containing project markers (`.git`, `go.mod`, `package.json`, `Makefile`, `Cargo.toml`, or custom markers) instead of cluttering history with random subfolders.
- **Bulk Workspace Scanner:** Recursively index all project roots under a workspace directory with `hop scan`.
- **Manual Bookmarks & Short Aliases:** Pin any folder and assign unique `@alias` shortcuts with `hop add --alias`.
- **Styled Fuzzy Search & Live Git Preview:** Pre-filtered interactive search powered by `go-fuzzyfinder` and `lipgloss` displaying tech stack badges, Git branch, dirty file counts, and latest commit details.
- **Relative Subdirectory Jumping:** Fuzzy-browse nested folders inside the current directory with `hop .` without polluting history.
- **Editor Integration:** Jump and open your configured editor immediately via `hop -e`, managed through `hop editor` and `~/.hop.yaml`.
- **Dead-Path Cleanup:** Prune deleted unpinned entries while safeguarding missing pinned bookmarks with `hop clean`.

## Installation

### Build from Source
```bash
git clone <your-repo-url>
cd hop
go mod tidy
make build-all
```
- **Linux binary:** `build/linux/hop` (move to `/usr/local/bin/hop`)
- **Windows binary:** `build/windows/hop.exe` (add folder to your system `PATH`)

## Shell Setup

Add the hook for your shell (handles `cd:<path>` and `edit:<editor>|<path>` directives automatically):

- **Zsh (`~/.zshrc`):**
  ```zsh
  eval "$(hop init zsh)"
  ```
- **Bash (`~/.bashrc`):**
  ```bash
  eval "$(hop init bash)"
  ```
- **Fish (`~/.config/fish/config.fish`):**
  ```fish
  hop init fish | source
  ```
- **PowerShell (`$PROFILE`):**
  ```powershell
  Invoke-Expression (& hop init powershell | Out-String)

## Installation

### Homebrew (macOS & Linux)

```sh
brew install --cask Rikkiuuu/tap/hop
```

### Shell Installer (macOS & Linux)

```sh
curl -fsSL [https://raw.githubusercontent.com/Rikkiuuu/hop/main/install.sh](https://raw.githubusercontent.com/Rikkiuuu/hop/main/install.sh) | sh
```

### PowerShell Installer (Windows)

```powershell
irm [https://raw.githubusercontent.com/Rikkiuuu/hop/main/install.ps1](https://raw.githubusercontent.com/Rikkiuuu/hop/main/install.ps1) | iex
```

### Go Install

```sh
go install [github.com/Rikkiuuu/hop/cmd/hop@v1.0.0](https://github.com/Rikkiuuu/hop/cmd/hop@v1.0.0)
```

### Build from Source

```sh
git clone [https://github.com/Rikkiuuu/hop.git](https://github.com/Rikkiuuu/hop.git)
cd hop
make build-all
```
```## Usage

| Command | Description |
| :--- | :--- |
| `hop` | Open interactive fuzzy search with live styled Git/stack preview |
| `hop <query>` | Jump immediately on single match, or open pre-filtered fuzzy menu |
| `hop @<alias>` | Jump directly to a directory by its explicit short alias |
| `hop .` | Fuzzy-jump into relative subdirectories of the current folder |
| `hop -e <target>` | Jump to `<query>`, `@<alias>`, or `.` and launch your configured editor |
| `hop editor [command]` | View or set your default editor in `~/.hop.yaml` (`--clear` to reset) |
| `hop add [path] [--alias <name>]` | Pin directory as a permanent bookmark with an optional unique alias |
| `hop rm [path \| @alias]` | Remove directory (defaults to cwd) or `@alias` from history |
| `hop scan [path]` | Bulk-scan `[path]` (defaults to cwd) and index new project roots |
| `hop clean` | Prune deleted unpinned entries and warn on missing pinned bookmarks |

## Configuration

User preferences are stored in `~/.hop.yaml` (optional; falls back to built-in defaults if absent):

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

Editor resolution order:
1. `editor` in `~/.hop.yaml`
2. `$EDITOR` environment variable
3. `code` (fallback default)

Navigation history, pinned bookmarks, and aliases are stored in `~/.hop.json`.

## License
Distributed under the MIT License. See `LICENSE` for more information.
MIT
