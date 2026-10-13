// Package charmdiff fornece visualização de diffs no terminal com estilos lipgloss.
//
// Suporta dois modos de exibição: unified diff e side-by-side diff,
// ambos com numeração de linhas, cores configuráveis e linhas de contexto.
//
// O formato do diff unificado acompanha o "git diff": os cabeçalhos de hunk
// (@@ -a,b +c,d @@), a seleção das linhas e a ordem das linhas são idênticas
// às do git.
package charmdiff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const (
	// minLineNumWidth é a largura mínima reservada para os números de linha.
	minLineNumWidth = 4
	// lineNumGap é o espaço entre os números de linha e o conteúdo.
	lineNumGap = 1
	// sepWidth é a largura do separador " | " usado no modo lado a lado.
	sepWidth = 3
)

// Opções de configuração para geração de diffs.
type Options struct {
	// Quantidade de linhas de contexto ao redor de cada mudança (padrão: 3).
	//
	// Valores negativos são normalizados para 0 (diff sem linhas de
	// contexto) em vez de provocar panic ou esconder as mudanças.
	ContextLines int
	// Estilo visual para renderização.
	//
	// O valor zero (DiffStyle{}) é substituído pelo estilo padrão
	// (DefaultStyle), evitando uma renderização silenciosa sem cores.
	Style DiffStyle
}

// DefaultOptions retorna opções padrão com 3 linhas de contexto e estilo terminal.
func DefaultOptions() Options {
	return Options{
		ContextLines: 3,
		Style:        DefaultStyle(),
	}
}

// normalizeOptions valida e normaliza as opções recebidas pelo usuário:
//
//   - ContextLines negativo é tratado como 0 (sem linhas de contexto);
//   - Style no valor zero é substituído pelo estilo padrão, para que o diff
//     nunca seja renderizado sem cores de forma silenciosa.
func normalizeOptions(opts Options) Options {
	if opts.ContextLines < 0 {
		opts.ContextLines = 0
	}
	if opts.Style.isZero() {
		opts.Style = DefaultStyle()
	}
	return opts
}

// Tipo de operação em cada linha do diff.
type opKind int

const (
	opEqual  opKind = iota // Linha inalterada
	opInsert               // Linha adicionada
	opDelete               // Linha removida
)

// Representa uma operação no diff.
type diffOp struct {
	Kind    opKind
	OldLine int    // Número da linha no texto antigo (-1 se não aplicável)
	NewLine int    // Número da linha no texto novo (-1 se não aplicável)
	Text    string // Conteúdo da linha
}

// UnifiedDiff gera um diff no formato unificado entre dois textos.
func UnifiedDiff(old, new string) string {
	return UnifiedDiffWithOptions(old, new, DefaultOptions())
}

