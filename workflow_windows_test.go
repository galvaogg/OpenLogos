package main

import (
	"image"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
)

func TestStudyWorkflowThroughGio(t *testing.T) {
	s := studio{
		window: new(app.Window), theme: studioTheme(), page: "home",
		path: filepath.Join(t.TempDir(), "library.json"), samples: sampleLessons(),
		lib:     Library{Provider: codex, Feedback: instant, Progress: make(map[string]*Progress)},
		results: make(chan result, 1), clicks: make(map[string]*studioButton),
		body: layout.List{Axis: layout.Vertical}, rail: layout.List{Axis: layout.Vertical},
		detail: layout.List{Axis: layout.Vertical},
	}
	s.editor.SingleLine, s.editor.Submit, s.editor.MaxLen = true, true, 240
	var router input.Router
	var ops op.Ops
	now := time.Now()
	frame := func() {
		now = now.Add(20 * time.Millisecond)
		ops.Reset()
		g := ctx{Now: now, Constraints: layout.Exact(image.Pt(1260, 810)), Ops: &ops, Source: router.Source()}
		s.tickReaderNote(g)
		s.collect()
		s.layout(g)
		router.Frame(&ops)
	}
	click := func(label string) {
		t.Helper()
		var labels []string
		for _, node := range router.AppendSemantics(nil) {
			if node.Desc.Label != "" {
				labels = append(labels, node.Desc.Label)
			}
			bounds := node.Desc.Bounds
			if !strings.HasPrefix(node.Desc.Label, label) || bounds.Empty() || node.Desc.Gestures&input.ClickGesture == 0 {
				continue
			}
			center := f32.Pt(float32(bounds.Min.X+bounds.Dx()/2), float32(bounds.Min.Y+bounds.Dy()/2))
			router.Queue(pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: center})
			frame()
			router.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: center})
			frame()
			frame()
			return
		}
		t.Fatalf("missing control %q in %v", label, labels)
	}
	editText := func(start, end int, text string) {
		router.Queue(key.EditEvent{Range: key.Range{Start: start, End: end}, Text: text})
		frame()
		frame()
	}

	frame()
	click("Pastas de estudo")
	click("Nova pasta")
	editText(0, 0, "Filosofia")
	click("Salvar pasta")
	if len(s.lib.Folders) != 1 || s.lib.Folders[0].Name != "Filosofia" {
		t.Fatalf("folder not saved: %+v", s.lib.Folders)
	}
	folder := s.lib.Folders[0].ID
	click("Novo estudo")
	click("Comandos de estudo")
	click("/praticar ·")
	editText(0, s.editor.Len(), "/pra")
	click("/praticar ·")
	editText(s.editor.Len(), s.editor.Len(), "identidade")
	draft := s.currentDraft()
	if draft.folder != folder || draft.prompt != "/praticar identidade" {
		t.Fatalf("wrong composer draft: %+v", draft)
	}

	s.stop = func() {}
	s.results <- result{lesson: s.samples[0], draft: draft}
	frame()
	frame()
	if s.page != "reader" || len(s.chat) != 1 {
		t.Fatal("generated lesson did not open")
	}
	if assigned := s.lib.folderFor(s.lesson.ID); assigned == nil || assigned.ID != folder {
		t.Fatal("generated lesson did not open in the selected folder")
	}
	click("Mover lição")
	click("Lições sem pasta")
	if s.page != "reader" || s.lib.folderFor(s.lesson.ID) != nil {
		t.Fatal("moving the lesson did not return to the reader")
	}
	click("Escolha o critério")
	click("Ver dica")
	if !s.hintOpen || s.cardIndex != 3 {
		t.Fatal("exercise hint did not open")
	}
	click("A: ")
	if answer, ok := s.lib.progress(*s.lesson).Answers[3]; !ok || answer != 0 {
		t.Fatal("exercise did not record the answer")
	}
	click("Uma tábua por vez")
	click("Pensar além")
	if !s.hintOpen {
		t.Fatal("reflection did not open")
	}
	click("Escrever nota")
	const note = "Testar um exemplo de identidade."
	editText(0, 0, note)
	click("Minhas notas")

	saved, err := loadLibrary(s.path)
	if err != nil || saved.Progress[s.lesson.ID].Notes[0] != note {
		t.Fatalf("note was not saved on navigation: %v", err)
	}
	click("Abrir nota de")
	if s.page != "reader" || !s.noteOpen || s.cardIndex != 0 {
		t.Fatal("notes index did not reopen the original card")
	}
	click("Aprofundar com IA")
	if s.page != "home" || !strings.HasPrefix(s.editor.Text(), "/aprofundar ") ||
		!strings.Contains(s.cardSource.text, note) {
		t.Fatal("follow-up did not attach the original study and note")
	}
}
