package main

import (
	"image"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
)

func boardNodeClip(ops *op.Ops, r image.Rectangle, radius int, kind string) clip.Op {
	switch kind {
	case "ellipse":
		return clip.Ellipse(r).Op(ops)
	case "diamond":
		center := r.Min.Add(r.Size().Div(2))
		var path clip.Path
		path.Begin(ops)
		path.MoveTo(f32.Pt(float32(center.X), float32(r.Min.Y)))
		path.LineTo(f32.Pt(float32(r.Max.X), float32(center.Y)))
		path.LineTo(f32.Pt(float32(center.X), float32(r.Max.Y)))
		path.LineTo(f32.Pt(float32(r.Min.X), float32(center.Y)))
		path.Close()
		return clip.Outline{Path: path.End()}.Op()
	case "text":
		return clip.Rect(r).Op()
	default:
		return clip.UniformRRect(r, radius).Op(ops)
	}
}
