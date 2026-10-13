package charmdiff

// Motor de cálculo de diff compatível com o git.
//
// A implementação reproduz o algoritmo do xdiff (o mesmo usado pelo git):
//
//  1. as linhas dos dois arquivos são classificadas (conteúdos iguais recebem
//     o mesmo identificador);
//  2. linhas sem correspondência no outro arquivo são descartadas
//     antecipadamente (otimização do xdiff que também influi no resultado);
//  3. o restante é comparado com o algoritmo de Myers (1986) em espaço
//     linear, por divisão e conquista sobre a "middle snake";
//  4. os blocos de mudança são deslizados ("compactados") para produzir o
//     mesmo alinhamento do git;
//  5. o roteiro de edições (script) é montado e os hunks são gerados com o
//     mesmo critério de contexto do git.
//
// O resultado é byte a byte igual ao de "git diff" no que diz respeito aos
// cabeçalhos de hunk, a seleção de linhas e a ordem das linhas.

const (
	// xdMaxEqLimit limita o número de repetições de uma linha aceito como
	// "candidata" a casar com o outro arquivo.
	xdMaxEqLimit = 1024
	// xdSimScanWindow é a janela usada para avaliar linhas repetidas.
	xdSimScanWindow = 100
	// xdKpdisRun é o peso usado na decisão de descartar linhas repetidas.
	xdKpdisRun = 4
	// xdSnakeCnt é o tamanho mínimo de um "snake" (sequência de linhas iguais).
	xdSnakeCnt = 20
	// xdKHeur é o peso da heurística de caminho "interessante".
	xdKHeur = 4
	// xdMaxCostMin é o custo máximo mínimo de busca antes do corte.
	xdMaxCostMin = 256
	// xdHeurMinCost é o custo a partir do qual a heurística é avaliada.
	xdHeurMinCost = 256
)

// ações do classificador de linhas.
const (
	xdDiscard uint8 = iota
	xdKeep
	xdInvestigate
)

// xdiffFile guarda as linhas de um arquivo, os identificadores do seu
// conteúdo e as marcas das linhas alteradas.
type xdiffFile struct {
	lines   []string
	ids     []int
	changed []bool
	// referenceIndex mapeia os índices de referência (usados pelo algoritmo)
	// para os índices reais das linhas.
	referenceIndex []int
	nrec           int
	nreff          int
	dstart         int
	dend           int
}

// newXdiffFile cria a estrutura de um arquivo a partir das suas linhas.
func newXdiffFile(lines []string) *xdiffFile {
	return &xdiffFile{
		lines:   lines,
		ids:     make([]int, len(lines)),
		changed: make([]bool, len(lines)+2),
		nrec:    len(lines),
	}
}

// changedAt informa se a linha de índice i foi alterada. As posições -1 e
// nrec são tratadas como linhas intactas (sentinelas).
func (f *xdiffFile) changedAt(i int) bool {
	if i < 0 || i >= f.nrec {
		return false
	}
	return f.changed[i+1]
}

// setChanged marca a linha de índice i como alterada (ou intacta).
func (f *xdiffFile) setChanged(i int, v bool) {
	f.changed[i+1] = v
}

// xdiffClassifier atribui um identificador único a cada conteúdo de linha e
// conta as ocorrências em cada arquivo.
type xdiffClassifier struct {
	ids  map[string]int
	len1 []int
	len2 []int
}

func newXdiffClassifier() *xdiffClassifier {
	return &xdiffClassifier{ids: make(map[string]int)}
}

// classify atribui os identificadores das linhas de um arquivo.
func (c *xdiffClassifier) classify(f *xdiffFile, pass int) {
	for i, line := range f.lines {
		id, ok := c.ids[line]
		if !ok {
			id = len(c.ids)
			c.ids[line] = id
			c.len1 = append(c.len1, 0)
			c.len2 = append(c.len2, 0)
		}
		if pass == 1 {
			c.len1[id]++
		} else {
			c.len2[id]++
		}
		f.ids[i] = id
	}
}

// xdiffChange é um bloco de mudanças do roteiro de edições: chg1 linhas do
// arquivo antigo e chg2 linhas do arquivo novo foram trocadas.
type xdiffChange struct {
	i1   int // início do bloco no arquivo antigo
	i2   int // início do bloco no arquivo novo
	chg1 int // quantidade de linhas alteradas no arquivo antigo
	chg2 int // quantidade de linhas alteradas no arquivo novo
}

