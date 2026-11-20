package main

import (
	"crypto/rand"
	"fmt"
	"slices"
	"strings"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
)

var boardManualShapes = []struct{ shape, label string }{
	{"step", "Retângulo"}, {"ellipse", "Elipse"}, {"diamond", "Losango"}, {"text", "Texto"},
}

func (ui *boardState) resetEdit() {
	ui.editing, ui.connecting = "", ""
	ui.renaming, ui.edgeOpen = false, false
}

func (ui *boardState) remember(b Board) {
	const limit = 32
	if ui.undo == nil {
		ui.undo = make(map[string][]Board)
	}
	stack := ui.undo[b.ID]
	if len(stack) == limit {
		stack[0] = Board{}
		stack = stack[1:]
	}
	ui.undo[b.ID] = append(stack, b)
}

func (s *studio) commitBoardEdit(b *Board, next Board) bool {
	ui := &s.boardUI
	if err := validateBoard(next); err != nil {
		ui.err = err.Error()
		return false
	}
	ui.err, ui.message = "", ""
	if b.Title == next.Title && b.Kind == next.Kind && slices.Equal(b.Nodes, next.Nodes) && slices.Equal(b.Edges, next.Edges) {
		return true
	}
	ui.remember(*b)
	*b = next
	s.save()
	return true
}

func (s *studio) undoBoardEdit(b *Board) {
	ui := &s.boardUI
	stack := ui.undo[b.ID]
	if len(stack) == 0 {
		return
	}
	*b = stack[len(stack)-1]
	stack[len(stack)-1] = Board{}
	ui.undo[b.ID] = stack[:len(stack)-1]
	ui.resetEdit()
	ui.err, ui.message = "", ""
	s.save()
}

func (s *studio) createBlankBoard() {
	b := Board{Title: "Quadro sem título", Kind: s.boardUI.kind}
	id := s.lib.addBoard(b)
	s.save()
	s.openBoard(id)
}

func (s *studio) boardManualTools(b *Board) view {
	ui := &s.boardUI
	return func(g ctx) dim {
		buttons := make([]view, 0, 2*len(boardManualShapes)+1)
		for _, item := range boardManualShapes {
			var add func()
			if len(b.Nodes) < boardMaxNodes {
				add = func() { s.addBoardNode(b, item.shape, item.label) }
			}
			buttons = append(buttons, s.button("board-add:"+item.shape, "+ "+item.label, "", outlineButton, add), gap(4))
		}
		buttons = append(buttons, s.button("board-rename", "Renomear quadro", "", quietButton, func() {
			ui.resetEdit()
			ui.title.MaxLen = 90
			ui.title.SetText(b.Title)
			ui.renaming, ui.err = true, ""
		}))
		return row(g, buttons...)
	}
}

func (s *studio) addBoardNode(b *Board, shape, label string) {
	ui := &s.boardUI
	c := &ui.canvas
	var p f32.Point
	if c.scale > 0 {
		center := f32.Pt(float32(c.size.X)/2, float32(c.size.Y)/2)
		p = center.Sub(c.origin).Div(c.scale).Sub(f32.Pt(boardNodeWidth/2, boardNodeHeight/2))
	}
	node := BoardNode{ID: "node-" + strings.ToLower(rand.Text()), Label: label, Shape: shape}
	for range len(b.Nodes) + 1 {
		node.X = min(10000, max(-10000, float64(p.X)))
		node.Y = min(10000, max(-10000, float64(p.Y)))
		if !slices.ContainsFunc(b.Nodes, func(n BoardNode) bool { return n.X == node.X && n.Y == node.Y }) {
			break
		}
		p = p.Add(f32.Pt(24, 24))
	}
	next := b.clone()
	next.Nodes = append(next.Nodes, node)
	if s.commitBoardEdit(b, next) {
		c.selectNode(node.ID)
		ui.editNode(node)
	}
}

