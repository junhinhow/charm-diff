package charmdiff

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// TestRegressaoContextLinesNegativo verifica que um ContextLines negativo
// não provoca panic e, principalmente, não esconde as mudanças.
//
// Antes: "ContextLines: -3" causava "panic: slice bounds out of range" e
// "ContextLines: -1" devolvia um diff vazio.
func TestRegressaoContextLinesNegativo(t *testing.T) {
	old := "a\nb\nc\n"
	new := "a\nX\nc\n"

	for _, ctx := range []int{-100, -3, -1} {
		t.Run(fmt.Sprintf("contexto=%d", ctx), func(t *testing.T) {
			got := UnifiedDiffWithOptions(old, new,
				Options{ContextLines: ctx, Style: DefaultStyle()})

			if got == "" {
				t.Fatal("o diff não pode ser vazio: as mudanças sumiram")
			}
			if !strings.Contains(ansi.Strip(got), "-b") {
				t.Errorf("esperava a remoção de 'b', obteve: %q", ansi.Strip(got))
			}
			if !strings.Contains(ansi.Strip(got), "+X") {
				t.Errorf("esperava a inserção de 'X', obteve: %q", ansi.Strip(got))
			}
			// Com contexto normalizado para 0, o hunk cobre apenas a
			// linha alterada.
			if cabeçalho := ansi.Strip(strings.SplitN(got, "\n", 2)[0]); cabeçalho != "@@ -2 +2 @@" {
				t.Errorf("cabeçalho inesperado: %q", cabeçalho)
			}
		})
	}
}

// TestRegressaoContextLinesNegativoLadoALado verifica o mesmo no modo lado
// a lado.
func TestRegressaoContextLinesNegativoLadoALado(t *testing.T) {
	got := SideBySideDiffWithOptions("a\nb\nc\n", "a\nX\nc\n", 80,
		Options{ContextLines: -3, Style: DefaultStyle()})
	if got == "" {
		t.Fatal("esperava um diff lado a lado não vazio")
	}
}

// TestRegressaoCRLF verifica que as quebras de linha "\r\n" são normalizadas:
// o "\r" deixado dentro do texto estilizado faz o terminal devolver o cursor
// para a coluna 0 e corrompe o render.
func TestRegressaoCRLF(t *testing.T) {
	old := "a\r\nb\r\nc\r\n"
	new := "a\r\nX\r\nc\r\n"

	got := UnifiedDiffWithOptions(old, new, Options{ContextLines: 3, Style: DefaultStyle()})

	if strings.ContainsRune(got, '\r') {
		t.Errorf("o diff não pode conter '\\r', obteve: %q", got)
	}

	// O conteúdo precisa ser idêntico ao do texto com "\n".
	esperado := UnifiedDiffWithOptions("a\nb\nc\n", "a\nX\nc\n",
		Options{ContextLines: 3, Style: DefaultStyle()})
	if got != esperado {
		t.Errorf("saída com CRLF difere da saída com LF:\n got: %q\nwant: %q",
			got, esperado)
	}
}

// TestRegressaoCRLFSplitLines verifica a normalização na divisão das linhas.
func TestRegressaoCRLFSplitLines(t *testing.T) {
	linhas := splitLines("a\r\nb\r\nc\r\n")
	if len(linhas) != 3 {
		t.Fatalf("esperava 3 linhas, obteve: %d (%v)", len(linhas), linhas)
	}
	for i, linha := range linhas {
		if strings.ContainsRune(linha, '\r') {
			t.Errorf("linha %d ainda contém '\\r': %q", i, linha)
		}
	}
	if linhas[0] != "a" || linhas[1] != "b" || linhas[2] != "c" {
		t.Errorf("conteúdo inesperado: %v", linhas)
	}
}

