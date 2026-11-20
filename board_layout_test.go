package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gioui.org/f32"
)

func examBoard(kind string) Board {
	var label, detail, relation string
	switch kind {
	case "mindmap":
		label = "Fotofosforilação e transporte de elétrons nos tilacoides"
		detail = "O PS II oxida H₂O, b₆f gera gradiente de H⁺ e o fotossistema I reduz NADP⁺ a NADPH nos tilacoides."
		relation = "produz ATP e reduz NADP⁺ em NADPH"
	case "timeline":
		label = "Imperativo categórico e universalização das máximas"
		detail = "Para Kant, agir por dever é seguir uma máxima que a razão consiga querer como lei universal válida."
		relation = "situa o conceito no Iluminismo"
	case "flowchart":
		label = "Derivada e discriminante de uma função quadrática"
		detail = "Calcule Δ = b² − 4ac. Se Δ > 0, ache x₁ e x₂; use f′(x) e f″(x) para estudar variação e extremos."
		relation = "confirma se há duas raízes reais"
	}
	return Board{Kind: kind, Title: label, Nodes: []BoardNode{
		{ID: "a", Label: label, Detail: detail, Shape: "topic", X: -500, Y: -250},
		{ID: "b", Label: strings.Repeat("W", 58), Detail: strings.Repeat("W", 100), Shape: "note", X: 0, Y: 200},
	}, Edges: []BoardEdge{{From: "a", To: "b", Label: relation}}}
}

func TestBoardExportExamTextFitsInsideEditableNodes(t *testing.T) {
	for _, kind := range []string{"mindmap", "timeline", "flowchart"} {
		t.Run(kind, func(t *testing.T) {
			board := examBoard(kind)
			data, err := boardExcalidraw(board)
			if err != nil {
				t.Fatal(err)
			}
			var scene struct {
				Elements []struct {
					ID, Text, OriginalText, ContainerID string
					X, Y, Width, Height                 float64
				}
			}
			if err := json.Unmarshal(data, &scene); err != nil {
				t.Fatal(err)
			}
			for i, node := range board.Nodes {
				shape, label := scene.Elements[2+i*2], scene.Elements[3+i*2]
				if label.ContainerID != shape.ID || label.OriginalText != node.Label+"\n"+node.Detail {
					t.Fatal("editable node text lost its content or binding")
				}
				if label.X < shape.X+10 || label.X+label.Width > shape.X+shape.Width-10 ||
					label.Y < shape.Y+10 || label.Y+label.Height > shape.Y+shape.Height-10 {
					t.Errorf("node %s: label y=%g h=%g overflows the padded shape y=%g h=%g", node.ID, label.Y, label.Height, shape.Y, shape.Height)
				}
			}
		})
	}
}

