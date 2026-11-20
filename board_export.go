package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
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
	text := func(id, content, container string, x, y, width, fontSize float64) excalidrawElement {
		const lineHeight = 1.25
		wrapped := wrapBoardExportText(content, max(1, int(width/(fontSize*0.55))))
		height := float64(strings.Count(wrapped, "\n")+1) * fontSize * lineHeight
		el := element(id, "text", x, y, width, height, "#172039", "transparent")
		el["fontSize"], el["fontFamily"], el["lineHeight"] = fontSize, 2, lineHeight
		el["text"], el["originalText"], el["autoResize"] = wrapped, content, container != ""
		el["textAlign"], el["verticalAlign"], el["containerId"] = "left", "top", nil
		if container != "" {
			el["textAlign"], el["verticalAlign"], el["containerId"] = "center", "middle", container
		}
		return el
	}
	nodes := make(map[string]int, len(b.Nodes))
	shapes := make([]excalidrawElement, len(b.Nodes))
	labels := make([]excalidrawElement, len(b.Nodes))
	bindings := make([][]excalidrawBoundElement, len(b.Nodes))
	minX, minY := 0.0, 0.0
	for i, node := range b.Nodes {
		nodes[node.ID] = i
		if i == 0 {
			minX, minY = node.X, node.Y
		} else {
			minX, minY = min(minX, node.X), min(minY, node.Y)
		}
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
		labelText := node.Label
		if node.Detail != "" {
			labelText += "\n" + node.Detail
		}
		kind := node.excalidrawType()
		if kind == "text" {
			label := text(id, labelText, "", node.X, node.Y, boardNodeWidth, 17)
			label["y"] = node.Y + (boardNodeHeight-label["height"].(float64))/2
			shapes[i] = label
			continue
		}
		shape := element(id, kind, node.X, node.Y, boardNodeWidth, boardNodeHeight, stroke, fill)
		if kind == "rectangle" {
			shape["roundness"] = map[string]int{"type": 3}
		}
		label := text(id+"-text", labelText, id, node.X+10, node.Y+6, boardNodeWidth-20, 14)
		label["y"] = node.Y + (boardNodeHeight-label["height"].(float64))/2
		shapes[i], labels[i] = shape, label
		bindings[i] = []excalidrawBoundElement{{ID: id + "-text", Type: "text"}}
	}
	elements := make([]excalidrawElement, 0, len(b.Nodes)*2+len(b.Edges)*2+1)
	for i, edge := range b.Edges {
		from, to := nodes[edge.From], nodes[edge.To]
		id := fmt.Sprintf("edge-%d", i)
		startID, endID := shapes[from]["id"].(string), shapes[to]["id"].(string)
		points, start, end := boardArrowGeometry(b.Nodes[from], b.Nodes[to])
		x, y := points[0][0], points[0][1]
		minDX, minDY, maxDX, maxDY := 0.0, 0.0, 0.0, 0.0
		for i := range points {
			dx, dy := points[i][0]-x, points[i][1]-y
			points[i] = [2]float64{dx, dy}
			minDX, minDY = min(minDX, dx), min(minDY, dy)
			maxDX, maxDY = max(maxDX, dx), max(maxDY, dy)
		}
		arrow := element(id, "arrow", x, y, maxDX-minDX, maxDY-minDY, "#46546c", "transparent")
		arrow["points"], arrow["elbowed"] = points, false
		arrow["startArrowhead"], arrow["endArrowhead"] = nil, "triangle"
		arrow["startBinding"] = excalidrawArrowBinding(startID, start)
		arrow["endBinding"] = excalidrawArrowBinding(endID, end)
		bindings[from] = append(bindings[from], excalidrawBoundElement{ID: id, Type: "arrow"})
		bindings[to] = append(bindings[to], excalidrawBoundElement{ID: id, Type: "arrow"})
		elements = append(elements, arrow)
		if edge.Label != "" {
			midX := x + (minDX+maxDX)/2
			midY := y + (minDY+maxDY)/2
			width := min(92, max(20, float64(len([]rune(edge.Label)))*7.2))
			label := text(id+"-text", edge.Label, id, midX-width/2, midY, width, 13)
			label["y"] = midY - label["height"].(float64)/2
			label["labelPosition"] = 0.5
			arrow["boundElements"] = []excalidrawBoundElement{{ID: id + "-text", Type: "text"}}
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
		titleWidth := max(float64(boardNodeWidth), float64(len([]rune(b.Title)))*11)
		elements = append(elements, text("board-title", b.Title, "", minX, minY-50, titleWidth, 22))
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
		"gap": 6, "focus": 0,
	}
}

func boardArrowGeometry(from, to BoardNode) (points [][2]float64, start, end [2]float64) {
	const halfW, halfH = boardNodeWidth / 2.0, boardNodeHeight / 2.0
	dx, dy := to.X-from.X, to.Y-from.Y
	distance := math.Hypot(dx, dy)
	scale := max(math.Abs(dx)/halfW, math.Abs(dy)/halfH)
	if scale <= 2 || distance*(1-2/scale) <= 12 {
		// Take coincident and closely overlapping connectors around the outside edge.
		farX := max(from.X, to.X) + boardNodeWidth + 30
		fromY, toY := from.Y+halfH*0.60, to.Y+halfH*1.40
		if math.Abs(fromY-toY) < 8 {
			toY = to.Y + halfH*0.25
		}
		return [][2]float64{{from.X + boardNodeWidth + 6, fromY}, {farX, fromY}, {farX, toY}, {to.X + boardNodeWidth + 6, toY}},
			[2]float64{1, (fromY - from.Y) / boardNodeHeight}, [2]float64{1, (toY - to.Y) / boardNodeHeight}
	}
	offsetX, offsetY := dx/scale, dy/scale
	gapX, gapY := 6*dx/distance, 6*dy/distance
	start = [2]float64{0.5 + offsetX/boardNodeWidth, 0.5 + offsetY/boardNodeHeight}
	end = [2]float64{1 - start[0], 1 - start[1]}
	return [][2]float64{
		{from.X + halfW + offsetX + gapX, from.Y + halfH + offsetY + gapY},
		{to.X + halfW - offsetX - gapX, to.Y + halfH - offsetY - gapY},
	}, start, end
}

func wrapBoardExportText(content string, width int) string {
	var lines []string
	for _, paragraph := range strings.Split(content, "\n") {
		line := []rune(paragraph)
		for len(line) > width {
			end := width
			for end > 0 && !unicode.IsSpace(line[end]) {
				end--
			}
			if end == 0 {
				end = width
			}
			lines = append(lines, strings.TrimRightFunc(string(line[:end]), unicode.IsSpace))
			line = []rune(strings.TrimLeftFunc(string(line[end:]), unicode.IsSpace))
		}
		lines = append(lines, string(line))
	}
	return strings.Join(lines, "\n")
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
