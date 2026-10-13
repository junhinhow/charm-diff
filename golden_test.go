package charmdiff

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// gitUnifiedDiff executa "git diff --no-index -U<ctx>" para o par de textos
// informado e devolve as linhas de hunk geradas pelo git (cabeçalhos e
// conteúdo). O teste é pulado quando o git não está disponível.
func gitUnifiedDiff(t *testing.T, old, new string, ctx int) []string {
	t.Helper()

	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git não está disponível no PATH: comparação com o git ignorada")
	}

	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(oldPath, []byte(old), 0o600); err != nil {
		t.Fatalf("não foi possível escrever o arquivo antigo: %v", err)
	}
	if err := os.WriteFile(newPath, []byte(new), 0o600); err != nil {
		t.Fatalf("não foi possível escrever o arquivo novo: %v", err)
	}

	cmd := exec.Command(gitBin,
		"-c", "core.autocrlf=false",
		"diff", "--no-index", "--no-color",
		"--unified="+strconv.Itoa(ctx),
		"--", oldPath, newPath,
	)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		// "git diff --no-index" devolve 1 quando existem diferenças.
		if !errors.As(err, &exitErr) || exitErr.ExitCode() > 1 {
			t.Fatalf("git diff falhou: %v", err)
		}
	}

	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	inicio := -1
	for i, line := range lines {
		if inicio < 0 && strings.HasPrefix(line, "@@") {
			inicio = i
		}
		// O git pode acrescentar o nome da função ao cabeçalho; o
		// charm-diff não faz isso, então o sufixo é descartado.
		if strings.HasPrefix(line, "@@") {
			lines[i] = hunkHeaderOnly(line)
		}
	}
	if inicio < 0 {
		return nil
	}
	return trimEmpty(lines[inicio:])
}

// charmUnifiedLines devolve as linhas do diff do charm-diff sem o prefixo de
// números de linha, no mesmo formato usado pelo git.
func charmUnifiedLines(out string, numWidth int) []string {
	if out == "" {
		return nil
	}
	prefix := lineNumPrefixWidth(numWidth)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = ansi.Strip(line)
		if strings.HasPrefix(line, "@@") {
			result = append(result, hunkHeaderOnly(line))
			continue
		}
		if len(line) < prefix {
			result = append(result, line)
			continue
		}
		result = append(result, line[prefix:])
	}
	return result
}

// charmUnifiedLinesFor calcula a largura dos números de linha e converte a
// saída do charm-diff para o formato de linhas do git.
func charmUnifiedLinesFor(old, new string, opts Options) []string {
	numWidth := lineNumWidthFor(len(splitLines(old)), len(splitLines(new)))
	return charmUnifiedLines(UnifiedDiffWithOptions(old, new, opts), numWidth)
}

// hunkHeaderOnly devolve apenas o cabeçalho do hunk ("@@ -a,b +c,d @@"),
// descartando o contexto de função que o git pode acrescentar.
func hunkHeaderOnly(header string) string {
	rest := strings.TrimPrefix(header, "@@")
	idx := strings.Index(rest, "@@")
	if idx < 0 {
		return header
	}
	return "@@" + rest[:idx+2]
}

