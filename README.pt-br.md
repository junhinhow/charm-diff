# charm-diff

Visualizador de diffs no terminal para o ecossistema [Charmbracelet](https://charm.sh). Produz diffs unificados e lado a lado com estilizacao [Lip Gloss](https://github.com/charmbracelet/lipgloss).

> **[Read in English](README.md)**

## Funcionalidades

- **Diff unificado** — formato classico com marcadores `+`/`-` e numeros de linha
- **Diff lado a lado** — comparacao em duas colunas para qualquer largura de terminal
- **Linhas de contexto configuraveis** — mostre quantas linhas ao redor quiser (padrao: 3)
- **Estilos predefinidos** — temas inspirados no GitHub e terminal minimalista
- **Estilos customizados** — crie seu proprio `DiffStyle` com qualquer estilo Lip Gloss

## Instalacao

```bash
go get github.com/junhinhow/charm-diff@latest
```

## Uso

### Diff Unificado

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

### Diff Lado a Lado

```go
fmt.Print(diff.SideBySideDiff(old, new, 120))
```

### Opcoes Personalizadas

```go
opts := diff.Options{
    ContextLines: 5,
    Style:        diff.GitHubStyle(),
}

fmt.Print(diff.UnifiedDiffWithOptions(old, new, opts))
fmt.Print(diff.SideBySideDiffWithOptions(old, new, 120, opts))
```

### Estilos Disponiveis

| Estilo | Descricao |
|--------|-----------|
| `TerminalStyle()` | Cores ANSI minimalistas (padrao) |
| `GitHubStyle()` | Fundos verde/vermelho como o GitHub |
| `DiffStyle{...}` | Crie o seu com estilos Lip Gloss |

## Referencia da API

### Funcoes

| Funcao | Descricao |
|--------|-----------|
| `UnifiedDiff(old, new string) string` | Diff unificado com opcoes padrao |
| `UnifiedDiffWithOptions(old, new string, opts Options) string` | Diff unificado com opcoes customizadas |
| `SideBySideDiff(old, new string, width int) string` | Diff lado a lado com opcoes padrao |
| `SideBySideDiffWithOptions(old, new string, width int, opts Options) string` | Diff lado a lado com opcoes customizadas |

### Tipos

| Tipo | Descricao |
|------|-----------|
| `Options` | Configuracao: `ContextLines int`, `Style DiffStyle` |
| `DiffStyle` | Estilos Lip Gloss para cada elemento do diff |

### Construtores de Estilo

| Construtor | Descricao |
|------------|-----------|
| `DefaultStyle()` | Retorna `TerminalStyle()` |
| `TerminalStyle()` | Cores ANSI basicas |
| `GitHubStyle()` | Fundos coloridos estilo GitHub |

## Licenca

MIT