// xdiffResult é o resultado completo do cálculo de diff.
type xdiffResult struct {
	old    *xdiffFile
	new    *xdiffFile
	ops    []diffOp
	script []xdiffChange
	// changeOpIndex[k] é o índice da primeira operação do bloco k.
	changeOpIndex []int
}

// xdiffSplit é o ponto de divisão encontrado na busca de Myers.
type xdiffSplit struct {
	i1    int
	i2    int
	minLo int
	minHi int
}

// xdiffEnv guarda os parâmetros de custo da busca.
type xdiffEnv struct {
	mxcost   int
	snakeCnt int
	heurMin  int
	// offset é o deslocamento usado para indexar os vetores de diagonais.
	offset int
}

// computeDiff calcula a sequência de operações que transforma oldLines em
// newLines, com o mesmo alinhamento do git.
func computeDiff(oldLines, newLines []string) []diffOp {
	return computeDiffFull(oldLines, newLines).ops
}

// computeDiffFull calcula o diff completo (operações e roteiro de edições).
func computeDiffFull(oldLines, newLines []string) *xdiffResult {
	res := &xdiffResult{
		old: newXdiffFile(oldLines),
		new: newXdiffFile(newLines),
	}

	if res.old.nrec == 0 && res.new.nrec == 0 {
		return res
	}

	// Classificação dos conteúdos.
	class := newXdiffClassifier()
	class.classify(res.old, 1)
	class.classify(res.new, 2)

	// Preparação do ambiente: corte das pontas comuns e limpeza dos
	// registros que não têm correspondência.
	xdiffTrimEnds(res.old, res.new)
	xdiffCleanupRecords(class, res.old, res.new)

	// Algoritmo de Myers em espaço linear.
	ndiags := res.old.nreff + res.new.nreff + 3
	size := res.old.nreff + res.new.nreff + 5
	kvdf := make([]int, size)
	kvdb := make([]int, size)
	for i := range kvdf {
		kvdf[i] = -1
		kvdb[i] = xdLineMax
	}

	env := &xdiffEnv{
		mxcost:   xdBogoSqrt(ndiags),
		snakeCnt: xdSnakeCnt,
		heurMin:  xdHeurMinCost,
		offset:   res.new.nreff + 2,
	}
	if env.mxcost < xdMaxCostMin {
		env.mxcost = xdMaxCostMin
	}

	xdiffRecsCmp(res.old, 0, res.old.nreff, res.new, 0, res.new.nreff,
		kvdf, kvdb, 0, env)

	// Compactação dos blocos de mudança (o mesmo ajuste do git).
	xdiffChangeCompact(res.old, res.new)
	xdiffChangeCompact(res.new, res.old)

	// Roteiro de edições.
	res.script = xdiffBuildScript(res.old, res.new)
	res.ops, res.changeOpIndex = xdiffBuildOps(res.old, res.new, res.script)

	return res
}

// xdLineMax é o maior valor representável, usado como sentinela.
const xdLineMax = int(^uint(0) >> 1)

// xdBogoSqrt estima a raiz quadrada de n (mesma heurística do xdiff).
func xdBogoSqrt(n int) int {
	i := 1
	for ; n > 0; n >>= 2 {
		i <<= 1
	}
	return i
}

// xdiffTrimEnds localiza o prefixo e o sufixo comuns aos dois arquivos.
func xdiffTrimEnds(f1, f2 *xdiffFile) {
	lim := f1.nrec
	if f2.nrec < lim {
		lim = f2.nrec
	}

	i := 0
	for i < lim && f1.ids[i] == f2.ids[i] {
		i++
	}
	f1.dstart = i
	f2.dstart = i

	lim -= i
	j := 0
	for j < lim && f1.ids[f1.nrec-1-j] == f2.ids[f2.nrec-1-j] {
		j++
	}
	f1.dend = f1.nrec - j - 1
	f2.dend = f2.nrec - j - 1
}