func trimEmpty(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// TestUnifiedDiffIgualAoGit compara a saída do charm-diff com a saída real do
// "git diff" para vários casos, verificando cabeçalhos de hunk, seleção de
// linhas e ordem.
func TestUnifiedDiffIgualAoGit(t *testing.T) {
	casos := []struct {
		nome    string
		old     string
		new     string
		context int
	}{
		{
			nome:    "inserção em arquivo vazio",
			old:     "",
			new:     "a\nb\n",
			context: 3,
		},
		{
			nome:    "inserção no início",
			old:     "a\nb\n",
			new:     "x\na\nb\n",
			context: 3,
		},
		{
			nome:    "inserção no meio",
			old:     "a\nb\nc\n",
			new:     "a\nx\nb\nc\n",
			context: 3,
		},
		{
			nome:    "inserção no fim",
			old:     "a\nb\nc\n",
			new:     "a\nb\nc\nd\n",
			context: 3,
		},
		{
			nome:    "remoção de tudo",
			old:     "a\nb\n",
			new:     "",
			context: 3,
		},
		{
			nome:    "remoção no início",
			old:     "a\nb\nc\n",
			new:     "b\nc\n",
			context: 3,
		},
		{
			nome:    "remoção no fim",
			old:     "a\nb\nc\n",
			new:     "a\nb\n",
			context: 3,
		},
		{
			nome:    "linha modificada",
			old:     "a\nb\nc\n",
			new:     "a\nX\nc\n",
			context: 3,
		},
		{
			nome:    "bloco modificado",
			old:     "a\nb\nc\nd\ne\n",
			new:     "a\nX\nY\nd\ne\n",
			context: 3,
		},
		{
			nome:    "sem contexto",
			old:     "a\nb\nc\n",
			new:     "a\nx\nb\nc\n",
			context: 0,
		},
		{
			nome:    "sem contexto, remoção",
			old:     "a\nb\nc\nd\n",
			new:     "a\nc\nd\n",
			context: 0,
		},
		{
			nome:    "sem contexto, inserção no fim",
			old:     "a\nb\nc\n",
			new:     "a\nb\nc\nd\n",
			context: 0,
		},
		{
			nome:    "hunks separados",
			old:     "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nl15\nl16\nl17\nl18\nl19\nl20\n",
			new:     "l1\nX\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nY\nl15\nl16\nl17\nl18\nl19\nl20\n",
			context: 3,
		},
		{
			nome:    "hunks com um espaçamento de contexto",
			old:     "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nl15\nl16\nl17\nl18\nl19\nl20\n",
			new:     "l1\nX\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nY\nl15\nl16\nl17\nl18\nl19\nl20\n",
			context: 1,
		},
		{
			nome:    "caso da auditoria",
			old:     "o00\no01\no02\no03\n",
			new:     "n00\nn01\no02\nn03\n",
			context: 3,
		},
		{
			nome:    "arquivo pequeno com várias mudanças",
			old:     "um\ndois\ntres\nquatro\ncinco\nseis\nsete\n",
			new:     "um\nDOIS\ntres\nquatro\ncinco\nSEIS\nsete\n",
			context: 2,
		},
	}

	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			opts := Options{ContextLines: tc.context, Style: DefaultStyle()}
			got := charmUnifiedLinesFor(tc.old, tc.new, opts)
			want := gitUnifiedDiff(t, tc.old, tc.new, tc.context)

			if len(got) != len(want) {
				t.Fatalf("número de linhas diferente: got %d, want %d\ngot:\n%s\nwant:\n%s",
					len(got), len(want), strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("linha %d diferente:\n got: %q\nwant: %q\nsaída completa:\n%s",
						i, got[i], want[i], strings.Join(got, "\n"))
				}
			}
		})
	}
}

// TestUnifiedDiffIgualAoGitAleatorio compara a saída com o git para um
// conjunto determinístico de entradas aleatórias.
func TestUnifiedDiffIgualAoGitAleatorio(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não está disponível no PATH: comparação com o git ignorada")
	}

	rng := rand.New(rand.NewSource(42))
	alfabeto := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	for caso := 0; caso < 200; caso++ {
		tamanho := 1 + rng.Intn(40)
		old := make([]string, tamanho)
		for i := range old {
			old[i] = alfabeto[rng.Intn(len(alfabeto))]
		}

		novo := make([]string, len(old))
		copy(novo, old)

		// Aplica algumas mudanças.
		for m := 0; m < 1+rng.Intn(6); m++ {
			switch rng.Intn(3) {
			case 0: // remoção
				if len(novo) > 0 {
					pos := rng.Intn(len(novo))
					novo = append(novo[:pos], novo[pos+1:]...)
				}
			case 1: // inserção
				pos := rng.Intn(len(novo) + 1)
				novo = append(novo, "")
				copy(novo[pos+1:], novo[pos:])
				novo[pos] = alfabeto[rng.Intn(len(alfabeto))]
			case 2: // modificação
				if len(novo) > 0 {
					novo[rng.Intn(len(novo))] = alfabeto[rng.Intn(len(alfabeto))]
				}
			}
		}

		oldText := ""
		for _, l := range old {
			oldText += l + "\n"
		}
		newText := ""
		for _, l := range novo {
			newText += l + "\n"
		}

		ctx := rng.Intn(4)
		opts := Options{ContextLines: ctx, Style: DefaultStyle()}
		got := charmUnifiedLinesFor(oldText, newText, opts)
		want := gitUnifiedDiff(t, oldText, newText, ctx)

		if len(got) != len(want) {
			t.Fatalf("caso %d (ctx=%d): número de linhas diferente: got %d, want %d\nantigo:\n%s\nnovo:\n%s\ngot:\n%s\nwant:\n%s",
				caso, ctx, len(got), len(want), oldText, newText,
				strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("caso %d (ctx=%d): linha %d diferente:\n got: %q\nwant: %q\nantigo:\n%s\nnovo:\n%s",
					caso, ctx, i, got[i], want[i], oldText, newText)
			}
		}
	}
}

