package main

import (
	"crypto/rand"
	encodinghex "encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	codex    = "Codex"
	claude   = "Claude Code"
	instant  = "immediate"
	deferred = "review"
)

type Lesson struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Topic           string   `json:"topic"`
	Summary         string   `json:"summary"`
	Cards           []Card   `json:"cards"`
	Sources         []string `json:"sources"`
	GoogleQuestions bool     `json:"google_questions,omitempty"`
}

type Card struct {
	Title          string          `json:"title"`
	Body           string          `json:"body"`
	Question       string          `json:"question"`
	Options        []string        `json:"options"`
	Correct        int             `json:"correct"`
	Explanation    string          `json:"explanation"`
	Hint           string          `json:"hint,omitempty"`
	QuestionOrigin string          `json:"question_origin,omitempty"`
	QuestionSource *QuestionSource `json:"question_source,omitempty"`
}

type QuestionSource struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

const (
	questionOriginal = "original"
	questionAdapted  = "adapted"
	questionCreated  = "created"
)

func (c Card) originLabel() string {
	switch c.QuestionOrigin {
	case questionOriginal:
		return "Questão original encontrada"
	case questionAdapted:
		return "Questão adaptada pela IA"
	case questionCreated:
		return "Questão criada pela IA"
	}
	return ""
}

func (l Lesson) questionSearchNotice() string {
	if l.GoogleQuestions {
		for _, card := range l.Cards {
			if card.QuestionSource != nil {
				return ""
			}
		}
		return "Nenhuma questão com evidência suficiente nos resultados do Google. Os exercícios foram criados pela IA."
	}
	return ""
}

type Progress struct {
	Cursor   int
	Visited  map[int]bool
	Answers  map[int]int
	Reviewed bool
	Notes    map[int]string `json:",omitempty"`
}

type Library struct {
	Generated []Lesson
	Progress  map[string]*Progress
	Provider  string
	Feedback  string
	Folders   []StudyFolder `json:",omitempty"`
	Boards    []Board       `json:",omitempty"`
}

func validateLesson(l Lesson) error {
	if !validText(l.Title, 100) || !validText(l.Topic, 90) || !validText(l.Summary, 400) ||
		len(l.Cards) < 3 || len(l.Cards) > 12 || len(l.Sources) > 8 || l.GoogleQuestions && len(l.Sources) != 0 {
		return errors.New("A lição retornou incompleta ou fora do formato esperado.")
	}
	invalidLink := errors.New("Uma URL apareceu fora dos campos de fonte. Tente de novo.")
	if l.GoogleQuestions && slices.ContainsFunc([]string{l.Title, l.Topic, l.Summary}, explicitStudyLink.MatchString) {
		return invalidLink
	}
	exercises := 0
	maxOptions := 4
	if l.GoogleQuestions {
		maxOptions = 5
	}
	for _, c := range l.Cards {
		if l.GoogleQuestions && (slices.ContainsFunc([]string{c.Title, c.Body, c.Question, c.Explanation, c.Hint}, explicitStudyLink.MatchString) ||
			slices.ContainsFunc(c.Options, explicitStudyLink.MatchString)) {
			return invalidLink
		}
		if !validText(c.Title, 100) {
			return errors.New("Um card retornou sem título válido.")
		}
		if c.Hint != "" && (!utf8.ValidString(c.Hint) || !validText(c.Hint, 400)) {
			return errors.New("Um card retornou com uma dica fora do formato esperado.")
		}
		if c.Question == "" {
			if !validText(c.Body, 1800) || len(c.Options) != 0 || c.QuestionOrigin != "" || c.QuestionSource != nil {
				return errors.New("Um card de leitura retornou fora do formato esperado.")
			}
			continue
		}
		exercises++
		if !validText(c.Question, 600) || utf8.RuneCountInString(c.Body) > 600 ||
			!validText(c.Explanation, 1000) || len(c.Options) < 2 || len(c.Options) > maxOptions ||
			c.Correct < 0 || c.Correct >= len(c.Options) {
			return errors.New("Um exercício retornou fora do formato esperado.")
		}
		for _, option := range c.Options {
			if !validText(option, 280) {
				return errors.New("Um exercício retornou com alternativa inválida.")
			}
		}
		switch c.QuestionOrigin {
		case questionOriginal, questionAdapted:
			source := c.QuestionSource
			if !l.GoogleQuestions || source == nil || utf8.RuneCountInString(source.Title) > 160 || !validQuestionURL(source.URL) {
				return errors.New("Um exercício retornou com fonte inválida.")
			}
		case questionCreated, "":
			if c.QuestionSource != nil || l.GoogleQuestions && c.QuestionOrigin == "" {
				return errors.New("Um exercício retornou com origem inválida.")
			}
		default:
			return errors.New("Um exercício retornou com origem inválida.")
		}
	}
	if exercises == 0 || exercises == len(l.Cards) {
		return errors.New("A lição precisa de leitura e prática.")
	}
	for _, source := range l.Sources {
		if !validText(source, 350) {
			return errors.New("Uma referência retornou fora do formato esperado.")
		}
	}
	return nil
}

func validText(s string, limit int) bool {
	return strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= limit
}

func libraryPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "OpenLogos", "library.json"), nil
}

func loadLibrary(path string) (Library, error) {
	l := Library{Provider: codex, Feedback: instant, Progress: make(map[string]*Progress)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return l, err
	}
	var saved Library
	if err := json.Unmarshal(data, &saved); err != nil {
		return l, err
	}
	if saved.Provider == claude {
		l.Provider = claude
	}
	if saved.Feedback == deferred {
		l.Feedback = deferred
	}
	if saved.Progress != nil {
		l.Progress = saved.Progress
	}
	seen := make(map[string]bool)
	for _, item := range saved.Generated {
		if !strings.HasPrefix(item.ID, "ai-") || seen[item.ID] || validateLesson(item) != nil {
			return l, errors.New("biblioteca local inválida")
		}
		seen[item.ID] = true
	}
	l.Generated = saved.Generated
	if err := l.restoreFolders(saved.Folders); err != nil {
		return l, err
	}
	boards := make(map[string]bool, len(saved.Boards))
	for _, board := range saved.Boards {
		if !strings.HasPrefix(board.ID, "board-") || len(board.ID) > 64 || boards[board.ID] || validateBoard(board) != nil {
			return l, errors.New("quadros inválidos na biblioteca local")
		}
		boards[board.ID] = true
	}
	l.Boards = saved.Boards
	return l, nil
}

func saveLibrary(path string, l Library) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "library-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(data)
	closeErr := tmp.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp.Name(), path)
}

func (l *Library) progress(lesson Lesson) *Progress {
	p := l.Progress[lesson.ID]
	if p == nil {
		p = &Progress{}
		l.Progress[lesson.ID] = p
	}
	p.Cursor = max(0, min(p.Cursor, len(lesson.Cards)))
	if p.Visited == nil {
		p.Visited = make(map[int]bool)
	}
	if p.Answers == nil {
		p.Answers = make(map[int]int)
	}
	return p
}

func (l *Library) add(lesson Lesson) error {
	if err := validateLesson(lesson); err != nil {
		return err
	}
	var id [12]byte
	if _, err := rand.Read(id[:]); err != nil {
		return fmt.Errorf("Falha ao criar ID da lição: %w", err)
	}
	lesson.ID = "ai-" + encodinghex.EncodeToString(id[:])
	l.Generated = append([]Lesson{lesson}, l.Generated...)
	return nil
}
