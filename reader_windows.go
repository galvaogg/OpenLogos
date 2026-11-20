package main

import (
	"fmt"
	"image"

	"gioui.org/font"
	"gioui.org/layout"
)

func (s *studio) reader(wideRail bool) []view {
	l := *s.lesson
	p := s.lib.progress(l)
	idx := p.Cursor
	items := []view{gap(22), s.label(l.Title, 15, muted, font.SemiBold), gap(12)}
	if !wideRail {
		items = append(items, func(g ctx) dim {
			perRow := max(1, g.Constraints.Max.X/g.Dp(44))
			var rows []view
			for start := 0; start < len(l.Cards); start += perRow {
				var steps []view
				for i := start; i < min(start+perRow, len(l.Cards)); i++ {
					if i != start {
						steps = append(steps, gap(3))
					}
					a := quietButton
					if i == idx {
						a = activeButton
					}
					steps = append(steps, s.button(fmt.Sprintf("mini:%s:%d", l.ID, i), fmt.Sprint(i+1), "", a, func() { s.jump(i) }))
				}
				rows = append(rows, func(g ctx) dim { return row(g, steps...) })
			}
			return column(g, rows...)
		}, gap(8))
	}
	if idx == len(l.Cards) {
		items = append(items, s.review(l, p)...)
	} else {
		c := l.Cards[idx]
		if s.noteReveal {
			s.body.ScrollTo(len(items) + 2)
			s.noteReveal = false
		}
		items = append(items, s.card(l, p, c, idx), gap(10), s.cardActions(l, p, idx), func(g ctx) dim {
			if !s.noteOpen {
				return dim{}
			}
			return column(g, gap(7), wide(s.layoutNoteEditor))
		})
	}
	items = append(items, gap(12), func(g ctx) dim {
		var prev func()
		if idx > 0 {
			prev = func() { s.jump(idx - 1) }
		}
		right := s.button("reader-next", "Continuar", "right", primaryButton, func() { s.jump(idx + 1) })
		if idx == len(l.Cards)-1 {
			right = s.button("reader-review", "Ver resumo", "right", primaryButton, func() { s.jump(idx + 1) })
		} else if idx == len(l.Cards) {
			right = s.button("reader-done", "Voltar ao estúdio", "left", outlineButton, func() { s.goTo("home") })
		}
		return (layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}).Layout(g,
			layout.Rigid(s.button("reader-prev", "Anterior", "left", quietButton, prev)),
			layout.Rigid(right),
		)
	}, gap(11), rule, gap(11), func(g ctx) dim {
		return row(g, s.button("export:"+l.ID, "Exportar Markdown", "folder", quietButton, func() { s.export(l) }),
			gap(8), s.moveLessonButton(l))
	})
	if s.notice != "" {
		items = append(items, s.label(s.notice, 13, muted, font.Normal))
	}
	return append(items, gap(24))
}

func (s *studio) card(l Lesson, p *Progress, c Card, i int) view {
	return func(g ctx) dim {
		return enter(g, s.cardMotion, s.cardShift, 0, func(g ctx) dim {
			return box(g, panel, line, 15, wide(padding(21, 22, 22, func(g ctx) dim {
				tag := "Leitura"
				if c.Question != "" {
					tag = "Exercício"
				}
				items := []view{s.label(fmt.Sprintf("%s  ·  %d / %d", tag, i+1, len(l.Cards)), 13, accent, font.SemiBold), gap(8),
					s.title(c.Title, 32), gap(12)}
				if c.Body != "" {
					items = append(items, func(g ctx) dim { return s.paragraphs(g, c.Body) }, gap(5))
				}
				if c.Question == "" {
					items = append(items, gap(4), s.cardHint(l, c, i))
				}
				if c.Question != "" {
					items = append(items, s.label(c.Question, 18, ink, font.SemiBold), gap(5), s.cardHint(l, c, i), gap(9))
					chosen, answered := p.Answers[i]
					show := answered && (s.lib.Feedback == instant || p.Reviewed)
					for j, choice := range c.Options {
						a := outlineButton
						if answered && chosen == j {
							a = activeButton
							if show && chosen != c.Correct {
								a = appearance{hex(0x382829), hex(0x724144), bad}
							}
						}
						if show && j == c.Correct {
							a = appearance{hex(0x22352d), hex(0x3d6653), good}
						}
						items = append(items, s.option(l.ID, i, j, choice, a, func() {
							if picked, ok := p.Answers[i]; ok && picked == j {
								return
							}
							p.Answers[i] = j
							s.feedbackMotion = g.Now
							s.save()
						}), gap(6))
					}
					if show {
						heading, tint := "Isso mesmo.", good
						if chosen != c.Correct {
							heading, tint = "Veja por que a outra alternativa funciona.", bad
						}
						items = append(items, gap(5), func(g ctx) dim {
							return enter(g, s.feedbackMotion, 0, 5, func(g ctx) dim {
								return column(g, s.label(heading, 14, tint, font.SemiBold), gap(4), s.label(c.Explanation, 15, muted, font.Normal))
							})
						})
					} else if answered {
						items = append(items, s.label("Resposta salva. A correção aparece no resumo.", 14, muted, font.Normal))
					}
				}
				return column(g, items...)
			})))
		})
	}
}

