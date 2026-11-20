package main

import (
	"fmt"
	"image"
	"strings"

	"gioui.org/font"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func (s *studio) layout(g ctx) dim {
	wideRail := g.Constraints.Max.X >= g.Dp(1110)
	return (layout.Flex{}).Layout(g,
		layout.Rigid(fixedWidth(224, s.sidebar)),
		layout.Flexed(1, func(g ctx) dim {
			return s.workspace(g, wideRail)
		}),
	)
}

func (s *studio) sidebar(g ctx) dim {
	size := g.Constraints.Max
	paint.FillShape(g.Ops, side, clip.Rect{Max: size}.Op())
	paint.FillShape(g.Ops, line, clip.Rect{Min: image.Pt(size.X-1, 0), Max: size}.Op())
	return (layout.Flex{Axis: layout.Vertical}).Layout(g,
		layout.Rigid(padding(18, 17, 16, wide(func(g ctx) dim {
			return row(g, func(g ctx) dim {
				return box(g, night, line, 10, func(g ctx) dim { return mascot(g, 36, 36, false, false) })
			}, gap(9), s.label("OpenLogos", 19, ink, font.Bold))
		}))),
		layout.Flexed(1, padding(0, 13, 0, func(g ctx) dim {
			return s.rail.Layout(g, 1+len(s.lib.Generated)+len(s.samples), func(g ctx, i int) dim {
				if i == 0 {
					a := quietButton
					if s.page == "home" {
						a = activeButton
					}
					return column(g, wide(s.button("home", "Novo estudo", "plus", a, func() { s.goTo("home") })),
						gap(24), padding(0, 11, 7, s.label("Biblioteca", 13, subtle, font.SemiBold)))
				}
				items := s.lib.Generated
				isSample := i > len(items)
				var l Lesson
				if isSample {
					l = s.samples[i-len(items)-1]
				} else {
					l = items[i-1]
				}
				a := quietButton
				if s.page == "reader" && s.lesson.ID == l.ID {
					a = activeButton
				}
				return padding(0, 0, 4, wide(s.button("side:"+l.ID, l.Title, "book", a, func() { s.open(l) })))(g)
			})
		})),
		layout.Rigid(padding(12, 13, 15, wide(func(g ctx) dim {
			a := quietButton
			if s.page == "files" {
				a = activeButton
			}
			return column(g, rule, gap(10), wide(s.button("files", "Arquivos de estudo", "folder", a, func() {
				s.refreshStudies()
				s.goTo("files")
			})), padding(4, 11, 0, s.label("Biblioteca neste dispositivo", 12, subtle, font.Normal)))
		}))),
	)
}

func (s *studio) workspace(g ctx, wideRail bool) dim {
	title := "Estúdio"
	if s.page == "reader" {
		title = "Biblioteca  /  " + s.lesson.Topic
	} else if s.page == "files" {
		title = "Estúdio  /  Arquivos"
	}
	return (layout.Flex{Axis: layout.Vertical}).Layout(g,
		layout.Rigid(wide(func(g ctx) dim {
			return column(g, padding(10, 25, 10, func(g ctx) dim {
				return (layout.Flex{Alignment: layout.Middle}).Layout(g,
					layout.Flexed(1, s.label(title, 14, muted, font.SemiBold)),
					layout.Rigid(s.button("provider-top", s.lib.Provider, "terminal", outlineButton, func() {
						if s.page == "settings" {
							s.goTo("home")
						} else {
							s.goTo("settings")
						}
					})),
				)
			}), rule)
		})),
		layout.Flexed(1, func(g ctx) dim {
			var children []layout.FlexChild
			children = append(children, layout.Flexed(1, func(g ctx) dim {
				var content []view
				switch s.page {
				case "reader":
					content = s.reader(wideRail)
				case "settings":
					content = []view{s.settings}
				case "files":
					content = s.files()
				default:
					content = s.home()
				}
				if s.localWarning != "" {
					content = append(content, s.label(s.localWarning, 14, bad, font.Normal))
				}
				return padding(0, 25, 0, func(g ctx) dim {
					return s.body.Layout(g, len(content), func(g ctx, i int) dim { return wide(content[i])(g) })
				})(g)
			}))
			if wideRail && (s.page == "home" || s.page == "reader") {
				children = append(children, layout.Rigid(fixedWidth(258, func(g ctx) dim {
					paint.FillShape(g.Ops, line, clip.Rect{Max: image.Pt(1, g.Constraints.Max.Y)}.Op())
					return padding(21, 19, 16, func(g ctx) dim {
						if s.page == "reader" {
							return s.detail.Layout(g, 1, func(g ctx, _ int) dim { return s.sequence(g) })
						}
						return s.session(g, "rail")
					})(g)
				})))
			}
			return (layout.Flex{}).Layout(g, children...)
		}),
	)
}

func (s *studio) home() []view {
	items := []view{gap(26), s.composer, gap(27)}
	if s.generationError != "" {
		items = append(items, s.label(s.generationError, 14, bad, font.SemiBold), gap(18))
	}
	if len(s.lib.Generated) > 0 {
		items = append(items, s.label("Seus estudos", 20, ink, font.SemiBold), gap(10))
		for _, l := range s.lib.Generated {
			items = append(items, s.lessonTile(l, false), gap(8))
		}
		items = append(items, gap(18))
	}
	items = append(items, s.label("Experimente uma lição", 20, ink, font.SemiBold),
		gap(5), s.label("Três estudos de metafísica disponíveis offline.", 14, muted, font.Normal), gap(13))
	for _, l := range s.samples {
		items = append(items, s.lessonTile(l, true), gap(8))
	}
	return append(items, gap(24))
}

func (s *studio) composer(g ctx) dim {
	working := s.stop != nil
	focused := g.Source.Focused(&s.editor)
	for {
		event, ok := s.editor.Update(g)
		if !ok {
			break
		}
		if _, ok := event.(widget.SubmitEvent); ok {
			s.generate()
		}
	}
	return box(g, hex(0x1a1d29), hex(0x353b52), 16, wide(padding(22, 22, 20, func(g ctx) dim {
		return column(g, func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Flexed(1, func(g ctx) dim {
					return column(g, s.title("O que você quer\naprender hoje?", 42), gap(11),
						s.label("Escolha um tema. A IA monta uma lição com cards e exercícios.", 15, muted, font.Normal))
				}),
				layout.Rigid(func(g ctx) dim { return mascot(g, 112, 126, focused, working) }),
			)
		}, gap(16), s.promptField, gap(11), func(g ctx) dim {
			if working {
				topic := s.activeTopic
				return (layout.Flex{Alignment: layout.Middle}).Layout(g,
					layout.Flexed(1, s.label("Criando a lição · "+topic, 14, accent, font.SemiBold)),
					layout.Rigid(s.button("stop", "Cancelar", "", outlineButton, func() {
						if s.stop != nil {
							s.stop()
						}
					})),
				)
			}
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Flexed(1, func(g ctx) dim {
					if s.note != "" {
						return s.button("clear-note", s.note+" · Remover", "folder", activeButton, func() { s.note = "" })(g)
					}
					return s.button("attach-note", "Usar Markdown", "folder", quietButton, func() {
						s.refreshStudies()
						s.goTo("files")
					})(g)
				}),
				layout.Rigid(gap(8)),
				layout.Rigid(s.button("generate", "Criar lição", "spark", primaryButton, s.generate)),
			)
		})
	})))
}

