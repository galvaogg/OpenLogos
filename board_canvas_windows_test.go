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
	var c boardCanvas
	var ops op.Ops
	var router input.Router
	editable, saves := true, 0
	size := image.Pt(730, 420)
	render := func() {
		ops.Reset()
		g := ctx{Ops: &ops, Source: router.Source(), Constraints: layout.Exact(size)}
		c.layout(g, &board, editable, func(BoardNode) { saves++ })
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
		return c.screen(c.rects[boardCanvasIndex(board.Nodes, n.ID)].center())
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

func TestBoardCanvasFitContainsDistantNodes(t *testing.T) {
	b := examBoard("flowchart")
	b.Nodes[0].X, b.Nodes[0].Y = -9900, -4500
	b.Nodes[1].X, b.Nodes[1].Y = 9900, 4500
	var c boardCanvas
	c.scene(&b)
	for _, size := range []image.Point{image.Pt(1500, 900), image.Pt(420, 260)} {
		c.frame(c.bounds, size, 1)
		for i, node := range b.Nodes {
			a, z := c.screen(c.rects[i].min), c.screen(c.rects[i].max)
			if a.X < 10 || a.Y < 10 || z.X > float32(size.X-10) || z.Y > float32(size.Y-10) {
				t.Errorf("%v: %s bounds %v→%v outside fit margin", size, node.ID, a, z)
			}
		}
		a, z := c.screen(c.bounds.min), c.screen(c.bounds.max)
		if a.X < 10 || a.Y < 10 || z.X > float32(size.X-10) || z.Y > float32(size.Y-10) {
			t.Errorf("%v: diagram bounds %v→%v outside fit margin", size, a, z)
		}
	}
	before := c.scale
	c.zoomBy(.8)
	if c.scale > before {
		t.Fatalf("zooming out increased scale from %g to %g", before, c.scale)
	}
}

func TestBoardCanvasFitRetainsMarginsAtDifferentSizesAndDPI(t *testing.T) {
	for _, subject := range []string{"biologia", "filosofia", "matematica"} {
		t.Run(subject, func(t *testing.T) {
			b := generatedExamBoard(t, subject)
			arrangeBoard(&b, nil)
			var c boardCanvas
			c.scene(&b)
			for _, viewport := range []struct {
				size image.Point
				dpi  float32
			}{{image.Pt(1350, 850), 1.5}, {image.Pt(420, 260), 1}, {image.Pt(1920, 1000), 1}} {
				c.frame(c.bounds, viewport.size, viewport.dpi)
				a, z := c.screen(c.bounds.min), c.screen(c.bounds.max)
				margin := 27 * viewport.dpi
				if a.X < margin || a.Y < margin || z.X > float32(viewport.size.X)-margin || z.Y > float32(viewport.size.Y)-margin {
					t.Fatalf("viewport %v at DPI %g clipped diagram bounds %v→%v", viewport.size, viewport.dpi, a, z)
				}
			}
			c.zoomBy(1.2)
			oldCenter := f32.Pt(float32(c.size.X)/2, float32(c.size.Y)/2).Sub(c.origin).Div(c.scale)
			c.frame(c.bounds, image.Pt(550, 400), 1.5)
			newCenter := f32.Pt(275, 200).Sub(c.origin).Div(c.scale)
			if oldCenter.Sub(newCenter).Round() != (image.Point{}) {
				t.Fatalf("resize changed the manual camera target %v → %v", oldCenter, newCenter)
			}
		})
	}
}