// UnifiedDiffWithOptions gera um diff unificado com opções personalizadas.
//
// As opções são normalizadas antes do uso: um ContextLines negativo vira 0 e
// um Style no valor zero vira o estilo padrão.
func UnifiedDiffWithOptions(old, new string, opts Options) string {
	opts = normalizeOptions(opts)

	oldLines := splitLines(old)
	newLines := splitLines(new)
	res := computeDiffFull(oldLines, newLines)
	hunks := buildHunks(res, opts.ContextLines)

	if len(hunks) == 0 {
		return ""
	}

	numWidth := lineNumWidthFor(len(oldLines), len(newLines))

	var buf strings.Builder
	style := opts.Style

	for _, hunk := range hunks {
		// Cabeçalho do hunk
		buf.WriteString(style.HunkHeader.Render(formatHunkHeader(hunk)))
		buf.WriteString("\n")

		for _, op := range hunk.ops {
			lineNum := formatLineNumbers(op, style, numWidth)
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

// SideBySideDiffWithOptions gera um diff lado a lado com opções personalizadas.
//
// A largura informada é o limite superior da linha renderizada: quando o
// terminal é estreito demais para caber duas colunas úteis, a largura é
// elevada para o mínimo viável (veja minSideBySideWidth).
//
// O modo lado a lado mostra o arquivo inteiro (as diferenças são destacadas,
// mas o contexto não é cortado); Options.ContextLines só vale para o diff
// unificado.
func SideBySideDiffWithOptions(old, new string, width int, opts Options) string {
	opts = normalizeOptions(opts)

	oldLines := splitLines(old)
	newLines := splitLines(new)
	ops := computeDiff(oldLines, newLines)

	if len(ops) == 0 {
		return ""
	}

	style := opts.Style

	// A largura é repartida entre as duas colunas, descontando os números de
	// linha, o espaço que os separa do texto e o separador central.
	numWidth := lineNumWidthFor(len(oldLines), len(newLines))
	width = clampSideBySideWidth(width, numWidth)
	overhead := 2*(numWidth+lineNumGap) + sepWidth
	colWidth := (width - overhead) / 2
	if colWidth < 1 {
		colWidth = 1
	}

	separator := style.Separator.Render(" | ")

	// Agrupa operações em pares para exibição lado a lado
	pairs := buildSideBySidePairs(ops)

	var buf strings.Builder
	for _, pair := range pairs {
		leftNum := strings.Repeat(" ", numWidth)
		leftText := ""
		rightNum := strings.Repeat(" ", numWidth)
		rightText := ""

		if pair.left != nil {
			leftNum = fmt.Sprintf("%*d", numWidth, pair.left.OldLine+1)
			leftText = pair.left.Text
		}
		if pair.right != nil {
			rightNum = fmt.Sprintf("%*d", numWidth, pair.right.NewLine+1)
			rightText = pair.right.Text
		}

		// Truncar ou preencher para largura fixa (medida em células de
		// exibição, nunca em bytes).
		leftText = padOrTruncate(leftText, colWidth)
		rightText = padOrTruncate(rightText, colWidth)

		// Aplicar estilo baseado no tipo de operação
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

		buf.WriteString(style.LineNumber.Render(leftNum))
		buf.WriteString(" ")
		buf.WriteString(leftText)
		buf.WriteString(separator)
		buf.WriteString(style.LineNumber.Render(rightNum))
		buf.WriteString(" ")
		buf.WriteString(rightText)
		buf.WriteString("\n")
	}

	return buf.String()
}

// Par de linhas para exibição lado a lado.
type sidePair struct {
	left  *diffOp
	right *diffOp
}

// Agrupa operações em pares para exibição lado a lado.
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
			// Verifica se a próxima operação é um insert (modificação)
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

// lineNumWidthFor calcula a largura necessária para exibir os números de
// linha dos dois arquivos. A largura nunca é menor que minLineNumWidth, o que
// mantém a alinhação dos arquivos pequenos.
func lineNumWidthFor(oldCount, newCount int) int {
	total := oldCount
	if newCount > total {
		total = newCount
	}

	digits := 1
	for total >= 10 {
		total /= 10
		digits++
	}

	if digits < minLineNumWidth {
		return minLineNumWidth
	}
	return digits
}

// lineNumPrefixWidth é a largura total do prefixo de números de linha
// (número antigo, espaço, número novo e espaço).
func lineNumPrefixWidth(numWidth int) int {
	return 2*numWidth + 2*lineNumGap
}

// minSideBySideWidth é a menor largura viável para o modo lado a lado:
// dois números de linha, dois espaços, o separador e uma célula de conteúdo
// em cada coluna.
func minSideBySideWidth(numWidth int) int {
	return 2*(numWidth+lineNumGap) + sepWidth + 2
}

// clampSideBySideWidth garante que a largura informada comporta o layout
// mínimo do modo lado a lado.
func clampSideBySideWidth(width, numWidth int) int {
	if min := minSideBySideWidth(numWidth); width < min {
		return min
	}
	return width
}

// Formata números de linha para exibição.
func formatLineNumbers(op diffOp, style DiffStyle, numWidth int) string {
	old := strings.Repeat(" ", numWidth)
	new := strings.Repeat(" ", numWidth)
	if op.OldLine >= 0 {
		old = fmt.Sprintf("%*d", numWidth, op.OldLine+1)
	}
	if op.NewLine >= 0 {
		new = fmt.Sprintf("%*d", numWidth, op.NewLine+1)
	}
	return style.LineNumber.Render(old+" "+new) + " "
}

// padOrTruncate trunca ou preenche a string até a largura informada, medida
// em células de exibição (e nunca em bytes, para não cortar glifos
// multibyte pela metade).
func padOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}

	w := ansi.StringWidth(s)
	if w <= width {
		return s + strings.Repeat(" ", width-w)
	}

	if width > 3 {
		return ansi.Truncate(s, width, "...")
	}
	return ansi.Truncate(s, width, "")
}

// splitLines divide o texto em linhas, normalizando as quebras de linha
// "\r\n" (Windows) e removendo o terminador final vazio.
//
// O "\r" é descartado porque, mantido dentro do texto estilizado, faz o
// terminal devolver o cursor para a coluna 0 e corrompe o render.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}

	// Normaliza "\r\n" para "\n" (arquivos gerados no Windows).
	s = strings.ReplaceAll(s, "\r\n", "\n")

	lines := strings.Split(s, "\n")
	// Remove linha vazia final causada por \n no fim do texto
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
