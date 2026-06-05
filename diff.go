// Package charmdiff fornece visualizacao de diffs no terminal com estilos lipgloss.
//
// Suporta dois modos de exibicao: unified diff e side-by-side diff,
// ambos com numeracao de linhas, cores configuraveis e linhas de contexto.
package charmdiff

import (
	"fmt"
	"strings"
)

// Opcoes de configuracao para geracao de diffs.
type Options struct {
	// Quantidade de linhas de contexto ao redor de cada mudanca (padrao: 3)
	ContextLines int
	// Estilo visual para renderizacao
	Style DiffStyle
}

// DefaultOptions retorna opcoes padrao com 3 linhas de contexto e estilo terminal.
func DefaultOptions() Options {
	return Options{
		ContextLines: 3,
		Style:        DefaultStyle(),
	}
}

// Tipo de operacao em cada linha do diff.
type opKind int

const (
	opEqual  opKind = iota // Linha inalterada
	opInsert               // Linha adicionada
	opDelete               // Linha removida
)

// Representa uma operacao no diff.
type diffOp struct {
	Kind    opKind
	OldLine int    // Numero da linha no texto antigo (-1 se nao aplicavel)
	NewLine int    // Numero da linha no texto novo (-1 se nao aplicavel)
	Text    string // Conteudo da linha
}

// UnifiedDiff gera um diff no formato unificado entre dois textos.
func UnifiedDiff(old, new string) string {
	return UnifiedDiffWithOptions(old, new, DefaultOptions())
}

// UnifiedDiffWithOptions gera um diff unificado com opcoes personalizadas.
func UnifiedDiffWithOptions(old, new string, opts Options) string {
	oldLines := splitLines(old)
	newLines := splitLines(new)
	ops := computeDiff(oldLines, newLines)
	hunks := buildHunks(ops, opts.ContextLines)

	if len(hunks) == 0 {
		return ""
	}

	var buf strings.Builder
	style := opts.Style

	for _, hunk := range hunks {
		// Cabecalho do hunk
		header := fmt.Sprintf("@@ -%d,%d +%d,%d @@",
			hunk.oldStart+1, hunk.oldCount, hunk.newStart+1, hunk.newCount)
		buf.WriteString(style.HunkHeader.Render(header))
		buf.WriteString("\n")

		for _, op := range hunk.ops {
			lineNum := formatLineNumbers(op, style)
			switch op.Kind {
			case opEqual:
				buf.WriteString(lineNum)
				buf.WriteString(style.Context.Render(" " + op.Text))
				buf.WriteString("\n")
			case opDelete:
				buf.WriteString(lineNum)
				buf.WriteString(style.Deletion.Render("-" + op.Text))
				buf.WriteString("\n")
			case opInsert:
				buf.WriteString(lineNum)
				buf.WriteString(style.Addition.Render("+" + op.Text))
				buf.WriteString("\n")
			}
		}
	}

	return buf.String()
}

// SideBySideDiff gera um diff lado a lado com largura total especificada.
func SideBySideDiff(old, new string, width int) string {
	return SideBySideDiffWithOptions(old, new, width, DefaultOptions())
}