func (s *studio) cardHint(l Lesson, c Card, i int) view {
	hint, label, hide := c.Hint, "Ver dica", "Ocultar dica"
	if c.Question == "" {
		label, hide = "Pensar além", "Ocultar reflexão"
		if hint == "" {
			hint = "Como você explicaria a ideia deste card em duas frases? Escreva um exemplo próprio na sua nota."
		}
	} else if hint == "" {
		hint = "Identifique a ideia que a pergunta pede e volte ao card de leitura que a apresentou. Compare as alternativas com sua própria explicação."
	}
	id := fmt.Sprintf("hint:%s:%d", l.ID, i)
	return func(g ctx) dim {
		name, a := label, quietButton
		if s.hintOpen {
			name, a = hide, activeButton
		}
		toggle := s.button(id, name, "spark", a, func() {
			s.hintOpen = !s.hintOpen
			s.hintMotion = g.Now
		})
		if !s.hintOpen {
			return toggle(g)
		}
		return column(g, toggle, gap(7), wide(func(g ctx) dim {
			return enter(g, s.hintMotion, 0, 5, func(g ctx) dim {
				return box(g, selected, clear, 9, inset(11, s.label(hint, 15, ink, font.Normal)))
			})
		}))
	}
}

func (s *studio) option(id string, index, choice int, text string, a appearance, action func()) view {
	letter := string(rune('A' + choice))
	return func(g ctx) dim {
		return s.control(g, fmt.Sprintf("answer:%s:%d:%d", id, index, choice), letter+": "+text, a, action,
			wide(padding(9, 12, 9, func(g ctx) dim {
				return (layout.Flex{Alignment: layout.Middle}).Layout(g,
					layout.Rigid(s.label(letter, 14, a.fg, font.Bold)), layout.Rigid(gap(12)),
					layout.Flexed(1, s.label(text, 15, a.fg, font.Normal)),
				)
			})))
	}
}

func (s *studio) review(l Lesson, p *Progress) []view {
	items := []view{s.title("Resumo da lição", 36), gap(9), s.label(l.Summary, 17, muted, font.Normal),
		gap(17), s.label("Exercícios", 19, ink, font.SemiBold), gap(8)}
	for i, c := range l.Cards {
		if c.Question == "" {
			continue
		}
		answer, ok := p.Answers[i]
		name := "Em aberto"
		if ok && answer == c.Correct {
			name = "Correta"
		} else if ok {
			name = "Para revisar"
		}
		items = append(items, wide(s.button(fmt.Sprintf("review-item-%d", i), c.Title+"  ·  "+name, "right", outlineButton, func() { s.jump(i) })), gap(6))
	}
	if links := s.noteLinks(l, p, "review"); len(links) != 0 {
		items = append(items, gap(15), s.label("Notas da lição", 19, ink, font.SemiBold), gap(8))
		items = append(items, links...)
	}
	items = append(items, gap(15), s.label("Bibliografia", 19, ink, font.SemiBold), gap(8))
	for _, source := range l.Sources {
		items = append(items, s.label(source, 15, muted, font.Normal), gap(7))
	}
	if len(l.Sources) == 0 {
		items = append(items, s.label("Esta lição veio sem referências bibliográficas.", 15, muted, font.Normal))
	}
	return items
}

func (s *studio) sequence(g ctx) dim {
	l := *s.lesson
	p := s.lib.progress(l)
	n := 0
	for i := range l.Cards {
		if p.Visited[i] {
			n++
		}
	}
	items := []view{s.label("Nesta lição", 16, ink, font.SemiBold), gap(6),
		s.label(fmt.Sprintf("%d de %d cards visitados", n, len(l.Cards)), 13, muted, font.Normal), gap(11),
		func(g ctx) dim {
			width := g.Constraints.Max.X
			cell := width / len(l.Cards)
			for i := range l.Cards {
				tint := line
				if p.Visited[i] {
					tint = accent
				}
				round(g, image.Rect(i*cell, 0, (i+1)*cell-g.Dp(3), g.Dp(4)), 2, tint)
			}
			return dim{Size: image.Pt(width, g.Dp(4))}
		}, gap(12)}
	for i, c := range l.Cards {
		a, icon := quietButton, "book"
		if c.Question != "" {
			icon = "spark"
		}
		if p.Visited[i] {
			icon = "check"
		}
		if i == p.Cursor {
			a = activeButton
		}
		items = append(items, wide(s.button(fmt.Sprintf("step-%d", i), c.Title, icon, a, func() { s.jump(i) })), gap(2))
	}
	a := quietButton
	if p.Cursor == len(l.Cards) {
		a = activeButton
	}
	items = append(items, gap(9), rule, gap(8), wide(s.button("sequence-review", "Resumo e bibliografia", "right", a, func() {
		s.jump(len(l.Cards))
	})))
	return column(g, items...)
}
