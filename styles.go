package charmdiff

import (
	"reflect"

	"charm.land/lipgloss/v2"
)

// DiffStyle define os estilos visuais para renderização de diffs.
type DiffStyle struct {
	// Estilo para linhas adicionadas
	Addition lipgloss.Style
	// Estilo para linhas removidas
	Deletion lipgloss.Style
	// Estilo para linhas de contexto (sem mudança)
	Context lipgloss.Style
	// Estilo para cabeçalho do hunk (@@...@@)
	HunkHeader lipgloss.Style
	// Estilo para números de linha
	LineNumber lipgloss.Style
	// Estilo para separador no modo lado a lado
	Separator lipgloss.Style
}

// GitHubStyle retorna um estilo inspirado no GitHub com fundos coloridos.
func GitHubStyle() DiffStyle {
	return DiffStyle{
		Addition: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#22863a")).
			Background(lipgloss.Color("#e6ffec")),
		Deletion: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cb2431")).
			Background(lipgloss.Color("#ffeef0")),
		Context: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6a737d")),
		HunkHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6f42c1")).
			Background(lipgloss.Color("#f1e5ff")),
		LineNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#959da5")),
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d1d5da")),
	}
}

// TerminalStyle retorna um estilo minimalista usando cores ANSI básicas.
func TerminalStyle() DiffStyle {
	return DiffStyle{
		Addition: lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")),
		Deletion: lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")),
		Context: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		HunkHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")),
		LineNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
	}
}

// DefaultStyle retorna o estilo padrão (Terminal).
func DefaultStyle() DiffStyle {
	return TerminalStyle()
}

// isZero informa se o estilo não define nenhum atributo visual. Serve para
// detectar o DiffStyle{} (zero) informado pelo usuário e usar o estilo padrão
// em vez de renderizar o diff sem nenhuma cor.
//
// O lipgloss devolve Style{} (o valor zero) em NewStyle(), então a
// comparação estrutural identifica exatamente os estilos sem cor, sem
// atributos, sem bordas, sem preenchimento e sem transformação.
func (s DiffStyle) isZero() bool {
	return reflect.DeepEqual(s, DiffStyle{})
}
