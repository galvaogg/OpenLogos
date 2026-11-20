package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBoardExportPreservesNodesAndBindings(t *testing.T) {
	board := Board{
		ID: "board-export", Kind: "flowchart", Title: "Ciclo da água",
		Nodes: []BoardNode{
			{ID: "vapor", Shape: "topic", Label: "Evaporação", Detail: "A água passa para o estado gasoso.", X: 345, Y: 70},
			{ID: "cloud", Shape: "note", Label: "Condensação", X: -35, Y: 230},
		},
		Edges: []BoardEdge{{From: "vapor", To: "cloud", Label: "forma nuvens"}},
	}
	data, err := boardExcalidraw(board)
	if err != nil {
		t.Fatal(err)
	}
	var scene struct {
		Type     string
		Version  int
		Elements []struct {
			ID, Type, Text, OriginalText, ContainerID, EndArrowhead string
			Points                                                  [][2]float64
			StartBinding, EndBinding                                struct{ ElementID, Mode string }
			BoundElements                                           []excalidrawBoundElement
		}
	}
	if err := json.Unmarshal(data, &scene); err != nil {
		t.Fatal(err)
	}
	if scene.Type != "excalidraw" || scene.Version != 2 || len(scene.Elements) != 7 {
		t.Fatalf("unexpected Excalidraw scene: %+v", scene)
	}
	arrow, arrowLabel, node, nodeLabel := scene.Elements[0], scene.Elements[1], scene.Elements[2], scene.Elements[3]
	if arrow.Type != "arrow" || arrow.EndArrowhead != "triangle" || arrow.StartBinding.ElementID != "node-0" ||
		arrow.EndBinding.ElementID != "node-1" || arrow.StartBinding.Mode != "orbit" || len(arrow.Points) != 2 ||
		arrow.Points[0] != ([2]float64{}) || arrow.Points[1][0] >= 0 {
		t.Fatalf("wrong arrow or endpoints: %+v", arrow)
	}
	if arrowLabel.ContainerID != arrow.ID || arrowLabel.Text != "forma nuvens" ||
		len(arrow.BoundElements) != 1 || arrow.BoundElements[0].ID != arrowLabel.ID {
		t.Fatalf("missing arrow label binding: %+v", arrowLabel)
	}
	if nodeLabel.ContainerID != node.ID || nodeLabel.OriginalText != "Evaporação\nA água passa para o estado gasoso." ||
		len(node.BoundElements) != 2 || node.BoundElements[1].ID != arrow.ID {
		t.Fatalf("missing editable text or reciprocal arrow binding: %+v", node)
	}
}

func TestBoardExportKeepsEarlierCopies(t *testing.T) {
	dir := t.TempDir()
	board := Board{
		ID: "board-export", Kind: "mindmap", Title: "Água: caminho",
		Nodes: []BoardNode{
			{ID: "water", Shape: "topic", Label: "Água"},
			{ID: "rain", Shape: "note", Label: "Chuva"},
		},
		Edges: []BoardEdge{{From: "water", To: "rain"}},
	}
	first, err := exportBoardExcalidraw(dir, board)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("edited"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := exportBoardExcalidraw(dir, board)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(first) != "Quadro-Água-caminho.excalidraw" ||
		filepath.Base(second) != "Quadro-Água-caminho-2.excalidraw" || filepath.Dir(second) != dir {
		t.Fatalf("unsafe or duplicated export names: %q, %q", first, second)
	}
	earlier, err := os.ReadFile(first)
	if err != nil || string(earlier) != "edited" {
		t.Fatalf("earlier edits overwritten: %q, %v", earlier, err)
	}
}