// xdiffCleanupRecords descarta as linhas que não têm correspondência no outro
// arquivo (elas são sempre alteradas) e mantém as demais como candidatas.
func xdiffCleanupRecords(c *xdiffClassifier, f1, f2 *xdiffFile) {
	off := f1.dstart
	len1 := f1.dend - off + 1
	len2 := f2.dend - off + 1
	if len1 < 0 {
		len1 = 0
	}
	if len2 < 0 {
		len2 = 0
	}

	mlim1 := xdBogoSqrt(f1.nrec)
	if mlim1 > xdMaxEqLimit {
		mlim1 = xdMaxEqLimit
	}
	mlim2 := xdBogoSqrt(f2.nrec)
	if mlim2 > xdMaxEqLimit {
		mlim2 = xdMaxEqLimit
	}

	action1 := make([]uint8, len1)
	action2 := make([]uint8, len2)

	for i := 0; i < len1; i++ {
		nm := c.len2[f1.ids[off+i]]
		switch {
		case nm == 0:
			action1[i] = xdDiscard
		case nm < mlim1:
			action1[i] = xdKeep
		default:
			action1[i] = xdInvestigate
		}
	}
	for i := 0; i < len2; i++ {
		nm := c.len1[f2.ids[off+i]]
		switch {
		case nm == 0:
			action2[i] = xdDiscard
		case nm < mlim2:
			action2[i] = xdKeep
		default:
			action2[i] = xdInvestigate
		}
	}

	f1.referenceIndex = make([]int, 0, len1)
	f2.referenceIndex = make([]int, 0, len2)

	for i := 0; i < len1; i++ {
		action := action1[i]
		if action == xdInvestigate {
			if !xdiffCleanMmatch(action1, i, len1) {
				action = xdKeep
			} else {
				action = xdDiscard
			}
		}
		if action == xdKeep {
			f1.referenceIndex = append(f1.referenceIndex, off+i)
			f1.nreff++
		} else {
			f1.setChanged(off+i, true)
		}
	}

	for i := 0; i < len2; i++ {
		action := action2[i]
		if action == xdInvestigate {
			if !xdiffCleanMmatch(action2, i, len2) {
				action = xdKeep
			} else {
				action = xdDiscard
			}
		}
		if action == xdKeep {
			f2.referenceIndex = append(f2.referenceIndex, off+i)
			f2.nreff++
		} else {
			f2.setChanged(off+i, true)
		}
	}
}

// xdiffCleanMmatch decide se uma linha que aparece muitas vezes no outro
// arquivo deve ser descartada. Retorna true quando a linha deve ser
// descartada.
func xdiffCleanMmatch(action []uint8, i, length int) bool {
	s := 0
	e := length - 1
	if i-s > xdSimScanWindow {
		s = i - xdSimScanWindow
	}
	if e-i > xdSimScanWindow {
		e = i + xdSimScanWindow
	}

	rDis0, rpDis0 := 0, 1
scanAntes:
	for r := 1; i-r >= s; r++ {
		switch action[i-r] {
		case xdDiscard:
			rDis0++
		case xdInvestigate:
			rpDis0++
		default:
			break scanAntes
		}
	}
	if rDis0 == 0 {
		return false
	}

	rDis1, rpDis1 := 0, 1
scanDepois:
	for r := 1; i+r <= e; r++ {
		switch action[i+r] {
		case xdDiscard:
			rDis1++
		case xdInvestigate:
			rpDis1++
		default:
			break scanDepois
		}
	}
	if rDis1 == 0 {
		return false
	}

	rDis1 += rDis0
	rpDis1 += rpDis0

	return rpDis1*xdKpdisRun < (rpDis1 + rDis1)
}

// xdiffRecsCmp compara os trechos equivalentes de dois arquivos, marcando as
// linhas alteradas (divide et impera).
func xdiffRecsCmp(f1 *xdiffFile, off1, lim1 int, f2 *xdiffFile, off2, lim2 int,
	kvdf, kvdb []int, needMin int, env *xdiffEnv) {

	// Encolhe a caixa andando pelas diagonais iguais.
	for off1 < lim1 && off2 < lim2 &&
		f1.ids[f1.referenceIndex[off1]] == f2.ids[f2.referenceIndex[off2]] {
		off1++
		off2++
	}
	for off1 < lim1 && off2 < lim2 &&
		f1.ids[f1.referenceIndex[lim1-1]] == f2.ids[f2.referenceIndex[lim2-1]] {
		lim1--
		lim2--
	}

	switch {
	case off1 == lim1:
		for ; off2 < lim2; off2++ {
			f2.setChanged(f2.referenceIndex[off2], true)
		}
	case off2 == lim2:
		for ; off1 < lim1; off1++ {
			f1.setChanged(f1.referenceIndex[off1], true)
		}
	default:
		var spl xdiffSplit
		xdiffSplitSearch(f1, off1, lim1, f2, off2, lim2, kvdf, kvdb, needMin, &spl, env)
		xdiffRecsCmp(f1, off1, spl.i1, f2, off2, spl.i2, kvdf, kvdb, spl.minLo, env)
		xdiffRecsCmp(f1, spl.i1, lim1, f2, spl.i2, lim2, kvdf, kvdb, spl.minHi, env)
	}
}

