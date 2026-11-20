package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"golang.org/x/image/math/fixed"
)

const (
	boardNodeWidth  = 256
	boardMinHeight  = 94
	boardNodePad    = 16
	boardTextGap    = 5
	boardColumnStep = boardNodeWidth + 126
	boardRowGap     = 56
	boardLabelPadX  = 5
	boardLabelPadY  = 3
)

type boardRect struct{ min, max f32.Point }

func (r boardRect) size() f32.Point   { return r.max.Sub(r.min) }
func (r boardRect) center() f32.Point { return r.min.Add(r.max).Mul(.5) }

func (r boardRect) outset(pad float32) boardRect {
	p := f32.Pt(pad, pad)
	return boardRect{r.min.Sub(p), r.max.Add(p)}
}

func (r boardRect) overlaps(other boardRect) bool {
	return r.min.X < other.max.X && other.min.X < r.max.X && r.min.Y < other.max.Y && other.min.Y < r.max.Y
}

func (r boardRect) union(other boardRect) boardRect {
	return boardRect{
		f32.Pt(min(r.min.X, other.min.X), min(r.min.Y, other.min.Y)),
		f32.Pt(max(r.max.X, other.max.X), max(r.max.Y, other.max.Y)),
	}
}

var boardFonts = sync.OnceValues(func() ([]font.FontFace, font.Typeface) {
	faces := gofont.Collection()
	var face font.Typeface
	if dir := os.Getenv("WINDIR"); filepath.IsAbs(dir) {
		for _, name := range []string{"arial.ttf", "arialbd.ttf"} {
			if data, err := os.ReadFile(filepath.Join(dir, "Fonts", name)); err == nil {
				if arial, err := opentype.ParseCollection(data); err == nil {
					faces, face = append(faces, arial...), arial[0].Font.Typeface
				}
			}
		}
	}
	return faces, face
})

func newBoardShaper() *text.Shaper {
	faces, _ := boardFonts()
	return text.NewShaper(text.NoSystemFonts(), text.WithCollection(faces))
}

type boardTextBlock struct {
	wrapped              string
	width, height        int
	fontSize, lineHeight int
	weight               font.Weight
}

func measureBoardText(shaper *text.Shaper, content string, width, size, leading int, weight font.Weight) boardTextBlock {
	block := boardTextBlock{fontSize: size, lineHeight: leading, weight: weight}
	if content == "" {
		return block
	}
	_, face := boardFonts()
	shaper.LayoutString(text.Parameters{
		Font: font.Font{Typeface: face, Weight: weight}, PxPerEm: fixed.I(size), MaxWidth: width,
		LineHeight: fixed.I(leading), LineHeightScale: 1, WrapPolicy: text.WrapHeuristically,
	}, content)
	runes := []rune(content)
	var lines []string
	var end, start, bottom int
	for glyph, ok := shaper.NextGlyph(); ok; glyph, ok = shaper.NextGlyph() {
		block.width = max(block.width, (glyph.X + max(glyph.Advance, glyph.Offset.X+glyph.Bounds.Max.X)).Ceil())
		bottom = max(bottom, int(glyph.Y)+max(glyph.Descent, glyph.Bounds.Max.Y).Ceil())
		end += int(glyph.Runes)
		if glyph.Flags&text.FlagLineBreak != 0 {
			lines = append(lines, strings.TrimRightFunc(string(runes[start:end]), unicode.IsSpace))
			start = end
		}
	}
	block.wrapped = strings.Join(lines, "\n")
	block.height = max(bottom, len(lines)*leading)
	return block
}

type boardNodeLayout struct {
	title, detail, exported boardTextBlock
	width, height           float32
}

func (n boardNodeLayout) rect(p f32.Point) boardRect {
	return boardRect{p, p.Add(f32.Pt(n.width, n.height))}
}

func (n boardNodeLayout) textHeight() int {
	if n.detail.height == 0 {
		return n.title.height
	}
	return n.title.height + boardTextGap + n.detail.height
}

func boardNodePosition(node BoardNode) f32.Point { return f32.Pt(float32(node.X), float32(node.Y)) }

func boardNodeText(node BoardNode) string {
	if node.Detail == "" {
		return node.Label
	}
	return node.Label + "\n" + node.Detail
}

var boardMeasure struct {
	sync.Mutex
	shaper *text.Shaper
}

