package main

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/io/pointer"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type ctx = layout.Context
type dim = layout.Dimensions
type view = layout.Widget

var (
	canvas       = hex(0x111316)
	side         = hex(0x15171b)
	panel        = hex(0x1a1d22)
	hover        = hex(0x252a32)
	line         = hex(0x2c3037)
	ink          = hex(0xf0f0f4)
	muted        = hex(0xa9aebc)
	subtle       = hex(0x939bad)
	accent       = hex(0xb4c3ff)
	accentHover  = hex(0xcbd4ff)
	night        = hex(0x12162e)
	selected     = hex(0x292f47)
	good         = hex(0x9be0ba)
	bad          = hex(0xffaeaa)
	clear        = color.NRGBA{}
	reduceMotion bool
)

func hex(c uint32) color.NRGBA {
	return color.NRGBA{R: byte(c >> 16), G: byte(c >> 8), B: byte(c), A: 255}
}

func gap(d unit.Dp) view {
	return layout.Spacer{Width: d, Height: d}.Layout
}

func inset(d unit.Dp, content view) view {
	return func(g ctx) dim { return layout.UniformInset(d).Layout(g, content) }
}

func padding(top, sides, bottom unit.Dp, content view) view {
	return func(g ctx) dim {
		return (layout.Inset{Top: top, Left: sides, Right: sides, Bottom: bottom}).Layout(g, content)
	}
}

func column(g ctx, children ...view) dim {
	items := make([]layout.FlexChild, 0, len(children))
	for _, child := range children {
		items = append(items, layout.Rigid(child))
	}
	return (layout.Flex{Axis: layout.Vertical}).Layout(g, items...)
}

func row(g ctx, children ...view) dim {
	items := make([]layout.FlexChild, 0, len(children))
	for _, child := range children {
		items = append(items, layout.Rigid(child))
	}
	return (layout.Flex{Alignment: layout.Middle}).Layout(g, items...)
}

func wide(w view) view {
	return func(g ctx) dim {
		g.Constraints.Min.X = g.Constraints.Max.X
		return w(g)
	}
}

func fixedWidth(d unit.Dp, w view) view {
	return func(g ctx) dim {
		g.Constraints.Max.X = min(g.Dp(d), g.Constraints.Max.X)
		g.Constraints.Min.X = g.Constraints.Max.X
		return w(g)
	}
}

func rule(g ctx) dim {
	size := image.Pt(g.Constraints.Max.X, 1)
	paint.FillShape(g.Ops, line, clip.Rect{Max: size}.Op())
	return dim{Size: size}
}

func dot(g ctx, x, y, rx, ry int, c color.NRGBA) {
	r := image.Rect(x-rx, y-ry, x+rx, y+ry)
	paint.FillShape(g.Ops, c, clip.Ellipse(r).Op(g.Ops))
}

func round(g ctx, r image.Rectangle, radius unit.Dp, c color.NRGBA) {
	if c.A != 0 && !r.Empty() {
		paint.FillShape(g.Ops, c, clip.UniformRRect(r, g.Dp(radius)).Op(g.Ops))
	}
}

func box(g ctx, fill, border color.NRGBA, radius unit.Dp, w view) dim {
	m := op.Record(g.Ops)
	d := w(g)
	content := m.Stop()
	bounds := image.Rectangle{Max: d.Size}
	if border.A != 0 {
		round(g, bounds, radius, border)
		bounds = bounds.Inset(1)
	}
	round(g, bounds, radius, fill)
	content.Add(g.Ops)
	return d
}

func (s *studio) label(value string, size unit.Sp, c color.NRGBA, weight font.Weight) view {
	return func(g ctx) dim {
		l := material.Label(s.theme, size, value)
		l.Color, l.Font.Weight = c, weight
		return l.Layout(g)
	}
}

