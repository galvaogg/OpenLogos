package main

import (
	"context"
	"errors"
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type boardState struct {
	id, kind, editing  string
	source             studyAttachment
	canvas             boardCanvas
	create, refine     widget.Editor
	label, detail      widget.Editor
	title, edgeLabel   widget.Editor
	shape, connecting  string
	renaming, edgeOpen bool
	edgeIndex          int
	preview            *Board
	undo               map[string][]Board
	running            bool
	err, message       string
}

type boardJob struct {
	origin, prompt, target string
	chat                   generationDraft
}

func (s *studio) startBoard(kind, prompt, material string, current *Board, draft generationDraft) {
	if s.stop != nil {
		return
	}
	ui := &s.boardUI
	prompt = strings.TrimSpace(prompt)
	if !validText(prompt, 500) {
		ui.err = "Escreva um pedido de até 500 caracteres."
		return
	}
	if ui.preview != nil {
		ui.err = "Aplique ou descarte a proposta pendente antes de pedir outra."
		s.generationError = ui.err
		return
	}
	job := boardJob{origin: s.page, prompt: prompt, chat: draft}
	var snapshot *Board
	if current != nil {
		value := current.clone()
		snapshot, job.target = &value, current.ID
	}
	provider := s.lib.Provider
	s.generationError, s.activeTopic, s.showCommands = "", prompt, false
	ui.err, ui.message, ui.running = "", "", true
	ui.resetEdit()
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Minute)
	s.stop = cancel
	s.work.Add(1)
	go func() {
		defer s.work.Done()
		board, err := generateBoard(ctx, provider, kind, prompt, material, snapshot)
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		s.send(result{board: board, boardJob: &job, err: err})
	}()
}

func (s *studio) finishBoard(b Board, job boardJob, err error) {
	ui := &s.boardUI
	ui.running = false
	if err == nil {
		if job.target == "" {
			err = validateGeneratedBoard(b)
		} else {
			err = validateBoard(b)
		}
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			err = errors.New("O agente demorou demais. Tente novamente.")
		}
		ui.err = err.Error()
		if job.origin == "home" {
			s.generationError = ui.err
		}
		return
	}
	if job.target != "" {
		if current := s.lib.board(job.target); current != nil {
			b.ID = current.ID
			arrangeBoard(&b, current)
			ui.preview = &b
			if ui.id == b.ID {
				ui.canvas.fit()
			}
		}
		return
	}
	arrangeBoard(&b, nil)
	id := s.lib.addBoard(b)
	s.save()
	if job.origin == "home" {
		s.chat = append([]studyTurn{{prompt: job.chat.prompt, boardID: id}}, s.chat...)
		s.chat = s.chat[:min(4, len(s.chat))]
		if s.currentDraft() != job.chat {
			return
		}
		s.editor.SetText("")
		s.note, s.cardSource = "", studyAttachment{}
	}
	if job.origin == "boards" && strings.TrimSpace(ui.create.Text()) == job.prompt {
		ui.create.SetText("")
		ui.source = studyAttachment{}
	}
	if s.page == job.origin {
		s.openBoard(id)
	}
}

func (s *studio) openBoard(id string) {
	if s.lib.board(id) == nil {
		return
	}
	ui := &s.boardUI
	if ui.id != id {
		ui.id, ui.message, ui.err = id, "", ""
		ui.resetEdit()
		ui.canvas.selectNode("")
		ui.canvas.fit()
		ui.refine.SetText("")
	}
	s.goTo("board")
}

func (s *studio) boardFromLesson(l Lesson) {
	var content strings.Builder
	fmt.Fprintf(&content, "%s\n%s\n\n", l.Title, l.Summary)
	for _, card := range l.Cards {
		fmt.Fprintf(&content, "%s\n%s\n%s\n\n", card.Title, card.Body, card.Question)
	}
	s.boardUI.kind = "mindmap"
	s.boardUI.source = studyAttachment{label: l.Title, text: content.String()}
	s.boardUI.create.MaxLen = 500
	s.boardUI.create.SetText(l.Title)
	s.goTo("boards")
}

func (s *studio) boardHub() []view {
	ui := &s.boardUI
	if ui.kind == "" {
		ui.kind = "mindmap"
	}
	items := []view{gap(23), s.title("Quadros visuais", 38), gap(9),
		s.label("Monte um quadro em branco ou gere um diagrama com o agente.", 15, muted, font.Normal), gap(13),
		func(g ctx) dim {
			return s.button("board-blank", "Novo quadro em branco", "plus", primaryButton, s.createBlankBoard)(g)
		}, gap(12),
		s.boardCreationPanel, gap(24)}
	if ui.preview != nil {
		items = append(items, s.button("board-pending", "Revisar proposta pendente", "spark", activeButton, func() { s.openBoard(ui.preview.ID) }), gap(13))
	}
	items = append(items, s.label("Seus quadros", 20, ink, font.SemiBold), gap(9))
	if len(s.lib.Boards) == 0 {
		items = append(items, s.label("Seus quadros salvos aparecem aqui.", 14, muted, font.Normal), gap(10))
	}
	for _, b := range s.lib.Boards {
		items = append(items, s.boardTile(b), gap(7))
	}
	return append(items, gap(24))
}

