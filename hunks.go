package charmdiff

import "fmt"

// Estrutura de um hunk no diff.
type hunk struct {
	// oldStart é o índice (base 0) da primeira linha antiga incluída no hunk.
	oldStart int
	// oldCount é a quantidade de linhas antigas incluídas no hunk.
	oldCount int
	// newStart é o índice (base 0) da primeira linha nova incluída no hunk.
	newStart int
	// newCount é a quantidade de linhas novas incluídas no hunk.
	newCount int
	// ops são as operações (contexto, remoções e inclusões) do hunk.
	ops []diffOp
}

// buildHunks agrupa as mudanças em hunks com linhas de contexto, usando as
// mesmas regras do "git diff":
//   - mudanças separadas por no máximo 2*contextLines linhas intactas são
//     unidas em um único hunk;
//   - o hunk recebe contextLines linhas de contexto antes e depois.
//
// O roteiro de edições (script) é a referência de posição: por isso a janela
// do hunk nunca é indexada pela posição da operação, o que eliminava os
// hunks sobrepostos quando a janela começava por uma inserção.
func buildHunks(res *xdiffResult, contextLines int) []hunk {
	if len(res.script) == 0 {
		return nil
	}
	// Defesa em profundidade: um contexto negativo nunca pode gerar uma
	// janela inválida.
	if contextLines < 0 {
		contextLines = 0
	}

	hunks := make([]hunk, 0, len(res.script))

	for first := 0; first < len(res.script); {
		last := xdiffHunkEnd(res.script, first, contextLines)

		xch := res.script[first]
		xche := res.script[last]

		s1 := xch.i1 - contextLines
		if s1 < 0 {
			s1 = 0
		}
		s2 := xch.i2 - contextLines
		if s2 < 0 {
			s2 = 0
		}

		// O contexto final é limitado pelo fim dos dois arquivos.
		lctx := contextLines
		if rest := res.old.nrec - (xche.i1 + xche.chg1); rest < lctx {
			lctx = rest
		}
		if rest := res.new.nrec - (xche.i2 + xche.chg2); rest < lctx {
			lctx = rest
		}

		e1 := xche.i1 + xche.chg1 + lctx
		e2 := xche.i2 + xche.chg2 + lctx

		// As operações do hunk: contexto inicial, mudanças e contexto final.
		start := res.changeOpIndex[first] - (xch.i2 - s2)
		end := res.changeOpIndex[last] + (xche.chg1 + xche.chg2) + (e2 - (xche.i2 + xche.chg2))

		hunks = append(hunks, hunk{
			oldStart: s1,
			oldCount: e1 - s1,
			newStart: s2,
			newCount: e2 - s2,
			ops:      res.ops[start:end],
		})

		first = last + 1
	}

	return hunks
}

// xdiffHunkEnd devolve o índice da última mudança que pertence ao hunk que
// começa na mudança first.
func xdiffHunkEnd(script []xdiffChange, first, contextLines int) int {
	maxCommon := 2 * contextLines
	lxch := first

	for xchp := first; xchp+1 < len(script); xchp++ {
		xch := xchp + 1
		distance := script[xch].i1 - (script[xchp].i1 + script[xchp].chg1)
		if distance > maxCommon {
			break
		}

		switch {
		case distance < contextLines:
			lxch = xch
		case lxch != xchp && script[xch].i1-(script[lxch].i1+script[lxch].chg1) > maxCommon:
			return lxch
		default:
			lxch = xch
		}
	}

	return lxch
}

// formatHunkHeader monta o cabeçalho do hunk no formato do git:
// "@@ -a,b +c,d @@".
func formatHunkHeader(h hunk) string {
	return fmt.Sprintf("@@ -%s +%s @@",
		formatHunkRange(h.oldStart, h.oldCount),
		formatHunkRange(h.newStart, h.newCount))
}

// formatHunkRange formata um intervalo de linhas no estilo do git.
// O git omite ",1" quando o intervalo tem uma única linha e, para
// intervalos vazios, usa a linha anterior ao intervalo.
func formatHunkRange(start, count int) string {
	switch count {
	case 0:
		return fmt.Sprintf("%d,0", start)
	case 1:
		return fmt.Sprintf("%d", start+1)
	default:
		return fmt.Sprintf("%d,%d", start+1, count)
	}
}