func (s *studio) title(value string, size unit.Sp) view {
	return func(g ctx) dim {
		l := material.Label(s.theme, size, value)
		l.Color, l.Font.Typeface = ink, "Georgia"
		return l.Layout(g)
	}
}

type appearance struct {
	fill, border, fg color.NRGBA
}

var (
	quietButton   = appearance{clear, clear, muted}
	outlineButton = appearance{panel, line, ink}
	activeButton  = appearance{selected, selected, accent}
	primaryButton = appearance{accent, clear, night}
)

func (s *studio) control(g ctx, id, name string, a appearance, action func(), content view) dim {
	b := s.clicks[id]
	if b == nil {
		b = new(widget.Clickable)
		s.clicks[id] = b
	}
	if action == nil {
		g = g.Disabled()
		a.fg = subtle
	}
	if b.Clicked(g) && action != nil {
		action()
		s.window.Invalidate()
	}
	return b.Layout(g, func(g ctx) dim {
		if b.Hovered() && g.Enabled() {
			if a.fill == accent {
				a.fill = accentHover
			} else if a.fill == clear || a.fill == panel {
				a.fill = hover
			}
			pointer.CursorPointer.Add(g.Ops)
		}
		if g.Source.Focused(b) {
			a.border = accent
		}
		semantic.Button.Add(g.Ops)
		semantic.LabelOp(name).Add(g.Ops)
		return box(g, a.fill, a.border, 9, content)
	})
}

func (s *studio) button(id, label, icon string, a appearance, action func()) view {
	if action == nil {
		a.fg = hex(0x717987)
	}
	return func(g ctx) dim {
		return s.control(g, id, label, a, action, padding(9, 12, 9, func(g ctx) dim {
			l := s.label(label, 14, a.fg, font.SemiBold)
			if icon == "" {
				return l(g)
			}
			return row(g, glyph(icon, a.fg), gap(8), l)
		}))
	}
}

func glyph(name string, c color.NRGBA) view {
	return func(g ctx) dim {
		size := g.Dp(18)
		u := float32(size) / 18
		p := func(x, y float32) f32.Point { return f32.Pt(x*u, y*u) }
		var path clip.Path
		path.Begin(g.Ops)
		switch name {
		case "plus":
			path.MoveTo(p(9, 3))
			path.LineTo(p(9, 15))
			path.MoveTo(p(3, 9))
			path.LineTo(p(15, 9))
		case "spark":
			path.MoveTo(p(9, 2))
			path.LineTo(p(11, 7))
			path.LineTo(p(16, 9))
			path.LineTo(p(11, 11))
			path.LineTo(p(9, 16))
			path.LineTo(p(7, 11))
			path.LineTo(p(2, 9))
			path.LineTo(p(7, 7))
			path.Close()
		case "book":
			path.MoveTo(p(9, 5))
			path.LineTo(p(4, 3))
			path.LineTo(p(2, 4))
			path.LineTo(p(2, 13))
			path.LineTo(p(4, 12))
			path.LineTo(p(9, 14))
			path.LineTo(p(14, 12))
			path.LineTo(p(16, 13))
			path.LineTo(p(16, 4))
			path.LineTo(p(14, 3))
			path.LineTo(p(9, 5))
			path.LineTo(p(9, 14))
		case "folder":
			path.MoveTo(p(2, 5))
			path.LineTo(p(7, 5))
			path.LineTo(p(9, 7))
			path.LineTo(p(16, 7))
			path.LineTo(p(16, 14))
			path.LineTo(p(2, 14))
			path.Close()
		case "left":
			path.MoveTo(p(9, 4))
			path.LineTo(p(4, 9))
			path.LineTo(p(9, 14))
			path.MoveTo(p(4, 9))
			path.LineTo(p(15, 9))
		case "right":
			path.MoveTo(p(9, 4))
			path.LineTo(p(14, 9))
			path.LineTo(p(9, 14))
			path.MoveTo(p(3, 9))
			path.LineTo(p(14, 9))
		case "terminal":
			path.MoveTo(p(3, 4))
			path.LineTo(p(8, 9))
			path.LineTo(p(3, 14))
			path.MoveTo(p(9, 14))
			path.LineTo(p(16, 14))
		case "check":
			path.MoveTo(p(3, 9))
			path.LineTo(p(7, 13))
			path.LineTo(p(15, 5))
		}
		paint.FillShape(g.Ops, c, clip.Stroke{Path: path.End(), Width: 1.5 * u}.Op())
		return dim{Size: image.Pt(size, size)}
	}
}