func (s *studio) promptField(g ctx) dim {
	border := line
	if g.Source.Focused(&s.editor) {
		border = accent
	}
	return box(g, canvas, border, 9, wide(padding(13, 14, 13, func(g ctx) dim {
		e := material.Editor(s.theme, &s.editor, "Ex.: por que o tempo passa mais devagar perto de um buraco negro?")
		e.TextSize, e.HintColor, e.Color = 15, subtle, ink
		semantic.LabelOp("Assunto da lição").Add(g.Ops)
		return e.Layout(g)
	})))
}

func (s *studio) lessonTile(l Lesson, isSample bool) view {
	return func(g ctx) dim {
		info := fmt.Sprintf("%s  ·  %d cards", l.Topic, len(l.Cards))
		if p := s.lib.Progress[l.ID]; p != nil {
			if p.Reviewed {
				info += "  ·  Revisada"
			} else {
				info += fmt.Sprintf("  ·  Card %d", p.Cursor+1)
			}
		} else if !isSample {
			info += "  ·  Gerada com IA"
		}
		return s.control(g, "tile:"+l.ID, l.Title, outlineButton, func() { s.open(l) }, wide(padding(9, 11, 9, func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Rigid(func(g ctx) dim {
					return box(g, selected, clear, 8, inset(8, glyph("book", accent)))
				}),
				layout.Rigid(gap(11)),
				layout.Flexed(1, func(g ctx) dim {
					return column(g, s.label(l.Title, 15, ink, font.SemiBold), gap(2), s.label(info, 12, muted, font.Normal))
				}),
				layout.Rigid(gap(8)), layout.Rigid(glyph("right", subtle)),
			)
		})))
	}
}

