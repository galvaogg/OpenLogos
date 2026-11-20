package main

import (
	"encoding/json"
	"image"
	"os"
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

func TestManualBoardWorkflowThroughGio(t *testing.T) {
	exportDir := t.TempDir()
	t.Setenv("OPENLOGOS_STUDIES_DIR", exportDir)
	s := studio{
		window: new(app.Window), theme: studioTheme(), page: "boards",
		path: filepath.Join(t.TempDir(), "library.json"), lib: Library{Provider: codex},
		clicks: make(map[string]*studioButton),
		body:   layout.List{Axis: layout.Vertical}, rail: layout.List{Axis: layout.Vertical},
	}
	var router input.Router
	var ops op.Ops
	now := time.Now()
	frame := func() {
		now = now.Add(20 * time.Millisecond)
		ops.Reset()
		g := ctx{Now: now, Constraints: layout.Exact(image.Pt(750, 580)), Ops: &ops, Source: router.Source()}
		s.layout(g)
		router.Frame(&ops)
	}
	bounds := func(label string) image.Rectangle {
		t.Helper()
		var labels []string
		for _, node := range router.AppendSemantics(nil) {
			if node.Desc.Label != "" {
				labels = append(labels, node.Desc.Label)
			}
			if strings.HasPrefix(node.Desc.Label, label) && !node.Desc.Bounds.Empty() {
				return node.Desc.Bounds
			}
		}
		t.Fatalf("missing control %q in %v", label, labels)
		return image.Rectangle{}
	}
	mouse := func(kind pointer.Kind, buttons pointer.Buttons, at f32.Point) {
		router.Queue(pointer.Event{Kind: kind, Source: pointer.Mouse, Buttons: buttons, Position: at})
		frame()
	}
	clickPoint := func(at f32.Point) {
		mouse(pointer.Press, pointer.ButtonPrimary, at)
		mouse(pointer.Release, 0, at)
		frame()
	}
	click := func(label string) {
		r := bounds(label)
		center := r.Min.Add(r.Size().Div(2))
		clickPoint(f32.Pt(float32(center.X), float32(center.Y)))
	}
	edit := func(label string, end int, next string) {
		click(label)
		router.Queue(key.EditEvent{Range: key.Range{End: end}, Text: next})
		frame()
	}
	load := func(id string) *Board {
		t.Helper()
		lib, err := loadLibrary(s.path)
		if err != nil {
			t.Fatal(err)
		}
		b := lib.board(id)
		if b == nil {
			t.Fatalf("board %q missing from local library", id)
		}
		return b
	}

	frame()
	click("Novo quadro em branco")
	id := s.boardUI.id
	current := s.lib.board(id)
	if s.page != "board" || current == nil || len(current.Nodes) != 0 || len(current.Edges) != 0 {
		t.Fatal("opening an empty board must not run the provider or seed content")
	}
	if empty := load(id); len(empty.Nodes) != 0 || len(empty.Edges) != 0 {
		t.Fatal("blank board should survive reopening")
	}

	click("Renomear quadro")
	edit("Título do quadro", s.boardUI.title.Len(), "Fluxo manual")
	click("Salvar título")
	click("+ Retângulo")
	edit("Título do nó", s.boardUI.label.Len(), "Início")
	edit("Detalhe do nó", 0, "Origem do fluxo")
	click("Salvar nó")
	if len(current.Nodes) != 1 || current.Title != "Fluxo manual" || current.Nodes[0].Label != "Início" {
		t.Fatalf("first node or title did not save: %+v", current)
	}

	nodePoint := func(n BoardNode) f32.Point {
		r := bounds("Área do quadro.")
		c := &s.boardUI.canvas
		p := f32.Pt(float32(n.X)+boardNodeWidth/2, float32(n.Y)+45)
		return f32.Pt(float32(r.Min.X), float32(r.Min.Y)).Add(c.origin).Add(p.Mul(c.scale))
	}
	dragLeft := func(node BoardNode) {
		from := nodePoint(node)
		to := from.Sub(f32.Pt(126, 14))
		mouse(pointer.Press, pointer.ButtonPrimary, from)
		mouse(pointer.Move, pointer.ButtonPrimary, to)
		mouse(pointer.Release, 0, to)
		frame()
	}
	original := current.Nodes[0]
	dragLeft(original)
	if moved := load(id).Nodes[0]; moved.X >= original.X || moved.Y >= original.Y {
		t.Fatalf("drag position did not save: %+v", moved)
	}
	click("Desfazer")
	if got := load(id).Nodes[0]; got != original {
		t.Fatalf("undo did not restore coordinates: %+v", got)
	}
	dragLeft(original)

	click("+ Elipse")
	click("Losango")
	click("Salvar nó")
	click("+ Elipse")
	click("Salvar nó")
	click("+ Texto")
	edit("Título do nó", s.boardUI.label.Len(), "Texto livre")
	click("Salvar nó")
	if len(current.Nodes) != 4 || current.Nodes[1].Shape != "diamond" || current.Nodes[2].Shape != "ellipse" ||
		current.Nodes[3].Shape != "text" || current.Nodes[3].Label != "Texto livre" {
		t.Fatalf("manual elements did not save: %+v", current.Nodes)
	}

	click("Conectar")
	clickPoint(nodePoint(current.Nodes[0]))
	click("Criar conexão")
	want := BoardEdge{From: current.Nodes[3].ID, To: original.ID, Label: "leva a"}
	click("Conexões (1)")
	edit("Rótulo da conexão", 0, want.Label)
	click("Salvar conexão")
	if edges := load(id).Edges; len(edges) != 1 || edges[0] != want {
		t.Fatalf("connection did not save: %+v", edges)
	}
	click("Conexões (1)")
	click("Excluir conexão")
	if len(load(id).Edges) != 0 {
		t.Fatal("connection removal did not save")
	}
	click("Desfazer")
	click("Excluir nó")
	if deleted := load(id); len(deleted.Nodes) != 3 || len(deleted.Edges) != 0 {
		t.Fatalf("node removal left a dangling edge: %+v", deleted)
	}
	click("Desfazer")
	saved := load(id)
	if len(saved.Nodes) != 4 || len(saved.Edges) != 1 || saved.Edges[0] != want || saved.Nodes[0].X >= original.X {
		t.Fatalf("undo did not restore the moved graph: %+v", saved)
	}

	click("Quadros visuais")
	reloaded, err := loadLibrary(s.path)
	if err != nil {
		t.Fatal(err)
	}
	s.lib, s.boardUI = reloaded, boardState{}
	frame()
	click("Abrir quadro Fluxo manual")
	if s.page != "board" || s.boardUI.id != id || len(s.lib.board(id).Nodes) != 4 {
		t.Fatal("reloaded board did not reopen through the library")
	}
	click("Exportar")
	data, err := os.ReadFile(filepath.Join(exportDir, "Quadro-Fluxo-manual.excalidraw"))
	if err != nil {
		t.Fatal(err)
	}
	var scene struct {
		Type     string
		Elements []struct {
			ID, Type, OriginalText, ContainerID string
			StartBinding, EndBinding            struct{ ElementID string }
			BoundElements                       []excalidrawBoundElement
			X, Y                                float64
			Locked                              bool
		}
	}
	if err := json.Unmarshal(data, &scene); err != nil || scene.Type != "excalidraw" {
		t.Fatalf("invalid exported scene: %v", err)
	}
	elements := make(map[string]int)
	for i, e := range scene.Elements {
		if e.Locked {
			t.Fatalf("element %q is locked", e.ID)
		}
		elements[e.ID] = i
	}
	for id, kind := range map[string]string{"node-0": "rectangle", "node-1": "diamond", "node-2": "ellipse", "node-3": "text", "edge-0": "arrow"} {
		if i, ok := elements[id]; !ok || scene.Elements[i].Type != kind {
			t.Fatalf("missing native %s element %s", kind, id)
		}
	}
	start := scene.Elements[elements["node-3"]]
	end := scene.Elements[elements["node-0"]]
	arrow := scene.Elements[elements["edge-0"]]
	caption := scene.Elements[elements["edge-0-text"]]
	if start.ContainerID != "" || start.OriginalText != "Texto livre" || len(start.BoundElements) != 1 ||
		start.BoundElements[0].ID != arrow.ID || end.X != saved.Nodes[0].X || end.Y != saved.Nodes[0].Y ||
		arrow.StartBinding.ElementID != start.ID || arrow.EndBinding.ElementID != end.ID ||
		caption.ID != "edge-0-text" || caption.ContainerID != arrow.ID || caption.OriginalText != want.Label {
		t.Fatalf("export lost an editable binding, label or saved position: %+v", scene.Elements)
	}
}