// The mascot shares the app's progress dots and only animates while the studio is in use.
func mascot(g ctx, width, height unit.Dp, focus, working bool) dim {
	w, h := g.Dp(width), g.Dp(height)
	scale := float32(min(w, h)) / 112
	offset := op.Offset(image.Pt((w-int(112*scale))/2, (h-int(112*scale))/2)).Push(g.Ops)
	defer offset.Pop()
	for y := 7; y <= 105; y += 9 {
		for x := 3; x <= 111; x += 9 {
			dx, dy := x-57, y-54
			if dx*dx+dy*dy < 3100 {
				c := hex(0x303959)
				if (x+2*y)%5 == 0 {
					c = hex(0x515d89)
				}
				dot(g, int(float32(x)*scale), int(float32(y)*scale), max(1, int(scale)), max(1, int(scale)), c)
			}
		}
	}
	t := float64(g.Now.UnixMilli()%6000) / 1000
	lift := float32(0)
	blink := !reduceMotion && (working || focus) && t >= 5.65 && t < 5.86
	if working && !reduceMotion {
		lift = 4 * float32(math.Sin(t*math.Pi*2/1.5))
		g.Execute(op.InvalidateCmd{At: g.Now.Add(time.Second / 30)})
	} else if focus && !reduceMotion {
		next := 5.65 - t
		if blink {
			next = 5.86 - t
		} else if next < 0 {
			next += 6
		}
		g.Execute(op.InvalidateCmd{At: g.Now.Add(max(time.Millisecond, time.Duration(next*float64(time.Second))))})
	}
	p := func(x, y float32) f32.Point { return f32.Pt(x*scale, (y+lift)*scale) }
	dot(g, int(57*scale), int(96*scale), int(27*scale), max(1, int(3*scale)), hex(0x202747))
	var shape clip.Path
	shape.Begin(g.Ops)
	shape.MoveTo(p(18, 66))
	shape.CubeTo(p(15, 46), p(25, 34), p(39, 27))
	shape.CubeTo(p(43, 25), p(44, 12), p(51, 14))
	shape.CubeTo(p(57, 14), p(58, 21), p(62, 22))
	shape.CubeTo(p(79, 22), p(93, 34), p(97, 55))
	shape.CubeTo(p(104, 78), p(86, 91), p(58, 91))
	shape.CubeTo(p(35, 92), p(21, 85), p(18, 66))
	shape.Close()
	mask := clip.Outline{Path: shape.End()}.Op().Push(g.Ops)
	paint.LinearGradientOp{Stop1: p(37, 19), Stop2: p(74, 94), Color1: hex(0xe0e6ff), Color2: hex(0x6779e3)}.Add(g.Ops)
	paint.PaintOp{}.Add(g.Ops)
	mask.Pop()
	ry := 9
	if blink {
		ry = 2
	}
	look := 0
	if focus || working {
		look = -4
	}
	c := hex(0x1e254f)
	dot(g, int(float32(46+look)*scale), int((57+lift)*scale), max(1, int(6*scale)), max(1, int(float32(ry)*scale)), c)
	dot(g, int(float32(70+look)*scale), int((54+lift)*scale), max(1, int(6*scale)), max(1, int(float32(ry)*scale)), c)
	return dim{Size: image.Pt(w, h)}
}
