package main

import (
	"image"
	"math"
	"slices"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type boardCanvas struct {
	selected   string
	origin     f32.Point
	size       image.Point
	scale, dpi float32
	refit      bool
	drag       boardDrag
}

type boardDrag struct {
	active, moved           bool
	pointer                 pointer.ID
	node                    string
	start, initial, current f32.Point
}

var boardPaper = hex(0x14171d)

func (c *boardCanvas) fit() {
	c.refit = true
	c.drag = boardDrag{}
}

func (c *boardCanvas) selectNode(id string) {
	c.selected = id
}

func (c *boardCanvas) zoomBy(factor float64) {
	c.zoomAt(f32.Pt(float32(c.size.X)/2, float32(c.size.Y)/2), factor)
}

func (c *boardCanvas) zoomAt(anchor f32.Point, factor float64) {
	if c.scale == 0 {
		return
	}
	scale := min(2.2*c.dpi, max(.16*c.dpi, c.scale*float32(factor)))
	c.origin = anchor.Sub(anchor.Sub(c.origin).Mul(scale / c.scale))
	c.scale = scale
}

func (c *boardCanvas) frame(nodes []BoardNode, size image.Point, dpi float32) {
	center := f32.Pt(float32(size.X)/2, float32(size.Y)/2)
	if c.scale == 0 || c.refit {
		c.scale, c.origin = dpi, center
		if len(nodes) > 0 {
			minX, minY := nodes[0].X, nodes[0].Y
			maxX, maxY := minX+boardNodeWidth, minY+boardNodeHeight
			for _, n := range nodes[1:] {
				minX, minY = min(minX, n.X), min(minY, n.Y)
				maxX, maxY = max(maxX, n.X+boardNodeWidth), max(maxY, n.Y+boardNodeHeight)
			}
			available := center.Mul(2).Sub(f32.Pt(56*dpi, 56*dpi))
			c.scale = max(.16*dpi, min(dpi, available.X/float32(maxX-minX), available.Y/float32(maxY-minY)))
			c.origin = center.Sub(f32.Pt(float32(minX+maxX)/2, float32(minY+maxY)/2).Mul(c.scale))
		}
		c.refit = false
	} else if c.dpi != dpi || c.size != size && !c.drag.active {
		previous := f32.Pt(float32(c.size.X)/2, float32(c.size.Y)/2)
		ratio := dpi / c.dpi
		c.origin = center.Sub(previous.Sub(c.origin).Mul(ratio))
		c.scale *= ratio
		c.drag = boardDrag{}
	}
	c.size, c.dpi = size, dpi
}

func (c *boardCanvas) layout(g ctx, s *studio, b *Board, editable bool, onMove func()) dim {
	size := g.Constraints.Max
	if size.X <= 2 || size.Y <= 2 {
		return dim{Size: size}
	}
	dpi := g.Metric.PxPerDp
	if dpi == 0 {
		dpi = 1
	}
	c.frame(b.Nodes, size, dpi)
	c.update(g, b.Nodes, editable, onMove)
	if c.refit {
		c.frame(b.Nodes, size, dpi)
	}

	bounds := image.Rectangle{Max: size}
	edgeColor := line
	if g.Source.Focused(c) {
		edgeColor = hex(0x586793)
	}
	paint.FillShape(g.Ops, edgeColor, clip.Rect(bounds).Op())
	inside := clip.Rect(bounds.Inset(1)).Push(g.Ops)
	defer inside.Pop()

	event.Op(g.Ops, c)
	semantic.LabelOp("Área do quadro. Enter seleciona um elemento. Setas movem a seleção ou o quadro.").Add(g.Ops)
	if c.drag.active {
		pointer.CursorGrabbing.Add(g.Ops)
	} else {
		pointer.CursorGrab.Add(g.Ops)
	}
	paint.Fill(g.Ops, boardPaper)
	c.grid(g)

	positions := make(map[string]f32.Point, len(b.Nodes))
	occupied := make([]image.Rectangle, 0, len(b.Nodes)+len(b.Edges))
	for _, node := range b.Nodes {
		p := f32.Pt(float32(node.X), float32(node.Y))
		if c.drag.node == node.ID && c.drag.moved {
			p = c.drag.current
		}
		position := c.origin.Add(p.Mul(c.scale))
		positions[node.ID] = position
		occupied = append(occupied, image.Rectangle{Min: position.Round(),
			Max: position.Add(f32.Pt(boardNodeWidth*c.scale, boardNodeHeight*c.scale)).Round()})
	}
	for _, edge := range b.Edges {
		a, from := positions[edge.From]
		z, to := positions[edge.To]
		if from && to {
			c.edge(g, s, a, z, edge, &occupied)
		}
	}
	for _, node := range b.Nodes {
		if node.ID != c.selected {
			c.node(g, s, node, positions[node.ID], false, bounds)
		}
	}
	if i := boardCanvasIndex(b.Nodes, c.selected); i >= 0 {
		node := b.Nodes[i]
		c.node(g, s, node, positions[node.ID], true, bounds)
	}
	return dim{Size: size}
}

func boardCanvasIndex(nodes []BoardNode, id string) int {
	return slices.IndexFunc(nodes, func(n BoardNode) bool { return n.ID == id })
}

func (c *boardCanvas) update(g ctx, nodes []BoardNode, editable bool, onMove func()) {
	if boardCanvasIndex(nodes, c.selected) < 0 {
		c.selected = ""
	}
	if !editable && c.drag.node != "" {
		c.drag = boardDrag{}
	}
	scroll := pointer.ScrollRange{Min: -4096, Max: 4096}
	filters := []event.Filter{
		pointer.Filter{Target: c, Kinds: pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel | pointer.Scroll, ScrollX: scroll, ScrollY: scroll},
		key.FocusFilter{Target: c},
	}
	for _, name := range []key.Name{key.NameReturn, key.NameEnter, key.NameEscape, key.NameHome, key.NameLeftArrow, key.NameRightArrow, key.NameUpArrow, key.NameDownArrow} {
		filters = append(filters, key.Filter{Focus: c, Name: name})
	}
	for {
		ev, ok := g.Event(filters...)
		if !ok {
			return
		}
		switch ev := ev.(type) {
		case pointer.Event:
			c.handlePointer(g, ev, nodes, editable, onMove)
		case key.Event:
			if ev.State == key.Press {
				c.handleKey(g, ev.Name, nodes, editable, onMove)
			}
		}
	}
}

func (c *boardCanvas) handlePointer(g ctx, e pointer.Event, nodes []BoardNode, editable bool, onMove func()) {
	switch e.Kind {
	case pointer.Scroll:
		if !c.drag.active {
			c.zoomAt(e.Position, math.Exp(-float64(e.Scroll.Y)/(400*float64(c.dpi))))
		}
	case pointer.Press:
		if c.drag.active || (e.Source == pointer.Mouse && !e.Buttons.Contain(pointer.ButtonPrimary) && !e.Buttons.Contain(pointer.ButtonTertiary)) {
			return
		}
		c.drag = boardDrag{active: true, pointer: e.PointerID, start: e.Position, initial: c.origin}
		if !e.Buttons.Contain(pointer.ButtonTertiary) {
			c.selected = c.hit(nodes, e.Position)
			if i := boardCanvasIndex(nodes, c.selected); editable && i >= 0 {
				c.drag.node = c.selected
				c.drag.initial = f32.Pt(float32(nodes[i].X), float32(nodes[i].Y))
			}
		}
		g.Execute(key.FocusCmd{Tag: c})
		g.Execute(op.InvalidateCmd{})
	case pointer.Drag, pointer.Release:
		if !c.drag.active || c.drag.pointer != e.PointerID {
			return
		}
		delta := e.Position.Sub(c.drag.start)
		c.drag.moved = c.drag.moved || delta.X*delta.X+delta.Y*delta.Y >= 9*c.dpi*c.dpi
		if c.drag.moved {
			if c.drag.node != "" {
				c.drag.current = c.drag.initial.Add(delta.Div(c.scale))
			} else {
				c.origin = c.drag.initial.Add(delta)
			}
		}
		if e.Kind == pointer.Release {
			if i := boardCanvasIndex(nodes, c.drag.node); i >= 0 && c.drag.moved {
				moveCanvasNode(&nodes[i], c.drag.current, onMove)
			}
			c.drag = boardDrag{}
		}
	case pointer.Cancel:
		if c.drag.pointer == e.PointerID {
			c.drag = boardDrag{}
		}
	}
}

func (c *boardCanvas) handleKey(g ctx, name key.Name, nodes []BoardNode, editable bool, onMove func()) {
	i := boardCanvasIndex(nodes, c.selected)
	var delta f32.Point
	switch name {
	case key.NameReturn, key.NameEnter:
		if len(nodes) > 0 {
			c.selected = nodes[(i+1)%len(nodes)].ID
		}
	case key.NameEscape:
		c.selected = ""
	case key.NameHome:
		c.fit()
	case key.NameLeftArrow:
		delta.X = -16
	case key.NameRightArrow:
		delta.X = 16
	case key.NameUpArrow:
		delta.Y = -16
	case key.NameDownArrow:
		delta.Y = 16
	}
	if delta != (f32.Point{}) {
		if editable && i >= 0 {
			node := &nodes[i]
			moveCanvasNode(node, f32.Pt(float32(node.X), float32(node.Y)).Add(delta), onMove)
		} else {
			c.origin = c.origin.Sub(delta.Mul(c.dpi))
		}
	}
	g.Execute(op.InvalidateCmd{})
}

func moveCanvasNode(node *BoardNode, p f32.Point, onMove func()) {
	x := min(10000, max(-10000, float64(p.X)))
	y := min(10000, max(-10000, float64(p.Y)))
	if node.X == x && node.Y == y {
		return
	}
	node.X, node.Y = x, y
	if onMove != nil {
		onMove()
	}
}

func (c *boardCanvas) hit(nodes []BoardNode, screen f32.Point) string {
	p := screen.Sub(c.origin).Div(c.scale)
	hit := func(n BoardNode) bool {
		x, y := float64(p.X)-n.X, float64(p.Y)-n.Y
		return x >= 0 && x <= boardNodeWidth && y >= 0 && y <= boardNodeHeight
	}
	if i := boardCanvasIndex(nodes, c.selected); i >= 0 && hit(nodes[i]) {
		return c.selected
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		if hit(nodes[i]) {
			return nodes[i].ID
		}
	}
	return ""
}

func (c *boardCanvas) grid(g ctx) {
	step := float64(c.scale) * 24
	for step < 19*float64(c.dpi) {
		step *= 2
	}
	x0 := math.Mod(float64(c.origin.X), step)
	y0 := math.Mod(float64(c.origin.Y), step)
	r := .8 * c.dpi
	var path clip.Path
	path.Begin(g.Ops)
	for y := y0; y < float64(c.size.Y); y += step {
		for x := x0; x < float64(c.size.X); x += step {
			p := f32.Pt(float32(x), float32(y))
			path.MoveTo(p.Add(f32.Pt(0, -r)))
			path.LineTo(p.Add(f32.Pt(r, 0)))
			path.LineTo(p.Add(f32.Pt(0, r)))
			path.LineTo(p.Add(f32.Pt(-r, 0)))
			path.Close()
		}
	}
	paint.FillShape(g.Ops, hex(0x46516b), clip.Outline{Path: path.End()}.Op())
}

func canvasConnector(a, z f32.Point, scale float32) (f32.Point, f32.Point, bool) {
	half := f32.Pt(boardNodeWidth*scale/2, boardNodeHeight*scale/2)
	a, z = a.Add(half), z.Add(half)
	delta := z.Sub(a)
	ratio := max(float32(math.Abs(float64(delta.X)))/half.X, float32(math.Abs(float64(delta.Y)))/half.Y)
	if ratio <= 2 {
		return f32.Point{}, f32.Point{}, false
	}
	edge := delta.Div(ratio)
	return a.Add(edge), z.Sub(edge), true
}

func (c *boardCanvas) edge(g ctx, s *studio, a, z f32.Point, edge BoardEdge, occupied *[]image.Rectangle) {
	a, z, ok := canvasConnector(a, z, c.scale)
	if !ok {
		return
	}
	d := z.Sub(a)
	length := float32(math.Hypot(float64(d.X), float64(d.Y)))
	u := d.Div(length)
	perp := f32.Pt(-u.Y, u.X)
	head := min(length/2, max(5*c.dpi, 10*c.scale))
	tint := hex(0x7385ac)
	if c.selected == edge.From || c.selected == edge.To {
		tint = accent
	}
	var path clip.Path
	path.Begin(g.Ops)
	path.MoveTo(a)
	path.LineTo(z.Sub(u.Mul(head / 2)))
	paint.FillShape(g.Ops, tint, clip.Stroke{Path: path.End(), Width: max(c.dpi, 1.8*c.scale)}.Op())
	path.Begin(g.Ops)
	base := z.Sub(u.Mul(head))
	path.MoveTo(z)
	path.LineTo(base.Add(perp.Mul(head * .46)))
	path.LineTo(base.Sub(perp.Mul(head * .46)))
	path.Close()
	paint.FillShape(g.Ops, tint, clip.Outline{Path: path.End()}.Op())
	if edge.Label == "" || c.scale/c.dpi < .42 {
		return
	}
	label := material.Label(s.theme, unit.Sp(max(9, 12*c.scale/c.dpi)), edge.Label)
	label.Color, label.Font.Weight, label.MaxLines = tint, font.Medium, 1
	lctx := g
	lctx.Constraints = layout.Constraints{Max: image.Pt(g.Dp(160), g.Dp(32))}
	record := op.Record(g.Ops)
	dimensions := box(lctx, boardPaper, clear, 5, padding(2, 5, 2, label.Layout))
	call := record.Stop()
	for _, position := range []float32{.5, .33, .67, .22, .78} {
		center := a.Add(d.Mul(position)).Round().Sub(dimensions.Size.Div(2))
		bounds := image.Rectangle{Min: center, Max: center.Add(dimensions.Size)}
		if slices.ContainsFunc(*occupied, func(other image.Rectangle) bool { return bounds.Overlaps(other) }) {
			continue
		}
		*occupied = append(*occupied, bounds.Inset(-1))
		offset := op.Offset(center).Push(g.Ops)
		call.Add(g.Ops)
		offset.Pop()
		break
	}
}

func (c *boardCanvas) node(g ctx, s *studio, n BoardNode, p f32.Point, chosen bool, viewport image.Rectangle) {
	origin := p.Round()
	end := p.Add(f32.Pt(boardNodeWidth*c.scale, boardNodeHeight*c.scale)).Round()
	bounds := image.Rectangle{Min: origin, Max: end}
	if !bounds.Overlaps(viewport) {
		return
	}
	local := image.Rectangle{Max: bounds.Size()}
	fill, border, tint := hex(0x242d3a), hex(0x495a75), hex(0xbccfec)
	switch n.Shape {
	case "topic":
		fill, border, tint = hex(0x303759), hex(0x6575ad), accent
	case "note":
		fill, border, tint = hex(0x23383a), hex(0x427173), hex(0xb2d8d1)
	case "decision":
		fill, border, tint = hex(0x392f28), hex(0x846748), hex(0xf2c88e)
	}
	if chosen {
		border = accent
	}
	offset := op.Offset(origin).Push(g.Ops)
	defer offset.Pop()
	radius := max(3, int(13*c.scale))
	outer := clip.UniformRRect(local, radius).Op(g.Ops).Push(g.Ops)
	defer outer.Pop()
	semantic.LabelOp(n.Label).Add(g.Ops)
	semantic.DescriptionOp(n.Detail).Add(g.Ops)
	semantic.SelectedOp(chosen).Add(g.Ops)
	paint.Fill(g.Ops, border)
	stroke := max(1, int(c.dpi))
	if chosen {
		stroke *= 2
	}
	inner := clip.UniformRRect(local.Inset(stroke), max(2, radius-stroke)).Op(g.Ops).Push(g.Ops)
	defer inner.Pop()
	if n.Shape == "topic" {
		gradient(g, f32.Pt(0, 0), f32.Pt(float32(local.Max.X), float32(local.Max.Y)), hex(0x404b7b), fill)
	} else {
		paint.Fill(g.Ops, fill)
	}

	zoom := c.scale / c.dpi
	fontSize := unit.Sp(max(8, 17*zoom))
	title := material.Label(s.theme, fontSize, n.Label)
	title.Color, title.Font.Weight, title.MaxLines = ink, font.SemiBold, 2
	detail := material.Label(s.theme, unit.Sp(max(8, 13*zoom)), n.Detail)
	detail.Color, detail.MaxLines = tint, 2
	g.Constraints = layout.Exact(local.Size())
	inset(unit.Dp(max(3, 14*zoom)), func(g ctx) dim {
		return layout.W.Layout(g, func(g ctx) dim {
			if n.Detail == "" || zoom < .42 {
				return title.Layout(g)
			}
			return column(g, title.Layout, gap(unit.Dp(max(1, 5*zoom))), detail.Layout)
		})
	})(g)
}