// xdiffSplitSearch procura o ponto de divisão (middle snake) entre os trechos
// informados, usando as diagonais do Myers.
func xdiffSplitSearch(f1 *xdiffFile, off1, lim1 int, f2 *xdiffFile, off2, lim2 int,
	kvdf, kvdb []int, needMin int, spl *xdiffSplit, env *xdiffEnv) int {

	offset := env.offset

	dmin := off1 - lim2
	dmax := lim1 - off2
	fmid := off1 - off2
	bmid := lim1 - lim2
	odd := (fmid - bmid) & 1
	fmin, fmax := fmid, fmid
	bmin, bmax := bmid, bmid

	kvdf[fmid+offset] = off1
	kvdb[bmid+offset] = lim1

	for ec := 1; ; ec++ {
		gotSnake := false

		// Alarga o domínio das diagonais do caminho progressivo.
		if fmin > dmin {
			fmin--
			kvdf[fmin-1+offset] = -1
		} else {
			fmin++
		}
		if fmax < dmax {
			fmax++
			kvdf[fmax+1+offset] = -1
		} else {
			fmax--
		}

		for d := fmax; d >= fmin; d -= 2 {
			var i1 int
			if kvdf[d-1+offset] >= kvdf[d+1+offset] {
				i1 = kvdf[d-1+offset] + 1
			} else {
				i1 = kvdf[d+1+offset]
			}
			prev1 := i1
			i2 := i1 - d
			for i1 < lim1 && i2 < lim2 &&
				f1.ids[f1.referenceIndex[i1]] == f2.ids[f2.referenceIndex[i2]] {
				i1++
				i2++
			}
			if i1-prev1 > env.snakeCnt {
				gotSnake = true
			}
			kvdf[d+offset] = i1
			if odd != 0 && bmin <= d && d <= bmax && kvdb[d+offset] <= i1 {
				spl.i1 = i1
				spl.i2 = i2
				spl.minLo = 1
				spl.minHi = 1
				return ec
			}
		}

		// Alarga o domínio das diagonais do caminho reverso.
		if bmin > dmin {
			bmin--
			kvdb[bmin-1+offset] = xdLineMax
		} else {
			bmin++
		}
		if bmax < dmax {
			bmax++
			kvdb[bmax+1+offset] = xdLineMax
		} else {
			bmax--
		}

		for d := bmax; d >= bmin; d -= 2 {
			var i1 int
			if kvdb[d-1+offset] < kvdb[d+1+offset] {
				i1 = kvdb[d-1+offset]
			} else {
				i1 = kvdb[d+1+offset] - 1
			}
			prev1 := i1
			i2 := i1 - d
			for i1 > off1 && i2 > off2 &&
				f1.ids[f1.referenceIndex[i1-1]] == f2.ids[f2.referenceIndex[i2-1]] {
				i1--
				i2--
			}
			if prev1-i1 > env.snakeCnt {
				gotSnake = true
			}
			kvdb[d+offset] = i1
			if odd == 0 && fmin <= d && d <= fmax && i1 <= kvdf[d+offset] {
				spl.i1 = i1
				spl.i2 = i2
				spl.minLo = 1
				spl.minHi = 1
				return ec
			}
		}

		if needMin != 0 {
			continue
		}

		// Heurística: procura um caminho "interessante" quando o custo da
		// busca já é alto (mesma regra do xdiff).
		if gotSnake && ec > env.heurMin {
			best := 0
			for d := fmax; d >= fmin; d -= 2 {
				dd := d - fmid
				if dd < 0 {
					dd = -dd
				}
				i1 := kvdf[d+offset]
				i2 := i1 - d
				v := (i1 - off1) + (i2 - off2) - dd

				if v > xdKHeur*ec && v > best &&
					off1+env.snakeCnt <= i1 && i1 < lim1 &&
					off2+env.snakeCnt <= i2 && i2 < lim2 {
					for k := 1; k <= env.snakeCnt; k++ {
						if f1.ids[f1.referenceIndex[i1-k]] != f2.ids[f2.referenceIndex[i2-k]] {
							break
						}
						if k == env.snakeCnt {
							best = v
							spl.i1 = i1
							spl.i2 = i2
							break
						}
					}
				}
			}
			if best > 0 {
				spl.minLo = 1
				spl.minHi = 0
				return ec
			}

			best = 0
			for d := bmax; d >= bmin; d -= 2 {
				dd := d - bmid
				if dd < 0 {
					dd = -dd
				}
				i1 := kvdb[d+offset]
				i2 := i1 - d
				v := (lim1 - i1) + (lim2 - i2) - dd

				if v > xdKHeur*ec && v > best &&
					off1 < i1 && i1 <= lim1-env.snakeCnt &&
					off2 < i2 && i2 <= lim2-env.snakeCnt {
					for k := 0; k < env.snakeCnt; k++ {
						if f1.ids[f1.referenceIndex[i1+k]] != f2.ids[f2.referenceIndex[i2+k]] {
							break
						}
						if k == env.snakeCnt-1 {
							best = v
							spl.i1 = i1
							spl.i2 = i2
							break
						}
					}
				}
			}
			if best > 0 {
				spl.minLo = 0
				spl.minHi = 1
				return ec
			}
		}

		// Corte: a busca ficou cara demais, usa o caminho mais longe
		// alcançado (mesma regra do xdiff).
		if ec >= env.mxcost {
			fbest, fbest1 := -1, -1
			for d := fmax; d >= fmin; d -= 2 {
				i1 := kvdf[d+offset]
				if i1 > lim1 {
					i1 = lim1
				}
				i2 := i1 - d
				if lim2 < i2 {
					i1 = lim2 + d
					i2 = lim2
				}
				if fbest < i1+i2 {
					fbest = i1 + i2
					fbest1 = i1
				}
			}

			bbest, bbest1 := xdLineMax, xdLineMax
			for d := bmax; d >= bmin; d -= 2 {
				i1 := kvdb[d+offset]
				if i1 < off1 {
					i1 = off1
				}
				i2 := i1 - d
				if i2 < off2 {
					i1 = off2 + d
					i2 = off2
				}
				if i1+i2 < bbest {
					bbest = i1 + i2
					bbest1 = i1
				}
			}

			if (lim1+lim2)-bbest < fbest-(off1+off2) {
				spl.i1 = fbest1
				spl.i2 = fbest - fbest1
				spl.minLo = 1
				spl.minHi = 0
			} else {
				spl.i1 = bbest1
				spl.i2 = bbest - bbest1
				spl.minLo = 0
				spl.minHi = 1
			}
			return ec
		}
	}
}