func (s *studio) boardTile(b Board) view {
	return func(g ctx) dim {
		return s.control(g, "board-tile:"+s.page+":"+b.ID, "Abrir quadro "+b.Title, outlineButton, func() { s.openBoard(b.ID) },
			wide(padding(10, 13, 10, func(g ctx) dim {
				return (layout.Flex{Alignment: layout.Middle}).Layout(g,
					layout.Flexed(1, func(g ctx) dim {
						return column(g, s.label(b.Title, 16, ink, font.SemiBold), gap(3),
							s.label(fmt.Sprintf("%s · %d nós", boardKindName(b.Kind), len(b.Nodes)), 13, muted, font.Normal))
					}), layout.Rigid(glyph("right", subtle)))
			})))
	}
}

func (s *studio) boardCreationPanel(g ctx) dim {
	ui := &s.boardUI
	var submit func()
	if s.stop == nil {
		submit = func() { s.startBoard(ui.kind, ui.create.Text(), ui.source.text, nil, generationDraft{}) }
	}
	return glass(g, 15, wide(inset(17, func(g ctx) dim {
		kinds := []view{s.boardKindButton("mindmap"), gap(6), s.boardKindButton("flowchart"), gap(6), s.boardKindButton("timeline")}
		rows := []view{func(g ctx) dim { return row(g, kinds...) }, gap(11)}
		if ui.source.text != "" {
			rows = append(rows, s.button("board-source", ui.source.label+" · Remover anexo", "book", activeButton, func() {
				ui.source = studyAttachment{}
			}), gap(8))
		}
		rows = append(rows, s.boardEditor(&ui.create, "Pedido para o quadro", "Ex.: fases da Lua e como elas se relacionam", 500, submit), gap(10),
			func(g ctx) dim {
				if ui.running {
					return row(g, s.label("Preparando quadro…", 14, accent, font.SemiBold), gap(9),
						s.button("board-cancel-hub", "Cancelar", "", quietButton, s.cancelStudyJob))
				}
				return row(g, s.button("board-create", "Gerar quadro", "spark", primaryButton, submit), gap(7),
					s.button("board-example", "Criar exemplo offline", "right", outlineButton, func() {
						b := exampleBoard()
						id := s.lib.addBoard(b)
						s.save()
						s.openBoard(id)
					}))
			})
		if ui.err != "" {
			rows = append(rows, gap(8), s.label(ui.err, 13, bad, font.Normal))
		}
		return column(g, rows...)
	})))
}

func (s *studio) boardKindButton(kind string) view {
	a := outlineButton
	if kind == s.boardUI.kind {
		a = activeButton
	}
	return s.button("board-kind:"+kind, boardKindName(kind), "", a, func() { s.boardUI.kind = kind })
}

func (s *studio) boardEditor(editor *widget.Editor, label, hint string, limit int, submit func()) view {
	return func(g ctx) dim {
		editor.SingleLine, editor.Submit, editor.MaxLen = true, submit != nil, limit
		for {
			event, ok := editor.Update(g)
			if !ok {
				break
			}
			if _, ok := event.(widget.SubmitEvent); ok && submit != nil {
				submit()
			}
		}
		border := line
		if g.Source.Focused(editor) {
			border = accent
		}
		return box(g, canvas, border, 9, wide(padding(9, 11, 9, func(g ctx) dim {
			e := material.Editor(s.theme, editor, hint)
			e.TextSize, e.HintColor, e.Color = 14, subtle, ink
			m := op.Record(g.Ops)
			d := e.Layout(g)
			content := m.Stop()
			defer clip.Rect(image.Rectangle{Max: d.Size}).Push(g.Ops).Pop()
			semantic.LabelOp(label).Add(g.Ops)
			content.Add(g.Ops)
			return d
		})))
	}
}

func (s *studio) cancelStudyJob() {
	if s.stop != nil {
		s.stop()
	}
}

func (s *studio) exportBoard(b Board) {
	dir, err := studiesDir()
	if err == nil {
		var file string
		file, err = exportBoardExcalidraw(dir, b)
		if err == nil {
			s.boardUI.message, s.boardUI.err = "Salvo em Estudos · "+filepath.Base(file), ""
			return
		}
	}
	s.boardUI.err = err.Error()
}

