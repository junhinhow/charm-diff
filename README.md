# charm-diff

Beautiful terminal diff viewer for the [Charmbracelet](https://charm.sh) ecosystem. Produces unified and side-by-side diffs with [Lip Gloss](https://github.com/charmbracelet/lipgloss) styling.

> **[Leia em Portugues](README.pt-br.md)**

## Features

- **Unified diff** — classic format with `+`/`-` markers and line numbers
- **Side-by-side diff** — two-column comparison at any terminal width
- **Configurable context lines** — show as many surrounding lines as you want (default: 3)
- **Predefined styles** — GitHub-inspired and minimal terminal themes
- **Custom styles** — build your own `DiffStyle` with any Lip Gloss style

## Install

```bash
go get github.com/junhinhow/charm-diff@latest
```

## Usage

### Unified Diff

```go
package main

import (
    "fmt"
    diff "github.com/junhinhow/charm-diff"
)

func main() {
    old := "func hello() {\n\tfmt.Println(\"hello\")\n}\n"
    new := "func hello() {\n\tfmt.Println(\"hello, world!\")\n}\n"

    fmt.Print(diff.UnifiedDiff(old, new))
}
```

### Side-by-Side Diff

```go
fmt.Print(diff.SideBySideDiff(old, new, 120))
```

### Custom Options

```go
opts := diff.Options{
    ContextLines: 5,
    Style:        diff.GitHubStyle(),
}

fmt.Print(diff.UnifiedDiffWithOptions(old, new, opts))
fmt.Print(diff.SideBySideDiffWithOptions(old, new, 120, opts))
```

### Available Styles

| Style | Description |
|-------|-------------|
| `TerminalStyle()` | Minimal ANSI colors (default) |
| `GitHubStyle()` | Green/red backgrounds like GitHub |
| `DiffStyle{...}` | Build your own with Lip Gloss styles |

## API Reference

### Functions

| Function | Description |
|----------|-------------|
| `UnifiedDiff(old, new string) string` | Unified diff with default options |
| `UnifiedDiffWithOptions(old, new string, opts Options) string` | Unified diff with custom options |
| `SideBySideDiff(old, new string, width int) string` | Side-by-side diff with default options |
| `SideBySideDiffWithOptions(old, new string, width int, opts Options) string` | Side-by-side diff with custom options |

### Types

| Type | Description |
|------|-------------|
| `Options` | Configuration: `ContextLines int`, `Style DiffStyle` |
| `DiffStyle` | Lip Gloss styles for each diff element |

### Style Constructors

| Constructor | Description |
|-------------|-------------|
| `DefaultStyle()` | Returns `TerminalStyle()` |
| `TerminalStyle()` | Basic ANSI colors |
| `GitHubStyle()` | GitHub-like colored backgrounds |

## License

MIT
