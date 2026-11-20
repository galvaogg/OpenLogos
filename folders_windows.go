package main

import (
	"fmt"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/io/semantic"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type folderState struct {
	active, destination string
	moving              Lesson
	returnToReader      bool
	name                widget.Editor
	editing, focusName  bool
	err                 string
}

func (s *studio) foldersNav() view {
	a := quietButton
	if s.page == "folders" {
		a = activeButton
	}
	return wide(s.button("folders", "Pastas de estudo", "folder", a, func() {
		s.selectFolder("")
		s.goTo("folders")
	}))
}

func (s *studio) selectFolder(id string) {
	f := &s.foldersUI
	f.active, f.err = id, ""
	f.moving = Lesson{}
	f.editing = false
	s.resetBody = true
}

func (s *studio) foldersView() []view {
	f := &s.foldersUI
	if f.moving.ID != "" {
		return s.folderPicker()
	}
	items := []view{gap(25)}
	active := s.lib.folder(f.active)
	if active == nil {
		items = append(items, func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Flexed(1, s.title("Pastas de estudo", 36)), layout.Rigid(gap(10)),
				layout.Rigid(s.button("folder-create", "Nova pasta", "plus", primaryButton, func() { s.editFolderName("") })))
		})
	} else {
		items = append(items, func(g ctx) dim {
			return row(g, s.button("folders-back", "Todas as pastas", "left", quietButton, func() { s.selectFolder("") }))
		},
			gap(10), s.title(active.Name, 36), gap(13), func(g ctx) dim {
				return row(g, s.button("folder-study", "Novo estudo", "plus", primaryButton, func() {
					s.foldersUI.destination = active.ID
					s.goTo("home")
				}), gap(8), s.button("folder-rename", "Renomear", "", outlineButton, func() { s.editFolderName(active.Name) }))
			})
	}
	if f.editing {
		items = append(items, gap(13), s.folderNameForm)
	}
	if f.err != "" {
		items = append(items, gap(6), s.label(f.err, 14, bad, font.Normal))
	}
	items = append(items, gap(23))

	assigned := make(map[string]string)
	for _, folder := range s.lib.Folders {
		for _, id := range folder.Lessons {
			assigned[id] = folder.ID
		}
		if active == nil {
			items = append(items, s.folderRow("folder:"+folder.ID, folder.Name, lessonCount(len(folder.Lessons)), false, func() {
				s.selectFolder(folder.ID)
			}), gap(7))
		}
	}
	if active == nil {
		if len(s.lib.Folders) != 0 {
			items = append(items, gap(16))
		}
		items = append(items, s.label("Lições sem pasta", 20, ink, font.SemiBold), gap(10))
	}
	shown := 0
	for _, shelf := range []struct {
		lessons []Lesson
		sample  bool
	}{{s.lib.Generated, false}, {s.samples, true}} {
		for _, lesson := range shelf.lessons {
			if assigned[lesson.ID] != f.active {
				continue
			}
			shown++
			items = append(items, func(g ctx) dim {
				return (layout.Flex{Alignment: layout.Middle}).Layout(g,
					layout.Flexed(1, s.lessonTile(lesson, shelf.sample)), layout.Rigid(gap(6)),
					layout.Rigid(s.moveLessonButton(lesson)))
			}, gap(8))
		}
	}
	if shown == 0 {
		message := "Todas as lições estão em pastas."
		if active != nil {
			message = "Crie um estudo aqui ou mova uma lição da biblioteca."
		}
		items = append(items, s.label(message, 15, muted, font.Normal))
	}
	return append(items, gap(24))
}

func lessonCount(n int) string {
	if n == 1 {
		return "1 lição"
	}
	return fmt.Sprintf("%d lições", n)
}

func (s *studio) folderRow(id, name, description string, current bool, action func()) view {
	return func(g ctx) dim {
		a := outlineButton
		if current {
			a = activeButton
		}
		return s.control(g, id, name+" · "+description, a, action, wide(padding(10, 12, 10, func(g ctx) dim {
			return (layout.Flex{Alignment: layout.Middle}).Layout(g,
				layout.Rigid(func(g ctx) dim {
					return box(g, selected, clear, 8, inset(8, glyph("folder", accent)))
				}), layout.Rigid(gap(11)),
				layout.Flexed(1, func(g ctx) dim {
					return column(g, s.label(name, 15, ink, font.SemiBold), gap(2), s.label(description, 13, muted, font.Normal))
				}), layout.Rigid(gap(8)), layout.Rigid(glyph("right", subtle)))
		})))
	}
}