func measureBoardNodes(nodes []BoardNode) []boardNodeLayout {
	boardMeasure.Lock()
	defer boardMeasure.Unlock()
	if boardMeasure.shaper == nil {
		boardMeasure.shaper = newBoardShaper()
	}
	shaper := boardMeasure.shaper
	layouts := make([]boardNodeLayout, len(nodes))
	for i, node := range nodes {
		n := &layouts[i]
		kind := node.excalidrawType()
		width := boardNodeWidth - 2*boardNodePad - 4
		switch kind {
		case "ellipse":
			width = 154
		case "diamond":
			width = 116
		}
		n.title = measureBoardText(shaper, node.Label, width, 18, 23, font.Bold)
		n.detail = measureBoardText(shaper, node.Detail, width, 15, 20, font.Normal)
		n.exported = measureBoardText(shaper, boardNodeText(node), width, 16, 20, font.Normal)
		contentHeight := max(n.textHeight(), n.exported.height)
		n.width, n.height = boardNodeWidth, float32(max(boardMinHeight, contentHeight+2*boardNodePad))
		switch kind {
		case "ellipse":
			n.height = max(n.height, float32((contentHeight+12)*3/2))
		case "diamond":
			n.height = max(n.height, float32((contentHeight+12)*2))
		case "text":
			n.width, n.height = float32(max(1, n.exported.width)), float32(max(1, n.exported.height))
		}
	}
	return layouts
}

// arrangeBoard only places new IDs in a revision. Their obstacles use the revised text sizes.
func arrangeBoard(b *Board, previous *Board) {
	if len(b.Nodes) == 0 {
		return
	}
	layouts := measureBoardNodes(b.Nodes)
	columns := boardColumns(*b)
	type column struct{ band, x int }
	slots := make([]column, len(b.Nodes))
	heights := make(map[column]float32)
	bandHeights := make([]float32, 1)
	for i, node := range b.Nodes {
		col := column{x: columns[node.ID]}
		if b.Kind == "timeline" {
			col = column{i / 4, i % 4}
		} else if b.Kind == "flowchart" {
			col = column{col.x / 4, col.x % 4}
		}
		if col.band%2 == 1 {
			col.x = 3 - col.x
		}
		slots[i] = col
		heights[col] += layouts[i].height + boardRowGap
		for len(bandHeights) <= col.band {
			bandHeights = append(bandHeights, 0)
		}
		bandHeights[col.band] = max(bandHeights[col.band], heights[col]-boardRowGap)
	}
	bandOffsets := make([]float32, len(bandHeights))
	for i := 1; i < len(bandOffsets); i++ {
		bandOffsets[i] = bandOffsets[i-1] + bandHeights[i-1] + boardRowGap
	}
	positions := make(map[string]BoardNode)
	if previous != nil {
		for _, n := range previous.Nodes {
			positions[n.ID] = n
		}
	}
	var placed []boardRect
	for i, n := range b.Nodes {
		if old, ok := positions[n.ID]; ok {
			b.Nodes[i].X, b.Nodes[i].Y = old.X, old.Y
			placed = append(placed, layouts[i].rect(boardNodePosition(old)))
		}
	}
	used := make(map[column]float32)
	for i := range b.Nodes {
		n := &b.Nodes[i]
		col := slots[i]
		center := float32(0)
		if b.Kind != "mindmap" {
			center = bandOffsets[col.band] + bandHeights[col.band]/2
		}
		p := f32.Pt(float32(col.x*boardColumnStep), center-(heights[col]-boardRowGap)/2+used[col])
		used[col] += layouts[i].height + boardRowGap
		if _, kept := positions[n.ID]; kept {
			continue
		}
		start := p.Y
		direction := float32(1)
		for {
			rect := layouts[i].rect(p)
			next := p.Y
			for _, obstacle := range placed {
				if obstacle.outset(boardRowGap / 2).overlaps(rect) {
					if direction > 0 {
						next = max(next, obstacle.max.Y+boardRowGap)
					} else {
						next = min(next, obstacle.min.Y-layouts[i].height-boardRowGap)
					}
				}
			}
			if next == p.Y {
				placed = append(placed, rect)
				n.X, n.Y = float64(p.X), float64(p.Y)
				break
			}
			p.Y = next
			if p.Y > 10000 {
				p.Y, direction = start, -1
			}
		}
	}
}