// SideBySideDiffWithOptions gera um diff lado a lado com opcoes personalizadas.
func SideBySideDiffWithOptions(old, new string, width int, opts Options) string {
	oldLines := splitLines(old)
	newLines := splitLines(new)
	ops := computeDiff(oldLines, newLines)

	if len(ops) == 0 {
		return ""
	}

	style := opts.Style

	// Largura de cada coluna: metade menos espaco para separador e numeros de linha
	lineNumWidth := 4
	sepWidth := 3
	colWidth := (width - sepWidth - 2*lineNumWidth) / 2
	if colWidth < 10 {
		colWidth = 10
	}

	separator := style.Separator.Render(" | ")

	// Agrupa operacoes em pares para exibicao lado a lado
	pairs := buildSideBySidePairs(ops)

	var buf strings.Builder
	for _, pair := range pairs {
		leftNum := "    "
		leftText := ""
		rightNum := "    "
		rightText := ""

		if pair.left != nil {
			leftNum = style.LineNumber.Render(fmt.Sprintf("%4d", pair.left.OldLine+1))
			leftText = pair.left.Text
		}
		if pair.right != nil {
			rightNum = style.LineNumber.Render(fmt.Sprintf("%4d", pair.right.NewLine+1))
			rightText = pair.right.Text
		}

		// Truncar ou preencher para largura fixa
		leftText = padOrTruncate(leftText, colWidth)
		rightText = padOrTruncate(rightText, colWidth)

		// Aplicar estilo baseado no tipo de operacao
		switch {
		case pair.left != nil && pair.left.Kind == opDelete && pair.right != nil && pair.right.Kind == opInsert:
			// Linha modificada
			leftText = style.Deletion.Render(leftText)
			rightText = style.Addition.Render(rightText)
		case pair.left != nil && pair.left.Kind == opDelete:
			leftText = style.Deletion.Render(leftText)
			rightText = style.Context.Render(rightText)
		case pair.right != nil && pair.right.Kind == opInsert:
			leftText = style.Context.Render(leftText)
			rightText = style.Addition.Render(rightText)
		default:
			leftText = style.Context.Render(leftText)
			rightText = style.Context.Render(rightText)
		}

		buf.WriteString(leftNum)
		buf.WriteString(" ")
		buf.WriteString(leftText)
		buf.WriteString(separator)
		buf.WriteString(rightNum)
		buf.WriteString(" ")
		buf.WriteString(rightText)
		buf.WriteString("\n")
	}

	return buf.String()
}

// Par de linhas para exibicao lado a lado.
type sidePair struct {
	left  *diffOp
	right *diffOp
}

// Agrupa operacoes em pares para exibicao lado a lado.
func buildSideBySidePairs(ops []diffOp) []sidePair {
	var pairs []sidePair
	i := 0
	for i < len(ops) {
		op := ops[i]
		switch op.Kind {
		case opEqual:
			equalOp := op
			pairs = append(pairs, sidePair{left: &equalOp, right: &equalOp})
			i++
		case opDelete:
			// Verifica se a proxima operacao e um insert (modificacao)
			delOp := op
			if i+1 < len(ops) && ops[i+1].Kind == opInsert {
				insOp := ops[i+1]
				pairs = append(pairs, sidePair{left: &delOp, right: &insOp})
				i += 2
			} else {
				pairs = append(pairs, sidePair{left: &delOp, right: nil})
				i++
			}
		case opInsert:
			insOp := op
			pairs = append(pairs, sidePair{left: nil, right: &insOp})
			i++
		}
	}
	return pairs
}

// Formata numeros de linha para exibicao.
func formatLineNumbers(op diffOp, style DiffStyle) string {
	old := "    "
	new := "    "
	if op.OldLine >= 0 {
		old = fmt.Sprintf("%4d", op.OldLine+1)
	}
	if op.NewLine >= 0 {
		new = fmt.Sprintf("%4d", op.NewLine+1)
	}
	return style.LineNumber.Render(old+" "+new) + " "
}

