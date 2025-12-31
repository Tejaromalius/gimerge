# Gimerge 🚀

![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go)
![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)
![Built with Bubble Tea](https://img.shields.io/badge/built%20with-Bubble%20Tea-B71C1C?style=flat-square)

A sleek TUI tool built with Go and Bubble Tea to manage your git branch merges and cleanups with style.

## Features

- **Smart Tagging**: Create structured tags on the source branch that record exactly what was merged and when.
- **Timestamped History**: Every merge tag includes a Unix timestamp for precise sorting.
- **Effortless Cleanup**: Interactive TUI to list merged tags and delete both the tags and their source branches in one go.
- **Safety First**: Prevents you from accidentally deleting the branch you currently have checked out.

## Installation

Ensure you have [Go](https://golang.org/doc/install) installed.

```bash
git clone https://github.com/Tejaromalius/gimerge/
cd gimerge
go build -o gimerge
```

## Usage

### 🏷️ Tagging a Merge

Run `gimerge` when you are on your **target branch** (e.g., `main` or `development`) after merging another branch.

```bash
./gimerge
```

1. Select the branch you just merged from the list.
2. It will create a tag in this format: `merged.{source_branch}->{your_current_branch}#sha123@1735381340`.

### 🧹 Cleaning Up

When those branches are no longer needed, run cleanup mode:

```bash
./gimerge -d
# or
./gimerge --delete
```

1. **Browse**: See a sorted list of your merges with human-readable dates.
2. **Select**: Use **Space** to toggle multiple merges you want to remove.
3. **Execute**: Press **Enter** to delete the tags and their corresponding source branches.

## Controls

| Key | Action |
| :--- | :--- |
| `↑/↓` or `j/k` | Navigate list |
| `/` | Filter/Search branches |
| `Space` | Select multiple (Cleanup mode only) |
| `Enter` | Confirm selection |
| `q` or `Ctrl+C` | Quit |

---
*Built with ❤️ using Charm's [Bubble Tea](https://github.com/charmbracelet/bubbletea).*