// xdlGroup é um grupo de linhas alteradas consecutivas de um arquivo.
type xdlGroup struct {
	start int
	end   int
}

// xdiffGroupInit posiciona o grupo no início do arquivo.
func xdiffGroupInit(f *xdiffFile) xdlGroup {
	g := xdlGroup{0, 0}
	for f.changedAt(g.end) {
		g.end++
	}
	return g
}

// xdiffGroupNext avança para o próximo grupo.
func xdiffGroupNext(f *xdiffFile, g *xdlGroup) bool {
	if g.end == f.nrec {
		return false
	}
	g.start = g.end + 1
	for g.end = g.start; f.changedAt(g.end); g.end++ {
	}
	return true
}

// xdiffGroupPrevious recua para o grupo anterior.
func xdiffGroupPrevious(f *xdiffFile, g *xdlGroup) bool {
	if g.start == 0 {
		return false
	}
	g.end = g.start - 1
	for g.start = g.end; f.changedAt(g.start - 1); g.start-- {
	}
	return true
}

// xdiffGroupSlideDown desloca o grupo para baixo, quando possível.
func xdiffGroupSlideDown(f *xdiffFile, g *xdlGroup) bool {
	if g.end < f.nrec && f.lines[g.start] == f.lines[g.end] {
		f.setChanged(g.start, false)
		g.start++
		f.setChanged(g.end, true)
		g.end++
		for f.changedAt(g.end) {
			g.end++
		}
		return true
	}
	return false
}

// xdiffGroupSlideUp desloca o grupo para cima, quando possível.
func xdiffGroupSlideUp(f *xdiffFile, g *xdlGroup) bool {
	if g.start > 0 && f.lines[g.start-1] == f.lines[g.end-1] {
		g.start--
		f.setChanged(g.start, true)
		g.end--
		f.setChanged(g.end, false)
		for f.changedAt(g.start - 1) {
			g.start--
		}
		return true
	}
	return false
}

