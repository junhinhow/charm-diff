package charmdiff

import (
	"strings"
	"testing"
)

func TestUnifiedDiff_NoChanges(t *testing.T) {
	text := "line1\nline2\nline3\n"
	result := UnifiedDiff(text, text)
	if result != "" {
		t.Errorf("esperava diff vazio para textos iguais, obteve: %q", result)
	}
}

func TestUnifiedDiff_Addition(t *testing.T) {
	old := "line1\nline3\n"
	new := "line1\nline2\nline3\n"
	result := UnifiedDiff(old, new)
	if !strings.Contains(result, "+line2") {
		t.Errorf("esperava '+line2' no diff, obteve: %q", result)
	}
}

func TestUnifiedDiff_Deletion(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nline3\n"
	result := UnifiedDiff(old, new)
	if !strings.Contains(result, "-line2") {
		t.Errorf("esperava '-line2' no diff, obteve: %q", result)
	}
}

func TestSideBySideDiff_Basic(t *testing.T) {
	old := "hello\nworld\n"
	new := "hello\nearth\n"
	result := SideBySideDiff(old, new, 80)
	if result == "" {
		t.Error("esperava resultado nao vazio para textos diferentes")
	}
	if !strings.Contains(result, "world") || !strings.Contains(result, "earth") {
		t.Errorf("esperava 'world' e 'earth' no resultado, obteve: %q", result)
	}
}

func TestSideBySideDiff_NoChanges(t *testing.T) {
	text := "same\n"
	result := SideBySideDiff(text, text, 80)
	// Deve conter o texto, mas sem marcacoes de add/delete
	if !strings.Contains(result, "same") {
		t.Errorf("esperava 'same' no resultado, obteve: %q", result)
	}
}

func TestComputeDiff_Empty(t *testing.T) {
	ops := computeDiff(nil, nil)
	if len(ops) != 0 {
		t.Errorf("esperava 0 operacoes para entradas vazias, obteve: %d", len(ops))
	}
}

func TestSplitLines(t *testing.T) {
	lines := splitLines("a\nb\nc\n")
	if len(lines) != 3 {
		t.Errorf("esperava 3 linhas, obteve: %d", len(lines))
	}
	if lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Errorf("conteudo inesperado: %v", lines)
	}
}

func TestPadOrTruncate(t *testing.T) {
	result := padOrTruncate("hello", 10)
	if len(result) != 10 {
		t.Errorf("esperava comprimento 10, obteve: %d", len(result))
	}

	result = padOrTruncate("hello world this is long", 10)
	if len(result) != 10 {
		t.Errorf("esperava comprimento 10, obteve: %d (%q)", len(result), result)
	}
}