// TestUnifiedDiffCabecalhoHunkSoInsercao verifica o cabeçalho de hunk apenas
// de inserção, que deve usar a linha anterior à inserção (como o git).
func TestUnifiedDiffCabecalhoHunkSoInsercao(t *testing.T) {
	casos := []struct {
		nome string
		old  string
		new  string
		want string
	}{
		{"arquivo vazio", "", "a\nb\n", "@@ -0,0 +1,2 @@"},
		{"inserção no início", "a\n", "x\na\n", "@@ -1 +1,2 @@"},
		{"inserção no meio", "a\nb\nc\n", "a\nx\nb\nc\n", "@@ -1,3 +1,4 @@"},
		{"inserção no fim", "a\nb\nc\n", "a\nb\nc\nd\n", "@@ -1,3 +1,4 @@"},
	}

	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			opts := Options{ContextLines: 3, Style: DefaultStyle()}
			linhas := charmUnifiedLinesFor(tc.old, tc.new, opts)
			if len(linhas) == 0 {
				t.Fatal("diff vazio")
			}
			if linhas[0] != tc.want {
				t.Fatalf("cabeçalho incorreto:\n got: %q\nwant: %q", linhas[0], tc.want)
			}
		})
	}
}

// TestNumeroDeHunksIgualAoGit confere a quantidade de hunks com o git.
func TestNumeroDeHunksIgualAoGit(t *testing.T) {
	old := strings.Repeat("linha\n", 50)
	novo := strings.Replace(old, "linha\n", "outra\n", 1)
	novo = strings.Replace(novo, "linha\n", "outra\n", 1)

	opts := Options{ContextLines: 3, Style: DefaultStyle()}
	got := charmUnifiedLinesFor(old, novo, opts)

	hunks := 0
	for _, linha := range got {
		if strings.HasPrefix(linha, "@@") {
			hunks++
		}
	}
	want := 0
	for _, linha := range gitUnifiedDiff(t, old, novo, 3) {
		if strings.HasPrefix(linha, "@@") {
			want++
		}
	}

	if hunks != want {
		t.Fatalf("número de hunks diferente: got %d, want %d\n%s", hunks, want,
			strings.Join(got, "\n"))
	}
}

// TestLinhasDeContextoIgualAoGit confere a quantidade de linhas de contexto
// entre as mudanças com o git.
func TestLinhasDeContextoIgualAoGit(t *testing.T) {
	old := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\nl11\nl12\nl13\nl14\nl15\nl16\nl17\nl18\nl19\nl20\n"
	novo := strings.Replace(old, "l10\n", "X\n", 1)

	opts := Options{ContextLines: 3, Style: DefaultStyle()}
	got := charmUnifiedLinesFor(old, novo, opts)
	want := gitUnifiedDiff(t, old, novo, 3)

	contextoGit := 0
	for _, linha := range want {
		if strings.HasPrefix(linha, " ") {
			contextoGit++
		}
	}
	contextoCharm := 0
	for _, linha := range got {
		if strings.HasPrefix(linha, " ") {
			contextoCharm++
		}
	}

	if contextoCharm != contextoGit {
		t.Fatalf("quantidade de linhas de contexto diferente: got %d, want %d", contextoCharm, contextoGit)
	}
}

// TestGitDisponivel apenas garante que o ambiente de teste tem git (o teste
// golden depende dele).
func TestGitDisponivel(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não está disponível no PATH")
	}
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		t.Fatalf("git --version falhou: %v", err)
	}
	t.Logf("git: %s", strings.TrimSpace(string(out)))
	fmt.Fprintln(os.Stderr, "")
}