func (ui *boardState) editNode(node BoardNode) {
	ui.resetEdit()
	ui.label.MaxLen, ui.detail.MaxLen = 58, 100
	ui.label.SetText(node.Label)
	ui.detail.SetText(node.Detail)
	ui.editing, ui.shape, ui.err = node.ID, node.Shape, ""
}

func (s *studio) boardManualDock(b *Board) view {
	return func(g ctx) dim {
		ui := &s.boardUI
		var controls view
		i := boardCanvasIndex(b.Nodes, ui.canvas.selected)
		switch {
		case ui.renaming:
			save := func() {
				next := b.clone()
				next.Title = strings.TrimSpace(ui.title.Text())
				if s.commitBoardEdit(b, next) {
					ui.resetEdit()
				}
			}
			controls = func(g ctx) dim {
				return (layout.Flex{Alignment: layout.Middle}).Layout(g,
					layout.Flexed(1, s.boardEditor(&ui.title, "Título do quadro", "Título do quadro", 90, save)),
					layout.Rigid(gap(5)),
					layout.Rigid(s.button("board-title-save", "Salvar título", "", outlineButton, save)),
					layout.Rigid(s.button("board-title-cancel", "Cancelar", "", quietButton, ui.resetEdit)))
			}
		case ui.connecting != "":
			controls = s.boardConnectionTarget(b)
		case i >= 0:
			if ui.editing != "" && ui.editing != ui.canvas.selected {
				ui.resetEdit()
			}
			controls = s.boardNodeControls(b, i)
		default:
			ui.resetEdit()
			return dim{}
		}
		return column(g, controls, gap(7))
	}
}

func (s *studio) boardNodeControls(b *Board, i int) view {
	ui := &s.boardUI
	node := b.Nodes[i]
	if ui.editing == node.ID && !ui.edgeOpen {
		return s.boardNodeEditor(b, i)
	}
	edges := make([]int, 0, len(b.Edges))
	for j, edge := range b.Edges {
		if edge.From == node.ID || edge.To == node.ID {
			edges = append(edges, j)
		}
	}
	if ui.edgeOpen && ui.editing == node.ID && len(edges) > 0 {
		return s.boardConnectionEditor(b, edges)
	}
	var connect, connections func()
	if len(b.Nodes) > 1 && len(b.Edges) < boardMaxEdges {
		connect = func() { ui.resetEdit(); ui.connecting = node.ID }
	}
	if len(edges) > 0 {
		connections = func() { ui.openConnection(b, node.ID, edges, 0) }
	}
	return func(g ctx) dim {
		return row(g, s.button("board-node-edit", "Editar nó selecionado", "", quietButton, func() { ui.editNode(node) }),
			gap(4), s.button("board-connect", "Conectar", "", outlineButton, connect),
			gap(4), s.button("board-connections", fmt.Sprintf("Conexões (%d)", len(edges)), "", quietButton, connections),
			gap(4), s.button("board-node-remove", "Excluir nó", "", quietButton, func() {
				next := b.clone()
				next.Nodes = slices.Delete(next.Nodes, i, i+1)
				next.Edges = slices.DeleteFunc(next.Edges, func(edge BoardEdge) bool {
					return edge.From == node.ID || edge.To == node.ID
				})
				if s.commitBoardEdit(b, next) {
					ui.resetEdit()
				}
			}))
	}
}

