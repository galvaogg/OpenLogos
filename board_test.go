package main

import (
	"math"
	"path/filepath"
	"testing"
)

func TestBoardArrangementAndPersistence(t *testing.T) {
	old := exampleBoard()
	old.Nodes[0].X, old.Nodes[0].Y = 40, 35
	proposed := old.clone()
	proposed.Nodes = append(proposed.Nodes, BoardNode{ID: "limites", Label: "Limites", Shape: "note"})
	proposed.Edges = append(proposed.Edges, BoardEdge{From: "pergunta", To: "limites"})
	arrangeBoard(&proposed, &old)
	if err := validateBoard(proposed); err != nil {
		t.Fatal(err)
	}
	layouts := measureBoardNodes(proposed.Nodes)
	added := len(old.Nodes)
	for i, node := range old.Nodes {
		next := proposed.Nodes[i]
		if next.X != node.X || next.Y != node.Y {
			t.Fatalf("moved existing node %q", node.ID)
		}
		if layouts[added].rect(boardNodePosition(proposed.Nodes[added])).overlaps(layouts[i].rect(boardNodePosition(next))) {
			t.Fatalf("new node overlaps %q", node.ID)
		}
	}
	path := filepath.Join(t.TempDir(), "library.json")
	var lib Library
	id := lib.addBoard(proposed)
	if err := saveLibrary(path, lib); err != nil {
		t.Fatal(err)
	}
	got, err := loadLibrary(path)
	if err != nil {
		t.Fatal(err)
	}
	b := got.board(id)
	if b == nil || len(b.Nodes) != len(proposed.Nodes) || b.Nodes[0].X != 40 {
		t.Fatalf("board data did not survive reload: %+v", b)
	}
}

func TestBoardRejectsInvalidProviderGraphs(t *testing.T) {
	for name, corrupt := range map[string]func(*Board){
		"dangling edge":   func(b *Board) { b.Edges[0].To = "missing" },
		"duplicate node":  func(b *Board) { b.Nodes[1].ID = b.Nodes[0].ID },
		"unbounded point": func(b *Board) { b.Nodes[0].X = math.Inf(1) },
		"unknown shape":   func(b *Board) { b.Nodes[0].Shape = "external" },
	} {
		t.Run(name, func(t *testing.T) {
			b := exampleBoard()
			corrupt(&b)
			if err := validateBoard(b); err == nil {
				t.Fatal("accepted invalid graph")
			}
		})
	}
}
