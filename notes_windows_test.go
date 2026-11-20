package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gioui.org/op"
)

func TestReaderNoteSavedOnCardNavigation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.json")
	lib, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	l := sampleLessons()[0]
	s := studio{path: path, lib: lib, lesson: &l, page: "reader"}
	p := s.lib.progress(l)
	s.loadReaderCard(l, p)
	s.noteOpen = true
	const note = "Critério: história de uso.\nTratar as duas hipóteses."
	s.noteEditor.SetText(note)
	s.tickReaderNote(ctx{Now: time.Now(), Ops: &op.Ops{}})
	if p.Notes[0] != note || s.noteDue.IsZero() {
		t.Fatal("editor did not record the note and schedule a save")
	}
	p.Cursor = 1
	s.loadReaderCard(l, p)
	if s.noteEditor.Len() != 0 || s.noteOpen || !s.noteDue.IsZero() {
		t.Fatal("previous card's note leaked into the next card")
	}
	stored, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := stored.progress(l).Notes[0]; got != note {
		t.Fatalf("reopened note = %q, want %q", got, note)
	}
	if !p.setCardNote(0, len(l.Cards), "\n ") || len(p.Notes) != 0 {
		t.Fatal("clearing the editor should remove the note")
	}
	if p.setCardNote(len(l.Cards), len(l.Cards), note) {
		t.Fatal("review screen must not receive a card note")
	}
}

func TestNotePreviewTruncatesRunes(t *testing.T) {
	if got := notePreview("  Uma\n\t dúvida  "); got != "Uma dúvida" {
		t.Fatalf("unexpected excerpt %q", got)
	}
	short := strings.Repeat("ç", 160)
	if got := notePreview(short + "é"); got != short+"…" {
		t.Fatalf("unexpected truncation %q", got)
	}
}