func (s *studio) boardWorkspace(g ctx) dim {
	ui := &s.boardUI
	b := s.lib.board(ui.id)
	if b == nil {
		return inset(20, s.button("board-missing", "Voltar para quadros", "left", quietButton, func() { s.goTo("boards") }))(g)
	}
	display, preview := b, ui.preview != nil && ui.preview.ID == b.ID
	if preview {
		display = ui.preview
	}
	editable := !preview && !ui.running
	var refit, zoomIn, zoomOut = ui.canvas.fit, func() { ui.canvas.zoomBy(1.2) }, func() { ui.canvas.zoomBy(1 / 1.2) }
	return padding(13, 19, 15, func(g ctx) dim {
		return (layout.Flex{Axis: layout.Vertical}).Layout(g,
			layout.Rigid(wide(func(g ctx) dim {
				var export, organize, undo func()
				if editable {
					export = func() { s.exportBoard(*b) }
					if len(b.Nodes) > 0 {
						organize = func() {
							next := b.clone()
							arrangeBoard(&next, nil)
							if s.commitBoardEdit(b, next) {
								ui.canvas.fit()
							}
						}
					}
					if len(ui.undo[b.ID]) > 0 {
						undo = func() { s.undoBoardEdit(b) }
					}
				}
				header := []view{s.title(display.Title, 29), gap(4),
					s.label(fmt.Sprintf("%s · %d nós · %d conexões", boardKindName(display.Kind), len(display.Nodes), len(display.Edges)), 13, muted, font.Normal),
					gap(8), func(g ctx) dim {
						return (layout.Flex{Spacing: layout.SpaceBetween, Alignment: layout.Middle}).Layout(g,
							layout.Rigid(func(g ctx) dim {
								return row(g, s.button("board-fit", "Enquadrar", "", outlineButton, refit),
									gap(3), s.button("board-zoom-out", "−", "", quietButton, zoomOut),
									gap(3), s.button("board-zoom-in", "+", "", quietButton, zoomIn))
							}),
							layout.Rigid(func(g ctx) dim {
								return row(g, s.button("board-arrange", "Organizar", "", quietButton, organize), gap(3),
									s.button("board-undo", "Desfazer", "", quietButton, undo), gap(3),
									s.button("board-export", "Exportar", "", outlineButton, export))
							}))
					}, gap(8)}
				if editable {
					header = append(header, s.boardManualTools(b), gap(8))
				}
				return column(g, header...)
			})),
			layout.Flexed(1, func(g ctx) dim {
				g.Constraints.Min = g.Constraints.Max
				return ui.canvas.layout(g, display, editable, func(previous BoardNode) {
					before := b.clone()
					before.Nodes[boardCanvasIndex(before.Nodes, previous.ID)] = previous
					ui.remember(before)
					ui.message = ""
					s.save()
				})
			}),
			layout.Rigid(wide(padding(9, 0, 0, func(g ctx) dim {
				return s.boardDock(g, b, preview, editable)
			}))),
		)
	})(g)
}

func (s *studio) boardDock(g ctx, b *Board, preview, editable bool) dim {
	ui := &s.boardUI
	var content []view
	if preview {
		content = append(content, s.label("Proposta do agente", 14, accent, font.SemiBold), gap(6), func(g ctx) dim {
			return row(g, s.button("board-apply", "Aplicar proposta", "check", primaryButton, func() {
				if s.commitBoardEdit(b, *ui.preview) {
					ui.preview = nil
					ui.resetEdit()
					ui.refine.SetText("")
				}
			}), gap(6), s.button("board-discard", "Descartar", "", quietButton, func() {
				ui.preview = nil
				ui.canvas.fit()
			}))
		})
	} else if ui.running {
		content = append(content, func(g ctx) dim {
			return row(g, s.label("Agente preparando o quadro…", 14, accent, font.SemiBold), gap(7),
				s.button("board-cancel-edit", "Cancelar", "", quietButton, s.cancelStudyJob))
		})
	} else {
		if editable {
			content = append(content, s.boardManualDock(b))
		}
		var request func()
		if s.stop == nil {
			request = func() { s.startBoard(b.Kind, ui.refine.Text(), "", b, generationDraft{}) }
		}
		content = append(content, func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Flexed(1, s.boardEditor(&ui.refine, "Pedido de alteração ao agente", "Peça uma mudança no quadro. Ex.: acrescente um exemplo", 500, request)),
				layout.Rigid(gap(7)),
				layout.Rigid(s.button("board-refine", "Propor alteração", "spark", primaryButton, request)))
		})
	}
	switch {
	case ui.err != "":
		content = append(content, gap(5), s.label(ui.err, 13, bad, font.Normal))
	case s.localWarning != "":
		content = append(content, gap(5), s.label(s.localWarning, 13, bad, font.Normal))
	case ui.message != "":
		content = append(content, gap(5), s.label(ui.message, 13, good, font.Normal))
	}
	return column(g, content...)
}
