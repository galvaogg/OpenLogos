package main

import (
	"image"
	"math"
	"testing"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
)

func TestBoardCanvasInputThroughGio(t *testing.T) {
	board := exampleBoard()
	s := &studio{theme: studioTheme()}
	var c boardCanvas
	var ops op.Ops
	var router input.Router
	editable, saves := true, 0
	size := image.Pt(730, 420)
	render := func() {
		ops.Reset()
		g := ctx{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(size)}
		c.layout(g, s, &board, editable, func(BoardNode) { saves++ })
		router.Frame(&ops)
	}
	mouse := func(kind pointer.Kind, buttons pointer.Buttons, p f32.Point) {
		router.Queue(pointer.Event{Kind: kind, Buttons: buttons, Source: pointer.Mouse, Position: p})
		render()
	}
	drag := func(from, to f32.Point) {
		mouse(pointer.Press, pointer.ButtonPrimary, from)
		mouse(pointer.Move, pointer.ButtonPrimary, to)
		mouse(pointer.Release, 0, to)
	}
	render()
	render()
	first := &board.Nodes[0]
	center := func(n *BoardNode) f32.Point {
		return c.origin.Add(f32.Pt(float32(n.X)+boardNodeWidth/2, float32(n.Y)+boardNodeHeight/2).Mul(c.scale))
	}
	from := center(first)
	to := from.Add(f32.Pt(28, 22))
	beforeX, beforeY := first.X, first.Y
	mouse(pointer.Press, pointer.ButtonPrimary, from)
	size.Y -= 38 // Selecting a node can reveal its controls below the canvas.
	mouse(pointer.Move, pointer.ButtonPrimary, to)
	if first.X != beforeX || first.Y != beforeY || saves != 0 || c.selected != first.ID {
		t.Fatal("drag must select and preview the node without saving")
	}
	mouse(pointer.Release, 0, to)
	if saves != 1 || math.Abs(first.X-beforeX-float64(28/c.scale)) > .01 ||
		math.Abs(first.Y-beforeY-float64(22/c.scale)) > .01 {
		t.Fatalf("drag must commit once at the released coordinates: %v, %v, saves %d", first.X, first.Y, saves)
	}
	origin := c.origin
	drag(f32.Pt(12, 12), f32.Pt(36, 20))
	if saves != 1 || c.selected != "" || c.origin.Sub(origin) != f32.Pt(24, 8) {
		t.Fatal("dragging empty space must pan without editing the board")
	}
	router.Queue(key.Event{Name: key.NameReturn, State: key.Press})
	render()
	if c.selected != first.ID {
		t.Fatalf("keyboard selection selected %q", c.selected)
	}

	editable = false
	render()
	from = center(first)
	drag(from, from.Add(f32.Pt(16, 16)))
	if saves != 1 {
		t.Fatal("dragging a proposed revision must preserve its coordinates")
	}
	anchor := f32.Pt(288, 200)
	world := anchor.Sub(c.origin).Div(c.scale)
	scale := c.scale
	router.Queue(pointer.Event{Kind: pointer.Scroll, Scroll: f32.Pt(0, -120), Position: anchor, Source: pointer.Mouse})
	render()
	if c.scale <= scale || anchor.Sub(c.origin).Div(c.scale).Sub(world).Round() != (image.Point{}) {
		t.Fatal("scroll zoom must preserve the world point beneath the pointer")
	}
}

func TestBoardCanvasConnectorEndsAtNodeBoundary(t *testing.T) {
	start, end, ok := canvasConnector(f32.Pt(0, 0), f32.Pt(321, 0), 1)
	if !ok || start != f32.Pt(boardNodeWidth, boardNodeHeight/2) || end != f32.Pt(321, boardNodeHeight/2) {
		t.Fatalf("connector endpoints: %v to %v, %v", start, end, ok)
	}
	if _, _, ok := canvasConnector(f32.Pt(100, 100), f32.Pt(120, 160), 1); ok {
		t.Fatal("overlapping nodes should not show a reversed connector")
	}
}
