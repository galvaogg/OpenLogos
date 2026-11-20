package main

import (
	"image"
	"image/color"
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

type studioButton struct {
	widget.Clickable
	from, target float32
	started      time.Time
}

func easeOut(now, started time.Time, duration time.Duration) float32 {
	if started.IsZero() {
		return 1
	}
	left := 1 - min(1, max(0, float32(now.Sub(started))/float32(duration)))
	return 1 - left*left*left
}

func (b *studioButton) emphasis(g ctx, target float32) float32 {
	if reduceMotion {
		return target
	}
	progress := easeOut(g.Now, b.started, 120*time.Millisecond)
	value := b.from + (b.target-b.from)*progress
	if b.target != target {
		b.from, b.target, b.started = value, target, g.Now
		progress = 0
	}
	if progress < 1 {
		g.Execute(op.InvalidateCmd{At: g.Now.Add(time.Second / 60)})
	}
	return value
}

func mixColor(a, b color.NRGBA, amount float32) color.NRGBA {
	if a.A == 0 {
		a.R, a.G, a.B = b.R, b.G, b.B
	}
	channel := func(x, y uint8) uint8 {
		return uint8(float32(x) + (float32(y)-float32(x))*amount)
	}
	return color.NRGBA{R: channel(a.R, b.R), G: channel(a.G, b.G), B: channel(a.B, b.B), A: channel(a.A, b.A)}
}

func enter(g ctx, started time.Time, x, y unit.Dp, w view) dim {
	progress := easeOut(g.Now, started, 200*time.Millisecond)
	if reduceMotion || progress == 1 {
		return w(g)
	}
	g.Execute(op.InvalidateCmd{At: g.Now.Add(time.Second / 60)})
	left := 1 - progress
	offset := op.Offset(image.Pt(int(math.Round(float64(float32(g.Dp(x))*left))),
		int(math.Round(float64(float32(g.Dp(y))*left))))).Push(g.Ops)
	defer offset.Pop()
	opacity := paint.PushOpacity(g.Ops, 0.68+0.32*progress)
	defer opacity.Pop()
	return w(g)
}

func glass(g ctx, radius unit.Dp, w view) dim {
	record := op.Record(g.Ops)
	d := w(g)
	content := record.Stop()
	bounds := image.Rectangle{Max: d.Size}
	if !bounds.Empty() {
		rim := clip.UniformRRect(bounds, g.Dp(radius)).Op(g.Ops).Push(g.Ops)
		gradient(g, f32.Pt(float32(d.Size.X)*0.8, 0), f32.Pt(float32(d.Size.X)*0.4, float32(d.Size.Y)),
			hex(0x535e82), hex(0x292c3d))
		rim.Pop()
		inner := bounds.Inset(1)
		if !inner.Empty() {
			mask := clip.UniformRRect(inner, g.Dp(radius)-1).Op(g.Ops).Push(g.Ops)
			gradient(g, f32.Pt(float32(d.Size.X)*0.84, 0), f32.Pt(float32(d.Size.X)*0.35, float32(d.Size.Y)),
				hex(0x2a304a), hex(0x191c27))
			gradient(g, f32.Pt(0, 1), f32.Pt(0, float32(g.Dp(36))),
				color.NRGBA{R: 190, G: 203, B: 255, A: 18}, color.NRGBA{R: 190, G: 203, B: 255})
			mask.Pop()
		}
	}
	content.Add(g.Ops)
	return d
}

func gradient(g ctx, from, to f32.Point, start, end color.NRGBA) {
	paint.LinearGradientOp{Stop1: from, Stop2: to, Color1: start, Color2: end}.Add(g.Ops)
	paint.PaintOp{}.Add(g.Ops)
}

type mascotFrame struct {
	lift, eyes, ripple float32
	next               time.Duration
}

func mascotMotion(now time.Time, working bool) mascotFrame {
	const cycle = 8400 * time.Millisecond
	const pulse = 780 * time.Millisecond
	const frameTime = time.Second / 30
	frame := mascotFrame{eyes: 1}
	phase := time.Duration(now.UnixMilli()%cycle.Milliseconds()) * time.Millisecond
	age := phase - (cycle - pulse)
	if age < 0 {
		frame.next = -age
	} else {
		progress := float64(age) / float64(pulse)
		frame.ripple = float32(progress)
		frame.lift = -1.8 * float32(math.Sin(math.Pi*progress))
		frame.eyes -= 0.8 * float32(max(0, 1-math.Abs(progress-0.64)/0.15))
		frame.next = frameTime
	}
	if working {
		frame.lift = 4 * float32(math.Sin(float64(now.UnixMilli()%1600)*2*math.Pi/1600))
		frame.next = frameTime
	}
	return frame
}