type boardEdgeLayout struct {
	points     []f32.Point
	start, end [2]float64
	label      boardTextBlock
	labelBox   boardRect
	labelBound bool
}

func layoutBoardEdges(shaper *text.Shaper, b Board, rects []boardRect) ([]boardEdgeLayout, boardRect) {
	var bounds boardRect
	if len(rects) > 0 {
		bounds = rects[0]
		for _, r := range rects[1:] {
			bounds = bounds.union(r)
		}
	}
	index := make(map[string]int, len(b.Nodes))
	for i, node := range b.Nodes {
		index[node.ID] = i
	}
	edges := make([]boardEdgeLayout, len(b.Edges))
	occupied := append([]boardRect(nil), rects...)
	for i, edge := range b.Edges {
		from, hasFrom := index[edge.From]
		to, hasTo := index[edge.To]
		if !hasFrom || !hasTo || from == to {
			continue
		}
		e := &edges[i]
		e.points, e.start, e.end = routeBoardEdge(rects, from, to)
		last := len(e.points) - 1
		if shape := b.Nodes[from].Shape; shape == "ellipse" || shape == "diamond" {
			e.points[0], e.start = boardShapePort(rects[from], e.points[1], shape)
		}
		if shape := b.Nodes[to].Shape; shape == "ellipse" || shape == "diamond" {
			e.points[last], e.end = boardShapePort(rects[to], e.points[last-1], shape)
		}
		for _, p := range e.points {
			bounds = bounds.union(boardRect{p, p}.outset(2))
		}
		e.label = measureBoardText(shaper, edge.Label, 118, 13, 16, font.Normal)
		if e.label.wrapped != "" {
			e.labelBox, e.labelBound = positionBoardEdgeLabel(e.points, e.label, occupied)
			occupied = append(occupied, e.labelBox)
			bounds = bounds.union(e.labelBox)
		}
	}
	return edges, bounds
}

func normBoardPoint(p f32.Point) float32 { return float32(math.Hypot(float64(p.X), float64(p.Y))) }

func boardPort(r boardRect, toward f32.Point) (f32.Point, [2]float64) {
	return boardShapePort(r, toward, "rectangle")
}

func boardShapePort(r boardRect, toward f32.Point, shape string) (f32.Point, [2]float64) {
	d := toward.Sub(r.center())
	size := r.size()
	x, y := d.X*2/size.X, d.Y*2/size.Y
	f := max(absBoardCoord(x), absBoardCoord(y))
	switch shape {
	case "ellipse":
		f = normBoardPoint(f32.Pt(x, y))
	case "diamond":
		f = absBoardCoord(x) + absBoardCoord(y)
	}
	if f == 0 {
		return f32.Pt(r.max.X+5, r.center().Y), [2]float64{1, .5}
	}
	offset := d.Div(f)
	return r.center().Add(offset).Add(d.Mul(5 / normBoardPoint(d))),
		[2]float64{float64(min(1, max(0, .5+offset.X/size.X))), float64(min(1, max(0, .5+offset.Y/size.Y)))}
}

func absBoardCoord(x float32) float32 { return max(x, -x) }

