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
			var steps []view
			for i := range l.Cards {
				if i != 0 {
					steps = append(steps, gap(3))
				}
				a := quietButton
				if i == idx {
					a = activeButton
				}
				steps = append(steps, s.button(fmt.Sprintf("mini-%d", i), fmt.Sprint(i+1), "", a, func() { s.jump(i) }))
			}
			return row(g, steps...)
		}, gap(8))
	}
	if idx == len(l.Cards) {
		items = append(items, s.review(l, p)...)
	} else {
		c := l.Cards[idx]
		items = append(items, s.card(l, p, c, idx))
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
	}, gap(11), rule, gap(11),
		s.button("export:"+l.ID, "Exportar Markdown", "folder", quietButton, func() { s.export(l) }))
	if s.notice != "" {
		items = append(items, s.label(s.notice, 13, muted, font.Normal))
	}
	return append(items, gap(24))
}

func (s *studio) card(l Lesson, p *Progress, c Card, i int) view {
	return func(g ctx) dim {
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
			if c.Question != "" {
				items = append(items, s.label(c.Question, 18, ink, font.SemiBold), gap(11))
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
						p.Answers[i] = j
						s.save()
					}), gap(6))
				}
				if show {
					heading, tint := "Isso mesmo.", good
					if chosen != c.Correct {
						heading, tint = "Veja por que a outra alternativa funciona.", bad
					}
					items = append(items, gap(5), s.label(heading, 14, tint, font.SemiBold), gap(4),
						s.label(c.Explanation, 15, muted, font.Normal))
				} else if answered {
					items = append(items, s.label("Resposta salva. A correção aparece no resumo.", 14, muted, font.Normal))
				}
			}
			return column(g, items...)
		})))
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