// xdiffChangeCompact desloca os blocos de mudança para produzir um diff mais
// "natural", igual ao git.
func xdiffChangeCompact(f *xdiffFile, fo *xdiffFile) {
	g := xdiffGroupInit(f)
	goGroup := xdiffGroupInit(fo)

	for {
		if g.end != g.start {
			groupSize := 0
			earliestEnd := 0
			endMatchingOther := -1

			for {
				groupSize = g.end - g.start
				endMatchingOther = -1

				// Desliza o grupo para cima o máximo possível.
				for xdiffGroupSlideUp(f, &g) {
					if !xdiffGroupPrevious(fo, &goGroup) {
						break
					}
				}

				earliestEnd = g.end
				if goGroup.end > goGroup.start {
					endMatchingOther = g.end
				}

				// Desliza o grupo para baixo o máximo possível.
				for {
					if !xdiffGroupSlideDown(f, &g) {
						break
					}
					if !xdiffGroupNext(fo, &goGroup) {
						break
					}
					if goGroup.end > goGroup.start {
						endMatchingOther = g.end
					}
				}

				if groupSize == g.end-g.start {
					break
				}
			}

			switch {
			case g.end == earliestEnd:
				// Não foi possível deslizar o grupo.
			case endMatchingOther != -1:
				// Alinha o grupo com a última mudança do outro arquivo.
				for goGroup.end == goGroup.start {
					if !xdiffGroupSlideUp(f, &g) {
						break
					}
					if !xdiffGroupPrevious(fo, &goGroup) {
						break
					}
				}
			}
		}

		if !xdiffGroupNext(f, &g) {
			break
		}
		if !xdiffGroupNext(fo, &goGroup) {
			break
		}
	}
}

// xdiffBuildScript monta o roteiro de edições a partir das marcas de linhas
// alteradas dos dois arquivos.
func xdiffBuildScript(f1, f2 *xdiffFile) []xdiffChange {
	var script []xdiffChange

	i1, i2 := f1.nrec, f2.nrec
	for i1 >= 0 || i2 >= 0 {
		if f1.changedAt(i1-1) || f2.changedAt(i2-1) {
			l1 := i1
			for f1.changedAt(i1 - 1) {
				i1--
			}
			l2 := i2
			for f2.changedAt(i2 - 1) {
				i2--
			}
			script = append(script, xdiffChange{
				i1:   i1,
				i2:   i2,
				chg1: l1 - i1,
				chg2: l2 - i2,
			})
		}
		i1--
		i2--
	}

	// A lista foi montada de trás para frente.
	for l, r := 0, len(script)-1; l < r; l, r = l+1, r-1 {
		script[l], script[r] = script[r], script[l]
	}

	return script
}

// xdiffBuildOps converte o roteiro de edições na lista de operações
// (contexto, remoções e inclusões).
func xdiffBuildOps(f1, f2 *xdiffFile, script []xdiffChange) ([]diffOp, []int) {
	ops := make([]diffOp, 0, f1.nrec+f2.nrec)
	changeOpIndex := make([]int, 0, len(script))

	i1, i2 := 0, 0
	for _, ch := range script {
		// Linhas de contexto entre as mudanças.
		for i1 < ch.i1 && i2 < ch.i2 {
			ops = append(ops, diffOp{
				Kind:    opEqual,
				OldLine: i1,
				NewLine: i2,
				Text:    f2.lines[i2],
			})
			i1++
			i2++
		}

		changeOpIndex = append(changeOpIndex, len(ops))

		for k := ch.i1; k < ch.i1+ch.chg1; k++ {
			ops = append(ops, diffOp{
				Kind:    opDelete,
				OldLine: k,
				NewLine: -1,
				Text:    f1.lines[k],
			})
		}
		for k := ch.i2; k < ch.i2+ch.chg2; k++ {
			ops = append(ops, diffOp{
				Kind:    opInsert,
				OldLine: -1,
				NewLine: k,
				Text:    f2.lines[k],
			})
		}

		i1 = ch.i1 + ch.chg1
		i2 = ch.i2 + ch.chg2
	}

	// Linhas de contexto finais.
	for i1 < f1.nrec && i2 < f2.nrec {
		ops = append(ops, diffOp{
			Kind:    opEqual,
			OldLine: i1,
			NewLine: i2,
			Text:    f2.lines[i2],
		})
		i1++
		i2++
	}

	return ops, changeOpIndex
}