// Trunca ou preenche string para largura fixa.
func padOrTruncate(s string, width int) string {
	if len(s) > width {
		if width > 3 {
			return s[:width-3] + "..."
		}
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// Divide texto em linhas removendo terminador final vazio.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	// Remove linha vazia final causada por \n no fim do texto
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// Estrutura de um hunk no diff.
type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	ops      []diffOp
}

// Constroi hunks a partir das operacoes de diff com linhas de contexto.
func buildHunks(ops []diffOp, contextLines int) []hunk {
	if len(ops) == 0 {
		return nil
	}

	// Identifica intervalos de mudancas
	type changeRange struct {
		start, end int
	}
	var changes []changeRange
	for i, op := range ops {
		if op.Kind != opEqual {
			if len(changes) == 0 || i > changes[len(changes)-1].end+1 {
				changes = append(changes, changeRange{i, i})
			} else {
				changes[len(changes)-1].end = i
			}
		}
	}

	if len(changes) == 0 {
		return nil
	}

	// Agrupa mudancas proximas em hunks
	var hunks []hunk
	for _, ch := range changes {
		start := ch.start - contextLines
		if start < 0 {
			start = 0
		}
		end := ch.end + contextLines + 1
		if end > len(ops) {
			end = len(ops)
		}

		// Mescla com hunk anterior se sobrepoem
		if len(hunks) > 0 {
			lastOps := hunks[len(hunks)-1].ops
			lastEnd := 0
			for _, o := range lastOps {
				if o.OldLine >= 0 {
					lastEnd = o.OldLine
				}
			}
			firstOp := ops[start]
			if firstOp.OldLine >= 0 && firstOp.OldLine <= lastEnd+1 {
				// Extende o hunk anterior
				for i := len(hunks[len(hunks)-1].ops); start+i-len(hunks[len(hunks)-1].ops) < end; i++ {
					idx := start + i - len(hunks[len(hunks)-1].ops)
					if idx >= start && idx < end {
						// Evita duplicatas verificando se ja esta no hunk
						found := false
						for _, existing := range hunks[len(hunks)-1].ops {
							if existing.OldLine == ops[idx].OldLine && existing.NewLine == ops[idx].NewLine {
								found = true
								break
							}
						}
						if !found {
							hunks[len(hunks)-1].ops = append(hunks[len(hunks)-1].ops, ops[idx])
						}
					}
				}
				recalculateHunk(&hunks[len(hunks)-1])
				continue
			}
		}

		h := hunk{ops: ops[start:end]}
		recalculateHunk(&h)
		hunks = append(hunks, h)
	}

	return hunks
}

// Recalcula os contadores de um hunk.
func recalculateHunk(h *hunk) {
	h.oldCount = 0
	h.newCount = 0
	h.oldStart = -1
	h.newStart = -1

	for _, op := range h.ops {
		switch op.Kind {
		case opEqual:
			h.oldCount++
			h.newCount++
			if h.oldStart < 0 && op.OldLine >= 0 {
				h.oldStart = op.OldLine
			}
			if h.newStart < 0 && op.NewLine >= 0 {
				h.newStart = op.NewLine
			}
		case opDelete:
			h.oldCount++
			if h.oldStart < 0 && op.OldLine >= 0 {
				h.oldStart = op.OldLine
			}
		case opInsert:
			h.newCount++
			if h.newStart < 0 && op.NewLine >= 0 {
				h.newStart = op.NewLine
			}
		}
	}

	if h.oldStart < 0 {
		h.oldStart = 0
	}
	if h.newStart < 0 {
		h.newStart = 0
	}
}

// Algoritmo de diff baseado em LCS (Longest Common Subsequence).
// Computa a sequencia de operacoes para transformar oldLines em newLines.
func computeDiff(oldLines, newLines []string) []diffOp {
	oldLen := len(oldLines)
	newLen := len(newLines)

	if oldLen == 0 && newLen == 0 {
		return nil
	}

	// Tabela LCS
	lcs := make([][]int, oldLen+1)
	for i := range lcs {
		lcs[i] = make([]int, newLen+1)
	}

	for i := 1; i <= oldLen; i++ {
		for j := 1; j <= newLen; j++ {
			if oldLines[i-1] == newLines[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else if lcs[i-1][j] >= lcs[i][j-1] {
				lcs[i][j] = lcs[i-1][j]
			} else {
				lcs[i][j] = lcs[i][j-1]
			}
		}
	}

	// Reconstroi as operacoes a partir da tabela LCS
	var ops []diffOp
	i, j := oldLen, newLen
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			ops = append(ops, diffOp{
				Kind:    opEqual,
				OldLine: i - 1,
				NewLine: j - 1,
				Text:    oldLines[i-1],
			})
			i--
			j--
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			ops = append(ops, diffOp{
				Kind:    opInsert,
				OldLine: -1,
				NewLine: j - 1,
				Text:    newLines[j-1],
			})
			j--
		} else if i > 0 {
			ops = append(ops, diffOp{
				Kind:    opDelete,
				OldLine: i - 1,
				NewLine: -1,
				Text:    oldLines[i-1],
			})
			i--
		}
	}

	// Inverte pois foi construido de tras pra frente
	for left, right := 0, len(ops)-1; left < right; left, right = left+1, right-1 {
		ops[left], ops[right] = ops[right], ops[left]
	}

	return ops
}
