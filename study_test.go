package main

import (
	"path/filepath"
	"testing"
)

func TestLibraryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "OpenLogos", "library.json")
	lib, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	samples := sampleLessons()
	for _, l := range samples {
		if err := validateLesson(l); err != nil {
			t.Fatalf("%s: %v", l.Title, err)
		}
	}
	if err := lib.add(samples[0]); err != nil {
		t.Fatal(err)
	}
	l := lib.Generated[0]
	p := lib.progress(l)
	p.Cursor = 3
	p.Visited[3] = true
	p.Answers[3] = 1
	lib.Provider, lib.Feedback = claude, deferred
	if err := saveLibrary(path, lib); err != nil {
		t.Fatal(err)
	}
	p.Reviewed = true
	if err := saveLibrary(path, lib); err != nil {
		t.Fatal(err)
	}
	saved, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := saved.progress(l)
	if saved.Provider != claude || saved.Feedback != deferred || len(saved.Generated) != 1 ||
		reopened.Cursor != 3 || !reopened.Visited[3] || reopened.Answers[3] != 1 || !reopened.Reviewed {
		t.Fatalf("library round trip lost state: %+v, %+v", saved, reopened)
	}
}