func generatedExamBoard(t *testing.T, subject string) Board {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "board-layout", subject+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var b Board
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBoardArrangesGeneratedExamBoards(t *testing.T) {
	for _, subject := range []string{"biologia", "filosofia", "matematica"} {
		t.Run(subject, func(t *testing.T) {
			b := generatedExamBoard(t, subject)
			arrangeBoard(&b, nil)
			if err := validateBoard(b); err != nil {
				t.Fatal(err)
			}
			shaper := newBoardShaper()
			cards := measureBoardNodes(b.Nodes)
			rects := make([]boardRect, len(b.Nodes))
			withoutWhitespace := func(s string) string { return strings.Join(strings.Fields(s), "") }
			for i, node := range b.Nodes {
				card := cards[i]
				rect := card.rect(boardNodePosition(node))
				for j, other := range rects[:i] {
					if rect.outset(12).overlaps(other) {
						t.Errorf("nodes %s and %s overlap or lack spacing", node.ID, b.Nodes[j].ID)
					}
				}
				rects[i] = rect
				for _, content := range []struct {
					original string
					block    boardTextBlock
				}{{node.Label, card.title}, {node.Detail, card.detail}, {boardNodeText(node), card.exported}} {
					if withoutWhitespace(content.block.wrapped) != withoutWhitespace(content.original) {
						t.Errorf("wrapping lost characters in %q", content.original)
					}
					for _, line := range strings.Split(content.block.wrapped, "\n") {
						block := content.block
						if measured := measureBoardText(shaper, line, 1000, block.fontSize, block.lineHeight, block.weight); measured.width > boardNodeWidth-2*boardNodePad {
							t.Errorf("%q takes %d px outside the padded node", line, measured.width)
						}
					}
				}
				if float32(card.textHeight()+2*boardNodePad) > card.height {
					t.Errorf("%s lacks vertical space for complete text", node.ID)
				}
			}
			links, _ := layoutBoardEdges(shaper, b, rects)
			occupied := append([]boardRect(nil), rects...)
			for i, link := range links {
				if len(link.points) < 2 || withoutWhitespace(link.label.wrapped) != withoutWhitespace(b.Edges[i].Label) {
					t.Errorf("connector %d lost its route or label", i)
				}
				if link.label.wrapped != "" {
					for _, other := range occupied {
						if link.labelBox.overlaps(other) {
							t.Errorf("connector %d label overlaps another element", i)
						}
					}
					occupied = append(occupied, link.labelBox)
				}
			}
			data, err := boardExcalidraw(b)
			if err != nil {
				t.Fatal(err)
			}
			var scene struct {
				Elements []struct {
					ID, OriginalText    string
					X, Y, Width, Height float64
					Points              [][2]float64
				}
			}
			if err := json.Unmarshal(data, &scene); err != nil {
				t.Fatal(err)
			}
			for _, el := range scene.Elements {
				var i int
				if _, err := fmt.Sscanf(el.ID, "node-%d", &i); err == nil && el.ID == fmt.Sprintf("node-%d", i) {
					r := rects[i]
					if el.Width != float64(r.size().X) || el.Height != float64(r.size().Y) || el.X != float64(r.min.X) || el.Y != float64(r.min.Y) {
						t.Errorf("exported shape %s differs from canvas geometry", el.ID)
					}
				}
				if len(el.Points) > 0 {
					_, _ = fmt.Sscanf(el.ID, "edge-%d", &i)
					for j, p := range el.Points {
						if actual := f32.Pt(float32(el.X+p[0]), float32(el.Y+p[1])); actual != links[i].points[j] {
							t.Errorf("exported edge %s point %d differs from canvas geometry", el.ID, j)
						}
					}
				}
			}
		})
	}
}

func TestBoardRevisionPlacesNewNodeAroundResizedExistingNode(t *testing.T) {
	old := examBoard("timeline")
	old.Nodes[1].Label, old.Nodes[1].Detail = "Curto", ""
	old.Nodes[1].X, old.Nodes[1].Y = boardColumnStep+.123456789, -140.123456789
	next := old.clone()
	long := examBoard("timeline").Nodes[1]
	next.Nodes[1].Label, next.Nodes[1].Detail = long.Label, long.Detail
	next.Nodes = []BoardNode{next.Nodes[0], {ID: "new", Label: "Novo", Shape: "note"}, next.Nodes[1]}
	arrangeBoard(&next, &old)
	if next.Nodes[2].X != old.Nodes[1].X || next.Nodes[2].Y != old.Nodes[1].Y {
		t.Fatal("revision moved the existing node")
	}
	layout := measureBoardNodes(next.Nodes)
	added, resized := layout[1].rect(boardNodePosition(next.Nodes[1])), layout[2].rect(boardNodePosition(next.Nodes[2]))
	if added.overlaps(resized) || added.min.Y < resized.max.Y {
		t.Fatalf("new node %v did not clear resized node %v", added, resized)
	}
}

func TestBoardEdgeRoutesAroundInterveningNode(t *testing.T) {
	card := boardNodeLayout{width: boardNodeWidth, height: 140}
	rects := []boardRect{card.rect(f32.Pt(780, 0)), card.rect(f32.Pt(390, 0)), card.rect(f32.Pt(0, 0))}
	points, start, end := routeBoardEdge(rects, 0, 2)
	if len(points) != 4 || start[1] != end[1] || start[1] != 0 && start[1] != 1 {
		t.Fatalf("wrong route or bindings around obstacle: %v, %v, %v", points, start, end)
	}
	for i, p := range points[1:] {
		if rects[1].intersects(points[i], p) {
			t.Errorf("segment %v → %v crosses the intervening node", points[i], p)
		}
	}
}

func TestBoardArrangesLongFlowchartInBands(t *testing.T) {
	b := Board{Kind: "flowchart"}
	for i := range 24 {
		id := fmt.Sprintf("step-%d", i)
		if i > 0 {
			b.Edges = append(b.Edges, BoardEdge{From: b.Nodes[i-1].ID, To: id})
		}
		b.Nodes = append(b.Nodes, BoardNode{ID: id, Shape: "step", Label: strings.Repeat("W", 58), Detail: strings.Repeat("W", 100)})
	}
	arrangeBoard(&b, nil)
	for _, node := range b.Nodes {
		if node.X > 3*boardColumnStep {
			t.Fatalf("long flow ran off the fourth column at %q", node.ID)
		}
	}
	if b.Nodes[3].X != b.Nodes[4].X || b.Nodes[4].Y <= b.Nodes[3].Y {
		t.Fatal("flowchart did not continue below the last column")
	}
}
