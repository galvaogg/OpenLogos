package main

import (
	"crypto/rand"
	"errors"
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	boardNodeWidth  = 216
	boardNodeHeight = 100
	boardStepX      = boardNodeWidth + 105
	boardStepY      = boardNodeHeight + 43
)

type Board struct {
	ID    string      `json:"id,omitempty"`
	Title string      `json:"title"`
	Kind  string      `json:"kind"`
	Nodes []BoardNode `json:"nodes"`
	Edges []BoardEdge `json:"edges"`
}

type BoardNode struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Detail string  `json:"detail"`
	Shape  string  `json:"shape"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
}

type BoardEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

func boardKindName(kind string) string {
	switch kind {
	case "mindmap":
		return "Mapa mental"
	case "flowchart":
		return "Fluxograma"
	case "timeline":
		return "Linha do tempo"
	}
	return ""
}

func boardText(value string, limit int, optional bool) bool {
	return (optional && value == "" || validText(value, limit)) &&
		utf8.ValidString(value) && strings.IndexFunc(value, unicode.IsControl) < 0
}

func validateBoard(b Board) error {
	if boardKindName(b.Kind) == "" || !boardText(b.Title, 90, false) ||
		len(b.Nodes) < 2 || len(b.Nodes) > 24 || len(b.Edges) < 1 || len(b.Edges) > 42 {
		return errors.New("O quadro precisa de título, 2 a 24 nós e conexões válidas.")
	}
	nodes := make(map[string]bool, len(b.Nodes))
	for _, n := range b.Nodes {
		if len(n.ID) == 0 || len(n.ID) > 36 || strings.IndexFunc(n.ID, func(r rune) bool {
			return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-')
		}) >= 0 || nodes[n.ID] || !boardText(n.Label, 58, false) || !boardText(n.Detail, 100, true) ||
			!(math.Abs(n.X) <= 10000 && math.Abs(n.Y) <= 10000) {
			return errors.New("O quadro contém um nó inválido ou repetido.")
		}
		switch n.Shape {
		case "topic", "step", "note", "decision":
		default:
			return errors.New("O quadro contém um tipo de nó desconhecido.")
		}
		nodes[n.ID] = true
	}
	edges := make(map[[2]string]bool, len(b.Edges))
	for _, e := range b.Edges {
		pair := [2]string{e.From, e.To}
		if !nodes[e.From] || !nodes[e.To] || e.From == e.To || edges[pair] || !boardText(e.Label, 35, true) {
			return errors.New("O quadro contém uma conexão inválida ou repetida.")
		}
		edges[pair] = true
	}
	return nil
}

func (b Board) clone() Board {
	b.Nodes = slices.Clone(b.Nodes)
	b.Edges = slices.Clone(b.Edges)
	return b
}

func (l *Library) board(id string) *Board {
	if i := slices.IndexFunc(l.Boards, func(b Board) bool { return b.ID == id }); i >= 0 {
		return &l.Boards[i]
	}
	return nil
}

func (l *Library) addBoard(b Board) string {
	b.ID = "board-" + strings.ToLower(rand.Text())
	l.Boards = append([]Board{b}, l.Boards...)
	return b.ID
}

// arrangeBoard uses semantic connections for placement and keeps the positions the learner already curated.
func arrangeBoard(b *Board, previous *Board) {
	columns := boardColumns(*b)
	counts, rows := map[int]int{}, map[int]int{}
	for _, col := range columns {
		counts[col]++
	}
	positions := make(map[string]BoardNode)
	if previous != nil {
		for _, n := range previous.Nodes {
			positions[n.ID] = n
		}
	}
	var placed []BoardNode
	for _, n := range b.Nodes {
		if old, ok := positions[n.ID]; ok {
			placed = append(placed, old)
		}
	}
	for i := range b.Nodes {
		n := &b.Nodes[i]
		col := columns[n.ID]
		row := rows[col]
		rows[col]++
		if old, ok := positions[n.ID]; ok {
			n.X, n.Y = old.X, old.Y
			continue
		}
		n.X = float64(col * boardStepX)
		n.Y = (float64(row) - float64(counts[col]-1)/2) * boardStepY
		if b.Kind == "timeline" {
			count := min(4, len(b.Nodes)-i/4*4)
			col := i % 4
			if i/4%2 == 1 {
				col = count - 1 - col
			}
			n.X, n.Y = float64(col*boardStepX), float64(i/4*boardStepY)
		}
		for slices.ContainsFunc(placed, func(p BoardNode) bool {
			return math.Abs(p.X-n.X) < boardNodeWidth+12 && math.Abs(p.Y-n.Y) < boardNodeHeight+12
		}) {
			n.Y += boardStepY
		}
		placed = append(placed, *n)
	}
}

func boardColumns(b Board) map[string]int {
	adjacent := make(map[string][]string)
	incoming := make(map[string]int)
	for _, edge := range b.Edges {
		adjacent[edge.From] = append(adjacent[edge.From], edge.To)
		incoming[edge.To]++
		if b.Kind == "mindmap" {
			adjacent[edge.To] = append(adjacent[edge.To], edge.From)
		}
	}
	columns := make(map[string]int, len(b.Nodes))
	root := b.Nodes[0].ID
	if b.Kind == "mindmap" {
		for _, n := range b.Nodes {
			if n.Shape == "topic" {
				root = n.ID
				break
			}
		}
	} else {
		for _, n := range b.Nodes {
			if incoming[n.ID] == 0 {
				root = n.ID
				break
			}
		}
	}
	queue := []string{root}
	columns[root] = 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, child := range adjacent[id] {
			if _, ok := columns[child]; ok {
				continue
			}
			col := columns[id]
			if col < 0 || b.Kind == "mindmap" && id == root && len(queue)%2 == 1 {
				columns[child] = col - 1
			} else {
				columns[child] = col + 1
			}
			queue = append(queue, child)
		}
	}
	for _, n := range b.Nodes {
		if _, ok := columns[n.ID]; !ok {
			columns[n.ID] = 0
		}
	}
	return columns
}

func exampleBoard() Board {
	b := Board{
		Title: "Como nasce uma investigação",
		Kind:  "mindmap",
		Nodes: []BoardNode{
			{ID: "pergunta", Label: "Pergunta de pesquisa", Detail: "O que queremos explicar?", Shape: "topic"},
			{ID: "observacao", Label: "Observação", Detail: "Registre o que chamou sua atenção.", Shape: "note", X: -340, Y: -100},
			{ID: "hipotese", Label: "Hipótese", Detail: "Proponha uma explicação que possa ser testada.", Shape: "step", X: 340, Y: 25},
			{ID: "evidencia", Label: "Evidências", Detail: "Compare registros, exemplos e medições.", Shape: "note", Y: -192},
			{ID: "teste", Label: "Teste", Detail: "Mude uma variável de cada vez.", Shape: "step", X: 340, Y: 180},
			{ID: "revisao", Label: "Revisão", Detail: "Refaça a pergunta à luz do que descobriu.", Shape: "note", X: -340, Y: 105},
			{ID: "interpretacao", Label: "Interpretação", Detail: "Procure tendências e explicações alternativas.", Shape: "step", X: 340, Y: -192},
		},
		Edges: []BoardEdge{
			{From: "pergunta", To: "observacao", Label: "parte de"},
			{From: "pergunta", To: "hipotese", Label: "sugere"},
			{From: "pergunta", To: "evidencia", Label: "busca"},
			{From: "hipotese", To: "teste", Label: "orienta"},
			{From: "evidencia", To: "interpretacao", Label: "sustenta"},
			{From: "observacao", To: "revisao", Label: "refina"},
		},
	}
	return b
}