func (s *studio) session(g ctx, key string) dim {
	c := s.connections[s.lib.Provider]
	return column(g, s.label("Agente de estudo", 16, ink, font.SemiBold), gap(11),
		wide(func(g ctx) dim {
			return box(g, panel, line, 11, wide(padding(11, 12, 12, func(g ctx) dim {
				return column(g, func(g ctx) dim {
					return row(g, glyph("terminal", accent), gap(8),
						s.label(s.lib.Provider, 15, ink, font.SemiBold))
				}, gap(6),
					s.label(c.status, 13, muted, font.Normal))
			})))
		}), gap(17), s.label("Feedback dos exercícios", 14, ink, font.SemiBold), gap(6),
		s.feedbackButton(key, instant, "Após cada resposta"), gap(3),
		s.feedbackButton(key, deferred, "No resumo da lição"), gap(17), rule, gap(14),
		s.button("files-"+key, "Pasta Estudos", "folder", quietButton, s.openStudies),
		s.label(s.studyError, 13, bad, font.Normal))
}

func (s *studio) feedbackButton(key, value, label string) view {
	a := quietButton
	icon := ""
	if s.lib.Feedback == value {
		a, icon = activeButton, "check"
	}
	return wide(s.button("feedback-"+key+value, label, icon, a, func() {
		s.lib.Feedback = value
		s.save()
	}))
}

func (s *studio) settings(g ctx) dim {
	return fixedWidth(530, func(g ctx) dim {
		return column(g, gap(25), s.title("Agente e preferências", 34), gap(8),
			s.label("Use a sessão já autenticada no terminal.", 15, muted, font.Normal), gap(18),
			func(g ctx) dim { return row(g, s.providerButton(codex), gap(8), s.providerButton(claude)) }, gap(10),
			s.button("check-provider", "Verificar sessão", "terminal", outlineButton, s.checkProvider), gap(22),
			func(g ctx) dim { return s.session(g, "settings") }, gap(24))
	})(g)
}

func (s *studio) providerButton(name string) view {
	a := outlineButton
	if s.lib.Provider == name {
		a = activeButton
	}
	return s.button("provider-"+name, name, "", a, func() {
		s.lib.Provider = name
		s.save()
		s.checkProvider()
	})
}

func (s *studio) files() []view {
	items := []view{gap(25), s.title("Arquivos de estudo", 34), gap(9),
		s.label("Selecione um arquivo .md da pasta Estudos para usar como contexto. Ele será enviado ao agente quando você criar a lição.", 15, muted, font.Normal),
		gap(10), s.label(s.studiesDir, 13, subtle, font.Normal), gap(15),
		func(g ctx) dim {
			return row(g, s.button("open-folder", "Abrir pasta", "folder", outlineButton, s.openStudies),
				gap(8), s.button("refresh-files", "Atualizar", "", quietButton, s.refreshStudies))
		}, gap(22)}
	if s.studyError != "" {
		items = append(items, s.label(s.studyError, 14, bad, font.Normal))
	} else if len(s.studyFiles) == 0 {
		items = append(items, s.label("Adicione um .md à pasta ou exporte uma lição da biblioteca.", 14, muted, font.Normal))
	} else {
		for _, file := range s.studyFiles {
			items = append(items, wide(s.button("file:"+file, file, "folder", outlineButton, func() {
				s.note, s.generationError = file, ""
				s.goTo("home")
			})), gap(7))
		}
	}
	return append(items, gap(20))
}

func (s *studio) paragraphs(g ctx, value string) dim {
	parts := strings.Split(value, "\n\n")
	var items []view
	for i, part := range parts {
		if i != 0 {
			items = append(items, gap(10))
		}
		items = append(items, s.label(part, 17, muted, font.Normal))
	}
	return column(g, items...)
}
