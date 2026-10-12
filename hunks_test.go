package charmdiff

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestHunksSobrepostos reproduz o caso da auditoria: a janela do hunk
// comecava por uma insercao (OldLine == -1) e a fusao dos hunks era pulada,
// produzindo dois hunks sobrepostos.
//
// Antes: "@@ -1,4 +1,3 @@" seguido de "@@ -2,3 +1,4 @@".
// Agora: um unico hunk "@@ -1,4 +1,4 @@".
func TestHunksSobrepostos(t *testing.T) {
	old := "o00\no01\no02\no03\n"
	new := "n00\nn01\no02\nn03\n"

	got := UnifiedDiffWithOptions(old, new, Options{ContextLines: 3, Style: DefaultStyle()})
	linhas := strings.Split(strings.TrimSuffix(got, "\n"), "\n")

	cabecalhos := 0
	for _, linha := range linhas {
		if strings.HasPrefix(ansi.Strip(linha), "@@") {
			cabecalhos++
		}
	}

	if cabecalhos != 1 {
		t.Fatalf("esperava 1 hunk, obteve %d:\n%s", cabecalhos, got)
	}
	if cabecalho := ansi.Strip(linhas[0]); cabecalho != "@@ -1,4 +1,4 @@" {
		t.Errorf("cabecalho incorreto:\n got: %q\nwant: %q", cabecalho, "@@ -1,4 +1,4 @@")
	}
}

// TestHunkSobrepostoComInsercaoNoInicio verifica que uma insercao no inicio
// do arquivo nao gera hunks sobrepostos.
func TestHunkSobrepostoComInsercaoNoInicio(t *testing.T) {
	old := "a\nb\nc\nd\n"
	new := "X\na\nb\nc\nd\n"

	got := UnifiedDiffWithOptions(old, new, Options{ContextLines: 3, Style: DefaultStyle()})

	if n := strings.Count(ansi.Strip(got), "@@ -"); n != 1 {
		t.Errorf("esperava 1 hunk, obteve %d:\n%s", n, got)
	}
}

// TestHunkSomenteInsercao verifica o cabecalho de um hunk que contem apenas
// linhas adicionadas: o git usa a posicao anterior a insercao.
func TestHunkSomenteInsercao(t *testing.T) {
	casos := []struct {
		nome string
		old  string
		new  string
		want string
	}{
		{"arquivo vazio", "", "a\nb\n", "@@ -0,0 +1,2 @@"},
		{"insercao no inicio", "a\n", "x\na\n", "@@ -1 +1,2 @@"},
		{"insercao no meio", "a\nb\nc\n", "a\nx\nb\nc\n", "@@ -1,3 +1,4 @@"},
		{"insercao no fim", "a\nb\nc\n", "a\nb\nc\nd\n", "@@ -1,3 +1,4 @@"},
	}

	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			got := UnifiedDiffWithOptions(tc.old, tc.new,
				Options{ContextLines: 3, Style: DefaultStyle()})
			linhas := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
			if got := ansi.Strip(linhas[0]); got != tc.want {
				t.Errorf("cabecalho incorreto:\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestHunksSeparadosPorDistancia verifica que mudancas distantes geram hunks
// separados, com a quantidade de contexto correta.
func TestHunksSeparadosPorDistancia(t *testing.T) {
	linhas := make([]string, 20)
	for i := range linhas {
		linhas[i] = "linha " + string(rune('a'+i%26))
	}
	old := strings.Join(linhas, "\n") + "\n"

	novas := append([]string(nil), linhas...)
	novas[1] = "ALTERADA"
	novas[15] = "TAMBEM ALTERADA"
	new := strings.Join(novas, "\n") + "\n"

	got := UnifiedDiffWithOptions(old, new, Options{ContextLines: 3, Style: DefaultStyle()})

	if n := strings.Count(ansi.Strip(got), "@@ -"); n != 2 {
		t.Errorf("esperava 2 hunks, obteve %d:\n%s", n, got)
	}
}

// TestBuildHunksContextoNegativoNaoQuebra verifica que um contexto negativo
// e tratado como 0 (nenhum panic, nenhuma mudanca escondida).
func TestBuildHunksContextoNegativoNaoQuebra(t *testing.T) {
	res := computeDiffFull(splitLines("a\nb\nc\n"), splitLines("a\nX\nc\n"))

	hunks := buildHunks(res, -5)
	if len(hunks) != 1 {
		t.Fatalf("esperava 1 hunk, obteve %d", len(hunks))
	}
	if hunks[0].oldCount != 1 || hunks[0].newCount != 1 {
		t.Errorf("contagens inesperadas: old=%d new=%d", hunks[0].oldCount, hunks[0].newCount)
	}
}
