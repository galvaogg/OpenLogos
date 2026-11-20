package main

import (
	"fmt"
	"image"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op"
)

func BenchmarkGioFrame(b *testing.B) {
	for _, page := range []string{"board", "reader", "notes"} {
		b.Run(page, func(b *testing.B) {
			lesson := sampleLessons()[0]
			s := studio{
				theme: studioTheme(), page: page, lesson: &lesson, samples: sampleLessons(),
				lib:    Library{Provider: codex, Feedback: instant, Progress: make(map[string]*Progress)},
				clicks: make(map[string]*studioButton),
				body:   layout.List{Axis: layout.Vertical}, rail: layout.List{Axis: layout.Vertical},
				detail: layout.List{Axis: layout.Vertical},
			}
			for i := range 300 {
				l := lesson
				l.ID, l.Title = fmt.Sprintf("ai-bench-%d", i), fmt.Sprintf("Lição de revisão %d", i)
				s.lib.Generated = append(s.lib.Generated, l)
				s.lib.progress(l).Notes = map[int]string{0: "Explique o conceito com um exemplo próprio."}
			}
			board := exampleBoard()
			for i := len(board.Nodes); i < 24; i++ {
				id := fmt.Sprintf("branch-%d", i)
				board.Nodes = append(board.Nodes, BoardNode{ID: id, Label: "Relação entre os conceitos",
					Detail: "A ideia principal, sua aplicação e um exemplo de revisão.", Shape: "step",
					X: float64((i%6)-3) * 360, Y: float64(i/6-1) * 180})
				board.Edges = append(board.Edges, BoardEdge{From: board.Nodes[i-1].ID, To: id, Label: "exemplo prático"})
			}
			s.boardUI.id = s.lib.addBoard(board)
			var router input.Router
			var ops op.Ops
			var origin f32.Point
			frame := func(i int) {
				ops.Reset()
				g := ctx{Now: time.Unix(1_000_000, int64(i)*16_666_667),
					Constraints: layout.Exact(image.Pt(1260, 810)), Ops: &ops, Source: router.Source()}
				if page == "board" && i != 0 {
					s.boardUI.canvas.origin = origin.Add(f32.Pt(float32(i%60*3), float32(i%60)))
				}
				s.layout(g)
				router.Frame(&ops)
			}
			frame(0)
			origin = s.boardUI.canvas.origin
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				frame(i + 1)
			}
		})
	}
}