func routeBoardEdge(rects []boardRect, from, to int) ([]f32.Point, [2]float64, [2]float64) {
	a, z := rects[from], rects[to]
	ac, zc := a.center(), z.center()
	clear := func(points ...f32.Point) bool {
		for i, r := range rects {
			if i == from || i == to {
				continue
			}
			for j, p := range points[1:] {
				if r.outset(7).intersects(points[j], p) {
					return false
				}
			}
		}
		return true
	}
	if !a.outset(7).overlaps(z.outset(7)) {
		start, bindingA := boardPort(a, zc)
		end, bindingZ := boardPort(z, ac)
		if clear(start, end) {
			return []f32.Point{start, end}, bindingA, bindingZ
		}
	}

	var best []f32.Point
	var start, end [2]float64
	bestLength := float32(math.Inf(1))
	try := func(p, q f32.Point) {
		s, sa := boardPort(a, p)
		e, ea := boardPort(z, q)
		length := normBoardPoint(s.Sub(p)) + normBoardPoint(p.Sub(q)) + normBoardPoint(q.Sub(e))
		if length < bestLength && clear(s, p, q, e) {
			best, start, end, bestLength = []f32.Point{s, p, q, e}, sa, ea, length
		}
	}
	for _, r := range rects {
		for _, x := range []float32{r.min.X - 16, r.max.X + 16} {
			if (x <= a.min.X-8 || x >= a.max.X+8) && (x <= z.min.X-8 || x >= z.max.X+8) && absBoardCoord(ac.Y-zc.Y) >= 8 {
				try(f32.Pt(x, ac.Y), f32.Pt(x, zc.Y))
			}
		}
		for _, y := range []float32{r.min.Y - 16, r.max.Y + 16} {
			if (y <= a.min.Y-8 || y >= a.max.Y+8) && (y <= z.min.Y-8 || y >= z.max.Y+8) && absBoardCoord(ac.X-zc.X) >= 8 {
				try(f32.Pt(ac.X, y), f32.Pt(zc.X, y))
			}
		}
	}
	if best != nil {
		return best, start, end
	}
	if !a.overlaps(z) && ac != zc {
		s, sa := boardPort(a, zc)
		e, ea := boardPort(z, ac)
		if e.Sub(s).X*(zc.X-ac.X)+e.Sub(s).Y*(zc.Y-ac.Y) > 0 {
			return []f32.Point{s, e}, sa, ea
		}
	}
	// shortcut: a fully enclosed port can cross a manually overlapping node, use a multi-bend router if editing needs it.
	x := max(a.max.X, z.max.X) + 24
	y1, y2 := a.min.Y+a.size().Y*.3, z.min.Y+z.size().Y*.7
	if absBoardCoord(y1-y2) < 8 {
		y2 = z.min.Y + z.size().Y*.9
	}
	return []f32.Point{f32.Pt(a.max.X+5, y1), f32.Pt(x, y1), f32.Pt(x, y2), f32.Pt(z.max.X+5, y2)},
		[2]float64{1, .3}, [2]float64{1, float64((y2 - z.min.Y) / z.size().Y)}
}

func (r boardRect) intersects(a, z f32.Point) bool {
	near, far := float32(0), float32(1)
	for _, axis := range [][4]float32{{a.X, z.X - a.X, r.min.X, r.max.X}, {a.Y, z.Y - a.Y, r.min.Y, r.max.Y}} {
		origin, delta, low, high := axis[0], axis[1], axis[2], axis[3]
		if delta == 0 {
			if origin < low || origin > high {
				return false
			}
			continue
		}
		p, q := (low-origin)/delta, (high-origin)/delta
		near, far = max(near, min(p, q)), min(far, max(p, q))
	}
	return near <= far
}

func positionBoardEdgeLabel(points []f32.Point, label boardTextBlock, occupied []boardRect) (boardRect, bool) {
	half := f32.Pt(float32(label.width)/2+boardLabelPadX, float32(label.height)/2+boardLabelPadY)
	var length float32
	for i, p := range points[1:] {
		length += normBoardPoint(p.Sub(points[i]))
	}
	for _, offset := range []float32{0, 1, -1, 2, -2, 3, -3} {
		for _, fraction := range []float32{.5, .33, .67, .2, .8} {
			distance := length * fraction
			for i, p := range points[1:] {
				d := p.Sub(points[i])
				segment := normBoardPoint(d)
				if segment == 0 || distance > segment {
					distance -= segment
					continue
				}
				u := d.Div(segment)
				n := f32.Pt(-u.Y, u.X)
				spacing := absBoardCoord(n.X)*half.X + absBoardCoord(n.Y)*half.Y + 5
				center := points[i].Add(u.Mul(distance)).Add(n.Mul(offset * spacing))
				box := boardRect{center.Sub(half), center.Add(half)}
				clear := true
				for _, r := range occupied {
					if r.outset(4).overlaps(box) {
						clear = false
						break
					}
				}
				if clear {
					return box, len(points) == 2 && offset == 0 && fraction == .5
				}
				break
			}
		}
	}
	center := points[0].Add(points[len(points)-1]).Mul(.5)
	for _, r := range occupied {
		if r.min.X < center.X+half.X && r.max.X > center.X-half.X {
			center.Y = min(center.Y, r.min.Y-half.Y-5)
		}
	}
	return boardRect{center.Sub(half), center.Add(half)}, false
}
