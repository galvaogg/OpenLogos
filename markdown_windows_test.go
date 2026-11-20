package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestStudyMarkdownExportPreservesEdits(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Estudos")
	t.Setenv("OPENLOGOS_STUDIES_DIR", dir)
	if got, err := studiesDir(); err != nil || got != dir {
		t.Fatalf("studiesDir() = %q, %v", got, err)
	}
	lesson := sampleLessons()[0]
	first, err := exportLessonMarkdown(dir, lesson)
	if err != nil {
		t.Fatal(err)
	}
	markdown, err := readStudyMarkdown(dir, filepath.Base(first))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"# " + lesson.Title, lesson.Summary, lesson.Cards[0].Body, lesson.Cards[3].Question, "## Gabarito", lesson.Cards[3].Explanation, "## Bibliografia", lesson.Sources[0]} {
		if !strings.Contains(markdown, text) {
			t.Errorf("export missing %q", text)
		}
	}
	const myNotes = "Anotações do meu estudo."
	if err := os.WriteFile(first, []byte(myNotes), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := exportLessonMarkdown(dir, lesson)
	if err != nil || second == first || !strings.HasSuffix(second, "-2.md") {
		t.Fatalf("second export = %q, %v", second, err)
	}
	if got, err := readStudyMarkdown(dir, filepath.Base(first)); got != myNotes || err != nil {
		t.Fatalf("edited original = %q, %v", got, err)
	}
}

func TestStudyMarkdownReadBoundary(t *testing.T) {
	dir := t.TempDir()
	for name, contents := range map[string]string{
		"A.MD":       "\ufeff# Órbita",
		"B.md":       strings.Repeat("é", maxStudyMarkdown+1),
		"C.md":       "\xff",
		"D.md":       " \n\t",
		"ignore.txt": "ignorar",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "subpasta.md"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Run("symlink", func(t *testing.T) {
		if err := os.Symlink(filepath.Join(dir, "A.MD"), filepath.Join(dir, "link.md")); err != nil {
			t.Skipf("Windows não permitiu criar o link simbólico: %v", err)
		}
		if _, err := readStudyMarkdown(dir, "link.md"); err == nil {
			t.Fatal("symlink accepted")
		}
	})
	if got, err := listStudyMarkdown(dir); err != nil || !slices.Equal(got, []string{"A.MD", "B.md", "C.md", "D.md"}) {
		t.Fatalf("listStudyMarkdown() = %v, %v", got, err)
	}
	if got, err := readStudyMarkdown(dir, "A.MD"); err != nil || got != "# Órbita" {
		t.Fatalf("readStudyMarkdown() = %q, %v", got, err)
	}
	for _, name := range []string{"B.md", "C.md", "D.md", "subpasta.md", "../A.MD", `.\A.MD`, filepath.Join(dir, "A.MD"), "A.MD:hidden.md", "ignore.txt"} {
		if _, err := readStudyMarkdown(dir, name); err == nil {
			t.Errorf("readStudyMarkdown() accepted %q", name)
		}
	}
}
