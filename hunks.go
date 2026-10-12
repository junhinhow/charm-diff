package charmdiff

import "fmt"

// Estrutura de um hunk no diff.
type hunk struct {
	// oldStart e o indice (base 0) da primeira linha antiga incluida no hunk.
	oldStart int
	// oldCount e a quantidade de linhas antigas incluidas no hunk.
	oldCount int
	// newStart e o indice (base 0) da primeira linha nova incluida no hunk.
	newStart int
	// newCount e a quantidade de linhas novas incluidas no hunk.
	newCount int
	// ops sao as operacoes (contexto, remocoes e inclusoes) do hunk.
	ops []diffOp
}

// buildHunks agrupa as mudancas em hunks com linhas de contexto, usando as
// mesmas regras do "git diff":
//   - mudancas separadas por no maximo 2*contextLines linhas intactas sao
//     unidas em um unico hunk;
//   - o hunk recebe contextLines linhas de contexto antes e depois.
//
// O roteiro de edicoes (script) e a referencia de posicao: por isso a janela
// do hunk nunca e indexada pela posicao da operacao, o que eliminava os
// hunks sobrepostos quando a janela comecava por uma insercao.
func buildHunks(res *xdiffResult, contextLines int) []hunk {
	if len(res.script) == 0 {
		return nil
	}
	// Defense em profundidade: um contexto negativo nunca pode gerar uma
	// janela invalida.
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

		// O contexto final e limitado pelo fim dos dois arquivos.
		lctx := contextLines
		if rest := res.old.nrec - (xche.i1 + xche.chg1); rest < lctx {
			lctx = rest
		}
		if rest := res.new.nrec - (xche.i2 + xche.chg2); rest < lctx {
			lctx = rest
		}

		e1 := xche.i1 + xche.chg1 + lctx
		e2 := xche.i2 + xche.chg2 + lctx

		// As operacoes do hunk: contexto inicial, mudancas e contexto final.
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

// xdiffHunkEnd devolve o indice da ultima mudanca que pertence ao hunk que
// comeca na mudanca first.
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

// formatHunkHeader monta o cabecalho do hunk no formato do git:
// "@@ -a,b +c,d @@".
func formatHunkHeader(h hunk) string {
	return fmt.Sprintf("@@ -%s +%s @@",
		formatHunkRange(h.oldStart, h.oldCount),
		formatHunkRange(h.newStart, h.newCount))
}

// formatHunkRange formata um intervalo de linhas no estilo do git.
// O git omite ",1" quando o intervalo tem uma unica linha e, para
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
