package charmdiff

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestUnifiedDiffSemMudancas verifica que textos idênticos não produzem diff.
func TestUnifiedDiffSemMudancas(t *testing.T) {
	texto := "line1\nline2\nline3\n"
	if got := UnifiedDiff(texto, texto); got != "" {
		t.Errorf("esperava diff vazio para textos iguais, obteve: %q", got)
	}
}

// TestUnifiedDiffCabecalhoGit verifica o formato do cabeçalho de hunk no
// mesmo estilo do git (a contagem ",1" é omitida quando há uma única linha).
func TestUnifiedDiffCabecalhoGit(t *testing.T) {
	casos := []struct {
		nome    string
		old     string
		new     string
		context int
		want    string
	}{
		{
			nome:    "uma linha em cada arquivo",
			old:     "a\n",
			new:     "b\n",
			context: 3,
			want:    "@@ -1 +1 @@",
		},
		{
			nome:    "várias linhas",
			old:     "a\nb\nc\n",
			new:     "a\nX\nc\n",
			context: 3,
			want:    "@@ -1,3 +1,3 @@",
		},
		{
			nome:    "inserção em arquivo vazio",
			old:     "",
			new:     "a\nb\n",
			context: 3,
			want:    "@@ -0,0 +1,2 @@",
		},
		{
			nome:    "sem contexto",
			old:     "a\nb\nc\n",
			new:     "a\nx\nb\nc\n",
			context: 0,
			want:    "@@ -1,0 +2 @@",
		},
	}

	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			got := UnifiedDiffWithOptions(tc.old, tc.new,
				Options{ContextLines: tc.context, Style: DefaultStyle()})
			linhas := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
			if len(linhas) == 0 {
				t.Fatal("diff vazio")
			}
			if got := ansi.Strip(linhas[0]); got != tc.want {
				t.Errorf("cabeçalho incorreto:\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestUnifiedDiffConteudo verifica o conteúdo das linhas do diff (prefixo,
// marcador e texto).
func TestUnifiedDiffConteudo(t *testing.T) {
	got := UnifiedDiffWithOptions("a\nb\nc\n", "a\nX\nc\n",
		Options{ContextLines: 3, Style: DefaultStyle()})

	linhas := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	want := []string{
		"@@ -1,3 +1,3 @@",
		"   1    1  a",
		"   2      -b",
		"        2 +X",
		"   3    3  c",
	}

	if len(linhas) != len(want) {
		t.Fatalf("número de linhas diferente: got %d, want %d\n%s",
			len(linhas), len(want), got)
	}
	for i := range want {
		if got := ansi.Strip(linhas[i]); got != want[i] {
			t.Errorf("linha %d incorreta:\n got: %q\nwant: %q", i, got, want[i])
		}
	}
}

// TestUnifiedDiffSemEstileUsaEstiloPadrao verifica que um DiffStyle no
// valor zero não renderiza o diff sem cores (achado 8 da auditoria).
func TestUnifiedDiffSemEstileUsaEstiloPadrao(t *testing.T) {
	got := UnifiedDiffWithOptions("a\n", "b\n", Options{ContextLines: 3})
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("esperava cores no diff com Options sem Style, obteve: %q", got)
	}

	// O mesmo vale para o modo lado a lado.
	sbs := SideBySideDiffWithOptions("a\n", "b\n", 80, Options{ContextLines: 3})
	if !strings.Contains(sbs, "\x1b[") {
		t.Errorf("esperava cores no diff lado a lado com Options sem Style, obteve: %q", sbs)
	}
}

// TestUnifiedDiffEstilePersonalizado verifica que um estilo válido é usado
// como informado.
func TestUnifiedDiffEstilePersonalizado(t *testing.T) {
	estilo := GitHubStyle()
	got := UnifiedDiffWithOptions("a\n", "b\n", Options{ContextLines: 3, Style: estilo})
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("esperava cores do estilo GitHub, obteve: %q", got)
	}
}

// TestSideBySideDiffEstrutura verifica as colunas do modo lado a lado.
func TestSideBySideDiffEstrutura(t *testing.T) {
	got := SideBySideDiff("hello\nworld\n", "hello\nearth\n", 80)
	linhas := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(linhas) != 2 {
		t.Fatalf("esperava 2 linhas, obteve %d: %q", len(linhas), got)
	}

	contexto := ansi.Strip(linhas[0])
	if !strings.Contains(contexto, "hello") {
		t.Errorf("esperava 'hello' na primeira linha, obteve: %q", contexto)
	}
	if !strings.Contains(contexto, " | ") {
		t.Errorf("esperava o separador ' | ', obteve: %q", contexto)
	}

	alterado := ansi.Strip(linhas[1])
	if !strings.Contains(alterado, "world") || !strings.Contains(alterado, "earth") {
		t.Errorf("esperava 'world' e 'earth' na segunda linha, obteve: %q", alterado)
	}
}

// TestSideBySideDiffSemMudancas mostra o arquivo nos dois lados.
func TestSideBySideDiffSemMudancas(t *testing.T) {
	got := SideBySideDiff("same\n", "same\n", 80)
	if !strings.Contains(got, "same") {
		t.Errorf("esperava 'same' no resultado, obteve: %q", got)
	}
}

// TestComputeDiffVazio verifica que entradas vazias não geram operações.
func TestComputeDiffVazio(t *testing.T) {
	if ops := computeDiff(nil, nil); len(ops) != 0 {
		t.Errorf("esperava 0 operações para entradas vazias, obteve: %d", len(ops))
	}
}

// TestSplitLines verifica a divisão em linhas.
func TestSplitLines(t *testing.T) {
	linhas := splitLines("a\nb\nc\n")
	if len(linhas) != 3 {
		t.Fatalf("esperava 3 linhas, obteve: %d", len(linhas))
	}
	if linhas[0] != "a" || linhas[1] != "b" || linhas[2] != "c" {
		t.Errorf("conteúdo inesperado: %v", linhas)
	}
}

// TestLineNumWidthFor verifica o cálculo da largura dos números de linha.
func TestLineNumWidthFor(t *testing.T) {
	casos := []struct {
		old, new int
		want     int
	}{
		{0, 0, 4},
		{1, 1, 4},
		{9999, 9999, 4},
		{10000, 5, 5},
		{1, 123456, 6},
	}
	for _, tc := range casos {
		if got := lineNumWidthFor(tc.old, tc.new); got != tc.want {
			t.Errorf("lineNumWidthFor(%d, %d) = %d, esperado %d",
				tc.old, tc.new, got, tc.want)
		}
	}
}

// TestPadOrTruncate verifica o preenchimento e a truncagem por células de
// exibição.
func TestPadOrTruncate(t *testing.T) {
	got := padOrTruncate("hello", 10)
	if ansi.StringWidth(got) != 10 {
		t.Errorf("esperava largura 10, obteve %d (%q)", ansi.StringWidth(got), got)
	}
	if got != "hello     " {
		t.Errorf("preenchimento inesperado: %q", got)
	}

	got = padOrTruncate("hello world this is long", 10)
	if ansi.StringWidth(got) != 10 {
		t.Errorf("esperava largura 10, obteve %d (%q)", ansi.StringWidth(got), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("esperava o sufixo '...', obteve: %q", got)
	}
}
