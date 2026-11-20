package main

import (
	"image"
	"image/color"
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
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

type boardCanvas struct {
	selected      string
	origin        f32.Point
	size          image.Point
	scale, dpi    float32
	fitScale      float32
	refit, fitted bool
	drag          boardDrag
	shaper        *text.Shaper
	nodes         []BoardNode
	edges         []BoardEdge
	cards         []boardNodeLayout
	rects         []boardRect
	links         []boardEdgeLayout
	bounds        boardRect
	framedBounds  boardRect
	dragged       string
	dragPoint     f32.Point
	gridOps       op.Ops
	gridCall      op.CallOp
	gridSize      image.Point
	gridStep      float64
	gridDPI       float32
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
	scale := min(2.2*c.dpi, max(min(.12*c.dpi, c.fitScale), c.scale*float32(factor)))
	c.origin = anchor.Sub(anchor.Sub(c.origin).Mul(scale / c.scale))
	c.scale, c.fitted = scale, false
}

func (c *boardCanvas) frame(bounds boardRect, size image.Point, dpi float32) {
	center := f32.Pt(float32(size.X)/2, float32(size.Y)/2)
	c.fitScale = boardFitScale(bounds, center, dpi)
	resized := c.dpi != dpi || c.size != size
	if c.scale == 0 || c.refit || c.fitted && (resized || bounds != c.framedBounds) && !c.drag.active {
		c.scale = c.fitScale
		c.origin = center.Sub(bounds.center().Mul(c.scale))
		c.refit, c.fitted = false, true
		c.framedBounds = bounds
	} else if resized && !c.drag.active {
		previous := f32.Pt(float32(c.size.X)/2, float32(c.size.Y)/2)
		ratio := dpi / c.dpi
		c.origin = center.Sub(previous.Sub(c.origin).Mul(ratio))
		c.scale *= ratio
	}
	c.size, c.dpi = size, dpi
}

func boardFitScale(bounds boardRect, center f32.Point, dpi float32) float32 {
	margin := f32.Pt(min(28*dpi, center.X*.28), min(28*dpi, center.Y*.28))
	available, world := center.Sub(margin).Mul(2), bounds.size()
	return min(1.25*dpi, available.X/max(1, world.X), available.Y/max(1, world.Y))
}

func (c *boardCanvas) focusNode(id string) {
	i := boardCanvasIndex(c.nodes, id)
	if i < 0 || c.scale == 0 {
		return
	}
	center := f32.Pt(float32(c.size.X)/2, float32(c.size.Y)/2)
	node := c.rects[i]
	c.scale = boardFitScale(node, center, c.dpi)
	c.origin = center.Sub(node.center().Mul(c.scale))
	c.fitted, c.refit = false, false
}

func (c *boardCanvas) scene(b *Board) {
	nodesChanged := !slices.Equal(c.nodes, b.Nodes)
	dragged, point := "", f32.Point{}
	if c.drag.moved {
		dragged, point = c.drag.node, c.drag.current
	}
	if !nodesChanged && slices.Equal(c.edges, b.Edges) && c.dragged == dragged && c.dragPoint == point {
		return
	}
	if nodesChanged {
		c.nodes = append(c.nodes[:0], b.Nodes...)
		c.cards = measureBoardNodes(b.Nodes)
		c.rects = make([]boardRect, len(b.Nodes))
	}
	c.edges = append(c.edges[:0], b.Edges...)
	c.dragged, c.dragPoint = dragged, point
	for i, node := range b.Nodes {
		p := boardNodePosition(node)
		if dragged == node.ID {
			p = point
		}
		c.rects[i] = c.cards[i].rect(p)
	}
	if c.shaper == nil {
		c.shaper = newBoardShaper()
	}
	c.links, c.bounds = layoutBoardEdges(c.shaper, *b, c.rects)
}

func (c *boardCanvas) screen(p f32.Point) f32.Point { return c.origin.Add(p.Mul(c.scale)) }

func (c *boardCanvas) layout(g ctx, b *Board, editable bool, onMove func(BoardNode)) dim {
	size := g.Constraints.Max
	if size.X <= 2 || size.Y <= 2 {
		return dim{Size: size}
	}
	dpi := g.Metric.PxPerDp
	if dpi == 0 {
		dpi = 1
	}
	c.scene(b)
	c.frame(c.bounds, size, dpi)
	c.update(g, b.Nodes, editable, onMove)
	c.scene(b)
	if c.refit {
		c.frame(c.bounds, size, dpi)
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

	for i, edge := range b.Edges {
		c.edge(g, c.links[i], c.selected == edge.From || c.selected == edge.To)
	}
	for i, edge := range b.Edges {
		c.edgeLabel(g, c.links[i], c.selected == edge.From || c.selected == edge.To)
	}
	for i, node := range b.Nodes {
		if node.ID != c.selected {
			c.node(g, node, i, false, bounds)
		}
	}
	if i := boardCanvasIndex(b.Nodes, c.selected); i >= 0 {
		c.node(g, b.Nodes[i], i, true, bounds)
	}
	return dim{Size: size}
}

func boardCanvasIndex(nodes []BoardNode, id string) int {
	return slices.IndexFunc(nodes, func(n BoardNode) bool { return n.ID == id })
}

func (c *boardCanvas) update(g ctx, nodes []BoardNode, editable bool, onMove func(BoardNode)) {
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

func (c *boardCanvas) handlePointer(g ctx, e pointer.Event, nodes []BoardNode, editable bool, onMove func(BoardNode)) {
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
			c.fitted = false
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

func (c *boardCanvas) handleKey(g ctx, name key.Name, nodes []BoardNode, editable bool, onMove func(BoardNode)) {
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
		c.fitted = false
		if editable && i >= 0 {
			node := &nodes[i]
			moveCanvasNode(node, f32.Pt(float32(node.X), float32(node.Y)).Add(delta), onMove)
		} else {
			c.origin = c.origin.Sub(delta.Mul(c.dpi))
		}
	}
	g.Execute(op.InvalidateCmd{})
}

func moveCanvasNode(node *BoardNode, p f32.Point, onMove func(BoardNode)) {
	x := min(10000, max(-10000, float64(p.X)))
	y := min(10000, max(-10000, float64(p.Y)))
	if node.X == x && node.Y == y {
		return
	}
	before := *node
	node.X, node.Y = x, y
	if onMove != nil {
		onMove(before)
	}
}

func (c *boardCanvas) hit(nodes []BoardNode, screen f32.Point) string {
	p := screen.Sub(c.origin).Div(c.scale)
	hit := func(i int) bool {
		r := c.rects[i]
		d, size := p.Sub(r.center()), r.size()
		x, y := absBoardCoord(d.X)*2/size.X, absBoardCoord(d.Y)*2/size.Y
		switch nodes[i].Shape {
		case "ellipse":
			return x*x+y*y <= 1
		case "diamond":
			return x+y <= 1
		default:
			return max(x, y) <= 1
		}
	}
	if i := boardCanvasIndex(nodes, c.selected); i >= 0 && hit(i) {
		return c.selected
	}
	for i := len(nodes) - 1; i >= 0; i-- {
		if hit(i) {
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
	if c.gridSize != c.size || c.gridStep != step || c.gridDPI != c.dpi {
		c.gridSize, c.gridStep, c.gridDPI = c.size, step, c.dpi
		c.gridOps.Reset()
		record := op.Record(&c.gridOps)
		r := .8 * c.dpi
		var path clip.Path
		path.Begin(&c.gridOps)
		for y := -step; y < float64(c.size.Y)+step; y += step {
			for x := -step; x < float64(c.size.X)+step; x += step {
				p := f32.Pt(float32(x), float32(y))
				path.MoveTo(p.Add(f32.Pt(0, -r)))
				path.LineTo(p.Add(f32.Pt(r, 0)))
				path.LineTo(p.Add(f32.Pt(0, r)))
				path.LineTo(p.Add(f32.Pt(-r, 0)))
				path.Close()
			}
		}
		paint.FillShape(&c.gridOps, hex(0x46516b), clip.Outline{Path: path.End()}.Op())
		c.gridCall = record.Stop()
	}
	offset := op.Affine(f32.Affine2D{}.Offset(f32.Pt(float32(x0), float32(y0)))).Push(g.Ops)
	c.gridCall.Add(g.Ops)
	offset.Pop()
}

func boardEdgeTint(chosen bool) color.NRGBA {
	if chosen {
		return accent
	}
	return hex(0x7385ac)
}

func (c *boardCanvas) edge(g ctx, edge boardEdgeLayout, chosen bool) {
	if len(edge.points) < 2 {
		return
	}
	z := c.screen(edge.points[len(edge.points)-1])
	d := z.Sub(c.screen(edge.points[len(edge.points)-2]))
	length := normBoardPoint(d)
	if length == 0 {
		return
	}
	u := d.Div(length)
	perp := f32.Pt(-u.Y, u.X)
	head := min(length/2, max(5*c.dpi, 10*c.scale))
	tint := boardEdgeTint(chosen)
	var path clip.Path
	path.Begin(g.Ops)
	path.MoveTo(c.screen(edge.points[0]))
	for _, p := range edge.points[1 : len(edge.points)-1] {
		path.LineTo(c.screen(p))
	}
	path.LineTo(z.Sub(u.Mul(head / 2)))
	paint.FillShape(g.Ops, tint, clip.Stroke{Path: path.End(), Width: max(c.dpi, 1.8*c.scale)}.Op())
	path.Begin(g.Ops)
	base := z.Sub(u.Mul(head))
	path.MoveTo(z)
	path.LineTo(base.Add(perp.Mul(head * .46)))
	path.LineTo(base.Sub(perp.Mul(head * .46)))
	path.Close()
	paint.FillShape(g.Ops, tint, clip.Outline{Path: path.End()}.Op())
}

func (c *boardCanvas) edgeLabel(g ctx, edge boardEdgeLayout, chosen bool) {
	if edge.label.wrapped == "" {
		return
	}
	bounds := image.Rectangle{Min: c.screen(edge.labelBox.min).Round(), Max: c.screen(edge.labelBox.max).Round()}
	paint.FillShape(g.Ops, boardPaper, clip.UniformRRect(bounds, int(5*c.scale)).Op(g.Ops))
	c.drawText(g, edge.label, c.screen(edge.labelBox.min.Add(f32.Pt(boardLabelPadX, boardLabelPadY))), boardEdgeTint(chosen), text.Middle)
}

func (c *boardCanvas) node(g ctx, n BoardNode, i int, chosen bool, viewport image.Rectangle) {
	r := c.rects[i]
	origin, end := c.screen(r.min).Round(), c.screen(r.max).Round()
	bounds := image.Rectangle{Min: origin, Max: end}
	if !bounds.Overlaps(viewport) {
		return
	}
	local := image.Rectangle{Max: bounds.Size()}
	card := c.cards[i]
	if n.Shape == "text" {
		offset := op.Offset(origin).Push(g.Ops)
		defer offset.Pop()
		area := clip.Rect(local).Push(g.Ops)
		semantic.LabelOp(n.Label).Add(g.Ops)
		semantic.DescriptionOp(n.Detail).Add(g.Ops)
		semantic.SelectedOp(chosen).Add(g.Ops)
		if chosen {
			round(g, local, 3, selected)
		}
		c.drawText(g, card.exported, f32.Point{}, ink, text.Start)
		area.Pop()
		return
	}
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
	outer := boardNodeClip(g.Ops, local, radius, n.Shape).Push(g.Ops)
	defer outer.Pop()
	semantic.LabelOp(n.Label).Add(g.Ops)
	semantic.DescriptionOp(n.Detail).Add(g.Ops)
	semantic.SelectedOp(chosen).Add(g.Ops)
	paint.Fill(g.Ops, border)
	stroke := max(1, int(c.dpi))
	if chosen {
		stroke *= 2
	}
	inside := local.Inset(stroke)
	if inside.Empty() {
		return
	}
	inner := boardNodeClip(g.Ops, inside, max(2, radius-stroke), n.Shape).Push(g.Ops)
	defer inner.Pop()
	if n.Shape == "topic" {
		gradient(g, f32.Pt(0, 0), f32.Pt(float32(local.Max.X), float32(local.Max.Y)), hex(0x404b7b), fill)
	} else {
		paint.Fill(g.Ops, fill)
	}

	p := f32.Pt(boardNodePad, (card.height-float32(card.textHeight()))/2)
	align := text.Start
	if n.Shape == "ellipse" || n.Shape == "diamond" {
		p.X, align = (card.width-float32(card.title.width))/2, text.Middle
	}
	c.drawText(g, card.title, p.Mul(c.scale), ink, align)
	if card.detail.wrapped != "" {
		p.Y += float32(card.title.height + boardTextGap)
		if align == text.Middle {
			p.X = (card.width - float32(card.detail.width)) / 2
		}
		c.drawText(g, card.detail, p.Mul(c.scale), tint, align)
	}
}

func (c *boardCanvas) drawText(g ctx, block boardTextBlock, position f32.Point, color color.NRGBA, alignment text.Alignment) {
	_, face := boardFonts()
	label := material.LabelStyle{
		Text: block.wrapped, Color: color, TextSize: unit.Sp(block.fontSize), Shaper: c.shaper,
		Font: font.Font{Typeface: face, Weight: block.weight}, Alignment: alignment,
		LineHeight: unit.Sp(block.lineHeight), LineHeightScale: 1, WrapPolicy: text.WrapWords,
	}
	g.Metric = unit.Metric{PxPerDp: 1, PxPerSp: 1}
	g.Constraints = layout.Constraints{Max: image.Pt(block.width+2, block.height)}
	if alignment == text.Middle {
		g.Constraints.Min.X = block.width
	}
	transform := op.Affine(f32.Affine2D{}.Scale(f32.Point{}, f32.Pt(c.scale, c.scale)).Offset(position)).Push(g.Ops)
	label.Layout(g)
	transform.Pop()
}
