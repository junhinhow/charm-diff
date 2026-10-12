# charm-diff

Visualizador de diffs no terminal para o ecossistema [Charmbracelet](https://charm.sh). Produz diffs unificados e lado a lado com estilização [Lip Gloss](https://github.com/charmbracelet/lipgloss).

> **[Read in English](README.md)**

## Funcionalidades

- **Diff unificado** — formato clássico com marcadores `+`/`-` e números de linha
- **Hunks compatíveis com o git** — cabeçalhos de hunk (`@@ -a,b +c,d @@`), seleção de linhas e ordem das linhas são idênticas ao `git diff -U<N>`
- **Diff lado a lado** — comparação em duas colunas que respeita a largura pedida
- **Linhas de contexto configuráveis** — mostre quantas linhas ao redor quiser (padrão: 3)
- **Estilos predefinidos** — temas inspirados no GitHub e terminal minimalista
- **Estilos customizados** — crie o seu próprio `DiffStyle` com qualquer estilo Lip Gloss
- **Memória linear** — o diff é calculado com o algoritmo de Myers, então arquivos com 10.000 linhas não esgotam a memória

## Instalação

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

### Opções Personalizadas

```go
opts := diff.Options{
    ContextLines: 5,
    Style:        diff.GitHubStyle(),
}

fmt.Print(diff.UnifiedDiffWithOptions(old, new, opts))
fmt.Print(diff.SideBySideDiffWithOptions(old, new, 120, opts))
```

### Estilos Disponíveis

| Estilo | Descrição |
|--------|-----------|
| `TerminalStyle()` | Cores ANSI minimalistas (padrão) |
| `GitHubStyle()` | Fundos verde/vermelho como o GitHub |
| `DiffStyle{...}` | Crie o seu com estilos Lip Gloss |

## Opções

| Opção | Padrão | Comportamento |
|-------|--------|---------------|
| `ContextLines` | `3` | Linhas de contexto mantidas ao redor de cada mudança. Um valor negativo é normalizado para `0` (sem contexto). |
| `Style` | `DefaultStyle()` | Estilos usados para renderizar o diff. O valor zero (`DiffStyle{}`) usa o `DefaultStyle()` em vez de renderizar silenciosamente sem cores. |

`Options.ContextLines` vale para o diff unificado. A visão lado a lado mostra sempre o arquivo inteiro e destaca as diferenças.

## Compatibilidade com o git

O diff unificado é byte a byte igual ao `git diff -U<N>` no que diz respeito aos cabeçalhos de hunk, à seleção de linhas e à ordem das linhas:

```console
$ diff.UnifiedDiff("a\nb\nc\n", "a\nX\nc\n")
@@ -1,3 +1,3 @@
   1    1  a
   2      -b
        2 +X
   3    3  c
```

- O sufixo `,1` é omitido quando o intervalo tem uma única linha, exatamente como o git.
- Um hunk que só adiciona linhas usa a posição anterior à inserção (`@@ -0,0 +1,2 @@`).
- O motor reproduz o xdiff do git (algoritmo de Myers em espaço linear), então o alinhamento dos blocos alterados é o mesmo do git.

Duas diferenças permanecem: o charm-diff não emite o marcador `\ No newline at end of file` e o cabeçalho do hunk não carrega o nome da função que o git pode acrescentar.

## Largura do modo lado a lado

`SideBySideDiff(old, new, width)` nunca renderiza uma linha mais larga que `width`: as duas colunas dividem o espaço que sobra depois dos números de linha e do separador ` | `. Larguras abaixo do layout mínimo viável (`2 * (4 + 1) + 3 + 2 = 15` colunas) são elevadas até esse mínimo.

## Referência da API

### Funções

| Função | Descrição |
|--------|-----------|
| `UnifiedDiff(old, new string) string` | Diff unificado com opções padrão |
| `UnifiedDiffWithOptions(old, new string, opts Options) string` | Diff unificado com opções customizadas |
| `SideBySideDiff(old, new string, width int) string` | Diff lado a lado com opções padrão |
| `SideBySideDiffWithOptions(old, new string, width int, opts Options) string` | Diff lado a lado com opções customizadas |

### Tipos

| Tipo | Descrição |
|------|-----------|
| `Options` | Configuração: `ContextLines int`, `Style DiffStyle` |
| `DiffStyle` | Estilos Lip Gloss para cada elemento do diff |

### Construtores de Estilo

| Construtor | Descrição |
|------------|-----------|
| `DefaultStyle()` | Retorna `TerminalStyle()` |
| `TerminalStyle()` | Cores ANSI básicas |
| `GitHubStyle()` | Fundos coloridos estilo GitHub |

## Licença

MIT