// TestRegressaoLarguraSideBySide verifica que a saída lado a lado respeita a
// largura pedida.
//
// Antes: a largura era calculada sem os 2 espaços após os números de linha,
// produzindo uma linha 1-2 colunas mais larga que o pedido.
func TestRegressaoLarguraSideBySide(t *testing.T) {
	casos := []struct {
		largura int
		old     string
		new     string
	}{
		{80, "hello\n", "world\n"},
		{80, "uma linha bem mais longa que o espaco disponivel\n", "outra\n"},
		{100, strings.Repeat("a\n", 30), strings.Repeat("b\n", 30)},
		{60, "x\n", "y\n"},
		{40, "x\n", "y\n"},
	}

	for _, tc := range casos {
		t.Run(fmt.Sprintf("largura=%d", tc.largura), func(t *testing.T) {
			got := SideBySideDiff(tc.old, tc.new, tc.largura)
			for i, linha := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
				w := ansi.StringWidth(linha)
				if w > tc.largura {
					t.Errorf("linha %d com largura %d, maior que a pedida (%d): %q",
						i, w, tc.largura, ansi.Strip(linha))
				}
			}
		})
	}
}

// TestRegressaoLarguraSideBySideMultibyte verifica a largura com acentos e
// caracteres de largura dupla.
func TestRegressaoLarguraSideBySideMultibyte(t *testing.T) {
	old := "acentuação e emoji 🎉 aqui\n"
	new := "日本語のテキストと_more_\n"

	got := SideBySideDiff(old, new, 80)
	for i, linha := range strings.Split(strings.TrimSuffix(got, "\n"), "\n") {
		w := ansi.StringWidth(linha)
		if w > 80 {
			t.Errorf("linha %d com largura %d, maior que 80: %q", i, w, ansi.Strip(linha))
		}
	}
}

// TestRegressaoTruncagemMultibyte verifica que a truncagem não corta glifos
// multibyte pela metade (o resultado precisa ser UTF-8 válido).
func TestRegressaoTruncagemMultibyte(t *testing.T) {
	casos := []string{
		"áéíóú çãõ",
		"日本語のテキスト",
		"emoji 🎉🎊 aqui",
		"mixto á日🎉",
	}

	for _, texto := range casos {
		for largura := 1; largura <= 12; largura++ {
			got := padOrTruncate(texto, largura)
			if !utf8Valida(got) {
				t.Errorf("largura %d: resultado não é UTF-8 válido: %q", largura, got)
			}
			if w := ansi.StringWidth(got); w > largura {
				t.Errorf("largura %d: resultado com largura %d: %q", largura, w, got)
			}
		}
	}
}

// utf8Valida informa se a string é uma sequência UTF-8 válida.
func utf8Valida(s string) bool {
	return utf8.ValidString(s)
}

// TestRegressaoMemoriaDezMilLinhas verifica que o cálculo do diff não usa
// uma tabela O(n*m): 10.000 linhas exigiriam ~800 MB com o LCS.
//
// Com o algoritmo de Myers a memória permanece linear.
func TestRegressaoMemoriaDezMilLinhas(t *testing.T) {
	const total = 10000

	linhas := make([]string, total)
	for i := range linhas {
		linhas[i] = fmt.Sprintf("linha %d", i)
	}
	old := strings.Join(linhas, "\n") + "\n"

	linhas[total/2] = "linha alterada"
	linhas[total-1] = "ultima linha alterada"
	new := strings.Join(linhas, "\n") + "\n"

	var antes, depois runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&antes)

	got := UnifiedDiffWithOptions(old, new, Options{ContextLines: 3, Style: DefaultStyle()})

	runtime.ReadMemStats(&depois)
	alocado := depois.TotalAlloc - antes.TotalAlloc

	if alocado > 128<<20 {
		t.Errorf("memória alocada alta para %d linhas: %d MB", total, alocado>>20)
	}

	plano := ansi.Strip(got)
	if !strings.Contains(plano, "-linha 5000") {
		t.Errorf("esperava a remoção da linha 5000")
	}
	if !strings.Contains(plano, "+linha alterada") {
		t.Errorf("esperava a inserção da linha alterada")
	}
}
