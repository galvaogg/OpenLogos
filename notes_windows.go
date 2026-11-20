package main

import (
	"fmt"
	"time"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/io/semantic"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type readerState struct {
	cardLesson string
	cardIndex  int
	cardMotion time.Time
	cardShift  unit.Dp

	hintOpen       bool
	hintMotion     time.Time
	feedbackMotion time.Time

	noteOpen   bool
	noteReveal bool
	noteMotion time.Time
	noteDue    time.Time
	noteEditor widget.Editor
}

func (s *studio) loadReaderCard(l Lesson, p *Progress) {
	if s.cardLesson == l.ID && s.cardIndex == p.Cursor {
		return
	}
	s.flushReaderNote()
	s.cardShift = 10
	if s.cardLesson == l.ID && p.Cursor < s.cardIndex {
		s.cardShift = -10
	}
	s.cardLesson, s.cardIndex, s.cardMotion = l.ID, p.Cursor, time.Now()
	s.hintOpen, s.noteOpen, s.noteReveal = false, false, false
	s.hintMotion, s.feedbackMotion = time.Time{}, time.Time{}
	s.noteEditor.MaxLen = maxCardNote
	s.noteEditor.SetText(p.Notes[p.Cursor])
}

func (s *studio) tickReaderNote(g ctx) {
	if s.page == "reader" && s.noteOpen && s.lesson.ID == s.cardLesson {
		for {
			if _, ok := s.noteEditor.Update(g); !ok {
				break
			}
		}
		p := s.lib.progress(*s.lesson)
		if p.setCardNote(s.cardIndex, len(s.lesson.Cards), s.noteEditor.Text()) {
			s.noteDue = g.Now.Add(450 * time.Millisecond)
		}
	}
	if s.noteDue.IsZero() {
		return
	}
	if g.Now.Before(s.noteDue) {
		g.Execute(op.InvalidateCmd{At: s.noteDue})
	} else {
		s.flushReaderNote()
	}
}

func (s *studio) flushReaderNote() {
	if !s.noteDue.IsZero() {
		s.noteDue = time.Time{}
		s.save()
	}
}

func (s *studio) cardActions(l Lesson, p *Progress, i int) view {
	return func(g ctx) dim {
		label, a := "Escrever nota", outlineButton
		if s.noteOpen {
			label, a = "Fechar nota", activeButton
		} else if p.Notes[i] != "" {
			label = "Minha nota"
		}
		id := fmt.Sprintf("note:%s:%d", l.ID, i)
		return row(g, s.button(id, label, "", a, func() {
			s.noteOpen = !s.noteOpen
			if s.noteOpen {
				s.noteMotion = g.Now
				g.Execute(key.FocusCmd{Tag: &s.noteEditor})
			} else {
				s.flushReaderNote()
			}
		}), gap(8), s.button("deepen:"+id, "Aprofundar com IA", "spark", quietButton, func() {
			s.studyFromCard(l, i, "/aprofundar")
		}))
	}
}

func (s *studio) layoutNoteEditor(g ctx) dim {
	return enter(g, s.noteMotion, 0, 5, func(g ctx) dim {
		return glass(g, 13, wide(inset(12, func(g ctx) dim {
			status, tint := "Salvamento automático neste dispositivo.", subtle
			if !s.noteDue.IsZero() {
				status = "Salvando…"
			} else if s.localWarning != "" {
				status, tint = s.localWarning, bad
			} else if s.noteEditor.Len() > 0 {
				status = "Nota salva neste dispositivo."
			}
			return column(g, wide(func(g ctx) dim {
				border := line
				if g.Source.Focused(&s.noteEditor) {
					border = accent
				}
				return box(g, canvas, border, 9, inset(11, func(g ctx) dim {
					g.Constraints.Max.Y = min(g.Constraints.Max.Y, g.Dp(144))
					g.Constraints.Min.Y = min(g.Constraints.Max.Y, g.Dp(76))
					e := material.Editor(s.theme, &s.noteEditor, "Registre uma dúvida, um exemplo ou sua explicação.")
					e.TextSize, e.HintColor = 15, subtle
					semantic.LabelOp("Nota deste card").Add(g.Ops)
					return e.Layout(g)
				}))
			}), gap(7), s.label(status, 12, tint, font.Normal))
		})))
	})
}

func (s *studio) notesView() []view {
	items := []view{gap(24), s.title("Minhas notas", 34), gap(14)}
	for _, lessons := range [][]Lesson{s.lib.Generated, s.samples} {
		for _, l := range lessons {
			p := s.lib.Progress[l.ID]
			if p == nil {
				continue
			}
			links := s.noteLinks(l, p, "index")
			if len(links) != 0 {
				items = append(items, s.label(l.Title, 19, ink, font.SemiBold), gap(8))
				items = append(items, links...)
				items = append(items, gap(12))
			}
		}
	}
	if len(items) == 3 {
		items = append(items, s.label("Abra um card e escolha Escrever nota para guardar suas ideias aqui.", 15, muted, font.Normal),
			gap(14), func(g ctx) dim {
				return s.button("notes-study", "Ir ao estúdio", "right", outlineButton, func() { s.goTo("home") })(g)
			})
	}
	return append(items, gap(24))
}

func (s *studio) noteLinks(l Lesson, p *Progress, scope string) []view {
	var items []view
	for i, c := range l.Cards {
		value := notePreview(p.Notes[i])
		if value == "" {
			continue
		}
		id := fmt.Sprintf("note-%s:%s:%d", scope, l.ID, i)
		items = append(items, func(g ctx) dim {
			return s.control(g, id, "Abrir nota de "+c.Title, outlineButton, func() {
				if s.page != "reader" {
					s.open(l)
				}
				s.jump(i)
				s.noteOpen, s.noteReveal, s.noteMotion = true, true, g.Now
			}, wide(inset(11, func(g ctx) dim {
				return column(g, s.label(c.Title, 15, ink, font.SemiBold), gap(3), s.label(value, 14, muted, font.Normal))
			})))
		}, gap(7))
	}
	return items
}