func (s *studio) editFolderName(value string) {
	f := &s.foldersUI
	f.editing, f.focusName, f.err = true, true, ""
	f.name.SingleLine, f.name.Submit, f.name.MaxLen = true, true, folderNameLimit
	f.name.SetText(value)
	f.name.SetCaret(0, f.name.Len())
}

func (s *studio) folderNameForm(g ctx) dim {
	f := &s.foldersUI
	if f.focusName {
		g.Execute(key.FocusCmd{Tag: &f.name})
		f.focusName = false
	}
	for {
		event, ok := f.name.Update(g)
		if !ok {
			break
		}
		if _, submit := event.(widget.SubmitEvent); submit {
			s.saveFolderName()
			g.Execute(op.InvalidateCmd{})
			if !f.editing {
				return dim{}
			}
		}
	}
	return column(g, func(g ctx) dim {
		border := line
		if g.Source.Focused(&f.name) {
			border = accent
		}
		return box(g, canvas, border, 9, wide(padding(10, 12, 10, func(g ctx) dim {
			e := material.Editor(s.theme, &f.name, "Nome da pasta")
			e.TextSize, e.HintColor, e.Color = 15, subtle, ink
			semantic.LabelOp("Nome da pasta").Add(g.Ops)
			return e.Layout(g)
		})))
	}, gap(7), func(g ctx) dim {
		label := "Salvar pasta"
		if f.moving.ID != "" {
			label = "Criar e mover"
		}
		return row(g, s.button("folder-save-name", label, "check", primaryButton, s.saveFolderName), gap(8),
			s.button("folder-cancel-name", "Cancelar", "", quietButton, func() { f.editing, f.err = false, "" }))
	})
}

func (s *studio) saveFolderName() {
	f := &s.foldersUI
	id := f.active
	if f.moving.ID != "" {
		id = ""
	}
	id, err := s.lib.nameFolder(id, f.name.Text())
	if err != nil {
		f.err = err.Error()
		return
	}
	if f.moving.ID != "" {
		if err := s.lib.moveLesson(f.moving.ID, id); err != nil {
			f.err = err.Error()
			return
		}
		s.finishMovingLesson()
	} else {
		s.selectFolder(id)
	}
	s.save()
}

func (s *studio) moveLessonButton(l Lesson) view {
	return s.button("move-lesson:"+l.ID, "Mover lição", "folder", quietButton, func() {
		f := &s.foldersUI
		f.moving, f.returnToReader, f.err = l, s.page == "reader", ""
		f.editing = false
		s.resetBody = true
		s.goTo("folders")
	})
}

func (s *studio) finishMovingLesson() {
	f := &s.foldersUI
	backToReader := f.returnToReader
	s.selectFolder(f.active)
	if backToReader {
		s.goTo("reader")
	}
}

func (s *studio) folderPicker() []view {
	f := &s.foldersUI
	currentID := ""
	if current := s.lib.folderFor(f.moving.ID); current != nil {
		currentID = current.ID
	}
	destination := func(id, name string) view {
		status := "Mover para cá"
		if id == currentID {
			status = "Local atual"
		}
		return s.folderRow("move-to:"+id, name, status, id == currentID, func() {
			if err := s.lib.moveLesson(f.moving.ID, id); err != nil {
				f.err = err.Error()
				return
			}
			if id != currentID {
				s.save()
			}
			s.finishMovingLesson()
		})
	}
	items := []view{gap(25), func(g ctx) dim {
		return row(g, s.button("move-back", "Voltar", "left", quietButton, s.finishMovingLesson))
	}, gap(10), s.title("Mover lição", 36), gap(8), s.label(f.moving.Title, 16, muted, font.Normal), gap(16),
		func(g ctx) dim {
			return row(g, s.button("move-create-folder", "Nova pasta", "plus", primaryButton, func() { s.editFolderName("") }))
		}}
	if f.editing {
		items = append(items, gap(13), s.folderNameForm)
	}
	if f.err != "" {
		items = append(items, gap(6))
		items = append(items, s.label(f.err, 14, bad, font.Normal))
	}
	if len(s.lib.Folders) != 0 {
		items = append(items, gap(17), destination("", "Lições sem pasta"), gap(7))
		for _, folder := range s.lib.Folders {
			items = append(items, destination(folder.ID, folder.Name), gap(7))
		}
	}
	return append(items, gap(24))
}

func (s *studio) newStudyFolderHint() view {
	folder := s.lib.folder(s.foldersUI.destination)
	if folder == nil {
		return func(ctx) dim { return dim{} }
	}
	return padding(0, 0, 9, s.button("clear-study-folder", "Salvar em "+folder.Name+" · Remover", "folder", activeButton, func() {
		s.foldersUI.destination = ""
	}))
}
