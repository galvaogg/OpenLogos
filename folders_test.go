package main

import (
	"path/filepath"
	"testing"
)

func TestStudyFoldersPreserveProgress(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.json")
	lib, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Folders) != 0 {
		t.Fatalf("unexpected folders in empty library: %v", lib.Folders)
	}
	lesson := sampleLessons()[0]
	lib.progress(lesson).Visited[0] = true
	if err := lib.add(lesson); err != nil {
		t.Fatal(err)
	}
	generated := lib.Generated[0]
	lib.progress(generated).Cursor = 2
	first, err := lib.nameFolder("", "  Metafísica  ")
	if err != nil || lib.folder(first).Name != "Metafísica" {
		t.Fatalf("create folder: %q, %v", first, err)
	}
	if _, err := lib.nameFolder("", "meTAfísiCA"); err == nil {
		t.Fatal("accepted duplicate name")
	}
	if _, err := lib.nameFolder("", "Nome\nCom quebra"); err == nil {
		t.Fatal("accepted line break in folder name")
	}
	second, err := lib.nameFolder("", "Lógica")
	if err != nil {
		t.Fatal(err)
	}
	for _, move := range []struct{ lesson, folder string }{
		{lesson.ID, first}, {generated.ID, first}, {generated.ID, second}, {generated.ID, second},
	} {
		if err := lib.moveLesson(move.lesson, move.folder); err != nil {
			t.Fatal(err)
		}
	}
	if err := lib.moveLesson(generated.ID, "missing"); err == nil {
		t.Fatal("accepted missing folder")
	}
	if renamed, err := lib.nameFolder(second, "Lógica clássica"); err != nil || renamed != second {
		t.Fatalf("rename changed identity: %q, %v", renamed, err)
	}
	if err := saveLibrary(path, lib); err != nil {
		t.Fatal(err)
	}
	reopened, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	sampleFolder, generatedFolder := reopened.folderFor(lesson.ID), reopened.folderFor(generated.ID)
	if sampleFolder == nil || sampleFolder.ID != first || len(sampleFolder.Lessons) != 1 ||
		generatedFolder == nil || generatedFolder.ID != second || len(generatedFolder.Lessons) != 1 || generatedFolder.Name != "Lógica clássica" {
		t.Fatalf("folder restore lost assignments: %+v", reopened.Folders)
	}
	if !reopened.progress(lesson).Visited[0] || reopened.progress(generated).Cursor != 2 {
		t.Fatal("moving lesson changed its progress")
	}
	if err := reopened.moveLesson(lesson.ID, ""); err != nil || reopened.folderFor(lesson.ID) != nil {
		t.Fatalf("return lesson to unfiled: %v", err)
	}
}

func TestStudyFoldersRejectConflictingAssignments(t *testing.T) {
	var lib Library
	if err := lib.restoreFolders([]StudyFolder{
		{ID: "first", Name: "Metafísica", Lessons: []string{"sample-navio-de-teseu"}},
		{ID: "second", Name: "Lógica", Lessons: []string{"sample-navio-de-teseu"}},
	}); err == nil || len(lib.Folders) != 0 {
		t.Fatalf("restored conflicting assignments: %+v, %v", lib.Folders, err)
	}
}
