// Package charmdiff fornece visualizacao de diffs no terminal com estilos lipgloss.
//
// Suporta dois modos de exibicao: unified diff e side-by-side diff,
// ambos com numeracao de linhas, cores configuraveis e linhas de contexto.
//
// O formato do diff unificado acompanha o "git diff": os cabecalhos de hunk
// (@@ -a,b +c,d @@), a selecao das linhas e a ordem das linhas sao identicas
// as do git.
package charmdiff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const (
	// minLineNumWidth e a largura minima reservada para os numeros de linha.
	minLineNumWidth = 4
	// lineNumGap e o espaco entre os numeros de linha e o conteudo.
	lineNumGap = 1
	// sepWidth e a largura do separador " | " usado no modo lado a lado.
	sepWidth = 3
)

// Opcoes de configuracao para geracao de diffs.
type Options struct {
	// Quantidade de linhas de contexto ao redor de cada mudanca (padrao: 3).
	//
	// Valores negativos sao normalizados para 0 (diff sem linhas de
	// contexto) em vez de provocar panic ou esconder as mudancas.
	ContextLines int
	// Estilo visual para renderizacao.
	//
	// O valor zero (DiffStyle{}) e substituido pelo estilo padrao
	// (DefaultStyle), evitando uma renderizacao silenciosa sem cores.
	Style DiffStyle
}

// DefaultOptions retorna opcoes padrao com 3 linhas de contexto e estilo terminal.
func DefaultOptions() Options {
	return Options{
		ContextLines: 3,
		Style:        DefaultStyle(),
	}
}

// normalizeOptions valida e normaliza as opcoes recebidas pelo usuario:
//
//   - ContextLines negativo e tratado como 0 (sem linhas de contexto);
//   - Style no valor zero e substituido pelo estilo padrao, para que o diff
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
//
// As opcoes sao normalizadas antes do uso: um ContextLines negativo vira 0 e
// um Style no valor zero vira o estilo padrao.
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
		// Cabecalho do hunk
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

// SideBySideDiffWithOptions gera um diff lado a lado com opcoes personalizadas.
//
// A largura informed e o limite superior da linha renderizada: quando o
// terminal e estreito demais para caber duas colunas uteis, a largura e
// elevada para o minimo viavel (veja minSideBySideWidth).
//
// O modo lado a lado mostra o arquivo inteiro (as diferencas sao destacadas,
// mas o contexto nao e cortado); Options.ContextLines so vale para o diff
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

	// A largura e repartida entre as duas colunas, descontando os numeros de
	// linha, o espaco que os separa do texto e o separador central.
	numWidth := lineNumWidthFor(len(oldLines), len(newLines))
	width = clampSideBySideWidth(width, numWidth)
	overhead := 2*(numWidth+lineNumGap) + sepWidth
	colWidth := (width - overhead) / 2
	if colWidth < 1 {
		colWidth = 1
	}

	separator := style.Separator.Render(" | ")

	// Agrupa operacoes em pares para exibicao lado a lado
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

		// Truncar ou preencher para largura fixa (medida em celulas de
		// exibicao, nunca em bytes).
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

// lineNumWidthFor calcula a largura necessaria para exibir os numeros de
// linha dos dois arquivos. A largura nunca e menor que minLineNumWidth, o que
// mantem a alinhacao dos arquivos pequenos.
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

// lineNumPrefixWidth e a largura total do prefixo de numeros de linha
// (numero antigo, espaco, numero novo e espaco).
func lineNumPrefixWidth(numWidth int) int {
	return 2*numWidth + 2*lineNumGap
}

// minSideBySideWidth e a menor largura viavel para o modo lado a lado:
// dois numeros de linha, dois espacos, o separador e uma celula de conteudo
// em cada coluna.
func minSideBySideWidth(numWidth int) int {
	return 2*(numWidth+lineNumGap) + sepWidth + 2
}

// clampSideBySideWidth garante que a largura informada comporta o layout
// minimo do modo lado a lado.
func clampSideBySideWidth(width, numWidth int) int {
	if min := minSideBySideWidth(numWidth); width < min {
		return min
	}
	return width
}

// Formata numeros de linha para exibicao.
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

// padOrTruncate trunca ou preenche a string ate a largura informada, medida
// em celulas de exibicao (e nunca em bytes, para nao cortar glifos
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
// O "\r" e descartado porque, mantido dentro do texto estilizado, faz o
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
