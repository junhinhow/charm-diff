package charmdiff

import (
	"charm.land/lipgloss/v2"
)

// DiffStyle define os estilos visuais para renderizacao de diffs.
type DiffStyle struct {
	// Estilo para linhas adicionadas
	Addition lipgloss.Style
	// Estilo para linhas removidas
	Deletion lipgloss.Style
	// Estilo para linhas de contexto (sem mudanca)
	Context lipgloss.Style
	// Estilo para cabecalho do hunk (@@...@@)
	HunkHeader lipgloss.Style
	// Estilo para numeros de linha
	LineNumber lipgloss.Style
	// Estilo para separador no modo side-by-side
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

// TerminalStyle retorna um estilo minimalista usando cores ANSI basicas.
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

// DefaultStyle retorna o estilo padrao (Terminal).
func DefaultStyle() DiffStyle {
	return TerminalStyle()
}