func (s *studio) boardNodeEditor(b *Board, i int) view {
	ui := &s.boardUI
	save := func() {
		next := b.clone()
		node := &next.Nodes[i]
		node.Label, node.Detail, node.Shape = strings.TrimSpace(ui.label.Text()), strings.TrimSpace(ui.detail.Text()), ui.shape
		if s.commitBoardEdit(b, next) {
			ui.resetEdit()
		}
	}
	return func(g ctx) dim {
		shapes := make([]view, 0, 2*len(boardManualShapes))
		for _, item := range boardManualShapes {
			a := quietButton
			if ui.shape == item.shape {
				a = activeButton
			}
			shapes = append(shapes, s.button("board-node-shape:"+item.shape, item.label, "", a, func() { ui.shape = item.shape }), gap(3))
		}
		return column(g, func(g ctx) dim { return row(g, shapes...) }, gap(4), func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Flexed(1, s.boardEditor(&ui.label, "Título do nó", "Título", 58, save)),
				layout.Rigid(gap(5)),
				layout.Flexed(1, s.boardEditor(&ui.detail, "Detalhe do nó", "Detalhe", 100, save)),
				layout.Rigid(gap(5)),
				layout.Rigid(s.button("board-node-save", "Salvar nó", "", outlineButton, save)),
				layout.Rigid(s.button("board-node-cancel", "Cancelar", "", quietButton, ui.resetEdit)))
		})
	}
}

func (s *studio) boardConnectionTarget(b *Board) view {
	ui := &s.boardUI
	from, to := ui.connecting, ui.canvas.selected
	var add func()
	if from != to && boardCanvasIndex(b.Nodes, to) >= 0 && !slices.ContainsFunc(b.Edges, func(e BoardEdge) bool {
		return e.From == from && e.To == to
	}) {
		add = func() {
			next := b.clone()
			next.Edges = append(next.Edges, BoardEdge{From: from, To: to})
			if s.commitBoardEdit(b, next) {
				ui.canvas.selectNode(from)
				ui.resetEdit()
			}
		}
	}
	return func(g ctx) dim {
		return (layout.Flex{Alignment: layout.Middle}).Layout(g,
			layout.Flexed(1, s.label("Selecione o nó de destino.", 14, muted, font.Normal)),
			layout.Rigid(s.button("board-connect-add", "Criar conexão", "", outlineButton, add)),
			layout.Rigid(s.button("board-connect-cancel", "Cancelar", "", quietButton, ui.resetEdit)))
	}
}

func (ui *boardState) openConnection(b *Board, node string, indices []int, index int) {
	ui.resetEdit()
	ui.edgeOpen, ui.edgeIndex, ui.editing, ui.err = true, index, node, ""
	ui.edgeLabel.MaxLen = 35
	ui.edgeLabel.SetText(b.Edges[indices[index]].Label)
}

func (s *studio) boardConnectionEditor(b *Board, indices []int) view {
	ui := &s.boardUI
	i := indices[ui.edgeIndex]
	edge := b.Edges[i]
	from, to := b.Nodes[boardCanvasIndex(b.Nodes, edge.From)], b.Nodes[boardCanvasIndex(b.Nodes, edge.To)]
	save := func() {
		next := b.clone()
		next.Edges[i].Label = strings.TrimSpace(ui.edgeLabel.Text())
		if s.commitBoardEdit(b, next) {
			ui.resetEdit()
		}
	}
	return func(g ctx) dim {
		var nextEdge func()
		if len(indices) > 1 {
			nextEdge = func() { ui.openConnection(b, ui.editing, indices, (ui.edgeIndex+1)%len(indices)) }
		}
		return column(g, func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Flexed(1, s.label(from.Label+" → "+to.Label, 14, muted, font.Normal)),
				layout.Rigid(s.button("board-edge-next", "Próxima conexão", "", quietButton, nextEdge)))
		}, func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Flexed(1, s.boardEditor(&ui.edgeLabel, "Rótulo da conexão", "Rótulo opcional", 35, save)),
				layout.Rigid(gap(5)),
				layout.Rigid(s.button("board-edge-save", "Salvar conexão", "", outlineButton, save)),
				layout.Rigid(s.button("board-edge-remove", "Excluir conexão", "", quietButton, func() {
					next := b.clone()
					next.Edges = slices.Delete(next.Edges, i, i+1)
					if s.commitBoardEdit(b, next) {
						ui.resetEdit()
					}
				})),
				layout.Rigid(s.button("board-edge-cancel", "Cancelar", "", quietButton, ui.resetEdit)))
		})
	}
}
