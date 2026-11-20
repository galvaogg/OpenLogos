package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"gioui.org/font"
)

type excalidrawElement map[string]any

type excalidrawBoundElement struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func boardExcalidraw(b Board) ([]byte, error) {
	if err := validateBoard(b); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	element := func(id, kind string, x, y, width, height float64, stroke, fill string) excalidrawElement {
		return excalidrawElement{
			"id": id, "type": kind, "x": x, "y": y, "width": width, "height": height,
			"strokeColor": stroke, "backgroundColor": fill, "fillStyle": "solid",
			"strokeWidth": 2, "strokeStyle": "solid", "roundness": nil,
			"opacity": 100, "roughness": 0, "angle": 0, "seed": 1, "version": 1, "versionNonce": 0,
			"index": nil, "groupIds": []string{}, "frameId": nil, "boundElements": nil,
			"link": nil, "isDeleted": false, "locked": false, "updated": now,
		}
	}
	text := func(id, original, container string, block boardTextBlock, x, y float64) excalidrawElement {
		el := element(id, "text", x, y, float64(block.width), float64(block.height), "#172039", "transparent")
		el["fontSize"], el["fontFamily"], el["lineHeight"] = block.fontSize, 2, float64(block.lineHeight)/float64(block.fontSize)
		el["text"], el["originalText"], el["autoResize"] = block.wrapped, original, container != ""
		el["textAlign"], el["verticalAlign"], el["containerId"] = "left", "top", nil
		if container != "" {
			el["textAlign"], el["verticalAlign"], el["containerId"] = "center", "middle", container
		}
		return el
	}
	nodes := make(map[string]int, len(b.Nodes))
	cards := measureBoardNodes(b.Nodes)
	rects := make([]boardRect, len(b.Nodes))
	shapes := make([]excalidrawElement, len(b.Nodes))
	labels := make([]excalidrawElement, len(b.Nodes))
	bindings := make([][]excalidrawBoundElement, len(b.Nodes))
	for i, node := range b.Nodes {
		nodes[node.ID] = i
		card := cards[i]
		rects[i] = card.rect(boardNodePosition(node))
		id := fmt.Sprintf("node-%d", i)
		stroke, fill := "#475569", "#f1f5f9"
		switch node.Shape {
		case "topic":
			stroke, fill = "#5142a0", "#e9e5ff"
		case "decision":
			stroke, fill = "#875414", "#fff2c9"
		case "note":
			stroke, fill = "#23695f", "#dbf4ef"
		}
		kind := node.excalidrawType()
		if kind == "text" {
			shapes[i] = text(id, boardNodeText(node), "", card.exported, node.X, node.Y)
			continue
		}
		shape := element(id, kind, node.X, node.Y, float64(card.width), float64(card.height), stroke, fill)
		if kind == "rectangle" {
			shape["roundness"] = map[string]int{"type": 3}
		}
		label := text(id+"-text", boardNodeText(node), id, card.exported,
			node.X+float64(card.width-float32(card.exported.width))/2, node.Y+float64(card.height-float32(card.exported.height))/2)
		shapes[i], labels[i] = shape, label
		bindings[i] = []excalidrawBoundElement{{ID: id + "-text", Type: "text"}}
	}
	shaper := newBoardShaper()
	links, bounds := layoutBoardEdges(shaper, b, rects)
	elements := make([]excalidrawElement, 0, len(b.Nodes)*2+len(b.Edges)*2+1)
	for i, edge := range b.Edges {
		from, to := nodes[edge.From], nodes[edge.To]
		id := fmt.Sprintf("edge-%d", i)
		startID, endID := shapes[from]["id"].(string), shapes[to]["id"].(string)
		link := links[i]
		points := make([][2]float64, len(link.points))
		x, y := float64(link.points[0].X), float64(link.points[0].Y)
		minDX, minDY, maxDX, maxDY := 0.0, 0.0, 0.0, 0.0
		for i, p := range link.points {
			dx, dy := float64(p.X)-x, float64(p.Y)-y
			points[i] = [2]float64{dx, dy}
			minDX, minDY = min(minDX, dx), min(minDY, dy)
			maxDX, maxDY = max(maxDX, dx), max(maxDY, dy)
		}
		arrow := element(id, "arrow", x, y, maxDX-minDX, maxDY-minDY, "#46546c", "transparent")
		arrow["points"], arrow["elbowed"] = points, false
		arrow["startArrowhead"], arrow["endArrowhead"] = nil, "triangle"
		arrow["startBinding"] = excalidrawArrowBinding(startID, link.start)
		arrow["endBinding"] = excalidrawArrowBinding(endID, link.end)
		bindings[from] = append(bindings[from], excalidrawBoundElement{ID: id, Type: "arrow"})
		bindings[to] = append(bindings[to], excalidrawBoundElement{ID: id, Type: "arrow"})
		elements = append(elements, arrow)
		if link.label.wrapped != "" {
			container := ""
			if link.labelBound {
				container = id
				arrow["boundElements"] = []excalidrawBoundElement{{ID: id + "-text", Type: "text"}}
			}
			label := text(id+"-text", edge.Label, container, link.label,
				float64(link.labelBox.min.X)+boardLabelPadX, float64(link.labelBox.min.Y)+boardLabelPadY)
			if container == "" {
				group := []string{id + "-group"}
				arrow["groupIds"], label["groupIds"], label["textAlign"] = group, group, "center"
			}
			elements = append(elements, label)
		}
	}
	for i := range shapes {
		shapes[i]["boundElements"] = bindings[i]
		elements = append(elements, shapes[i])
		if labels[i] != nil {
			elements = append(elements, labels[i])
		}
	}
	if len(b.Nodes) > 0 {
		title := measureBoardText(shaper, b.Title, max(boardNodeWidth, int(bounds.size().X)), 22, 28, font.Normal)
		elements = append(elements, text("board-title", b.Title, "", title, float64(bounds.min.X), float64(bounds.min.Y)-float64(title.height)-24))
	}
	return json.MarshalIndent(map[string]any{
		"type": "excalidraw", "version": 2, "source": "https://github.com/galvaogg/OpenLogos",
		"elements": elements, "appState": map[string]string{"viewBackgroundColor": "#fbfaf8"},
		"files": map[string]any{},
	}, "", "  ")
}

func excalidrawArrowBinding(id string, point [2]float64) map[string]any {
	return map[string]any{
		"elementId": id, "fixedPoint": point, "mode": "orbit",
		"gap": 5, "focus": 0,
	}
}

func exportBoardExcalidraw(dir string, b Board) (string, error) {
	data, err := boardExcalidraw(b)
	if err != nil {
		return "", err
	}
	title := strings.Join(strings.FieldsFunc(b.Title, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), "-")
	stem := "Quadro"
	if chars := []rune(title); len(chars) > 0 {
		stem += "-" + string(chars[:min(60, len(chars))])
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", fmt.Errorf("Não foi possível abrir a pasta Estudos: %w", err)
	}
	defer root.Close()
	name := stem + ".excalidraw"
	for copy := 2; ; copy++ {
		file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if errors.Is(err, os.ErrExist) {
			name = fmt.Sprintf("%s-%d.excalidraw", stem, copy)
			continue
		}
		if err != nil {
			return "", fmt.Errorf("Não foi possível criar o arquivo Excalidraw: %w", err)
		}
		_, writeErr := file.Write(data)
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return "", fmt.Errorf("Não foi possível salvar o arquivo Excalidraw: %w", errors.Join(err, root.Remove(name)))
		}
		return filepath.Join(dir, name), nil
	}
}
